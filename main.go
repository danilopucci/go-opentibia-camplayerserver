package main

import (
	"encoding/hex"
	"fmt"
	"go-opentibia-camplayerserver/config"
	"go-opentibia-camplayerserver/crypt"
	"go-opentibia-camplayerserver/packet"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const INCOMING_PACKET_SIZE = 1024
const chunkSize = 28192

type LoginRequest struct {
	ClientOs          uint16
	ProtocolVersion   uint16
	XteaKey           [4]uint32
	GamemasterFlag    uint8
	AccountNumber     uint32
	Character         string
	Password          string
	OTCv8StringLength uint16
	OTCv8String       string
	OTCv8Version      uint16
	IsValid           bool
}

type Client struct {
	conn      net.Conn
	fileId    string
	cancelCh  <-chan bool
	commandCh chan string // Channel for receiving commands
	XteaKey   [4]uint32
}

func startCamServer(closeCamServerCh <-chan bool, wg *sync.WaitGroup, decrypter *crypt.RSA, cfg *config.Config) {
	defer wg.Done()

	fmt.Printf("Cam server starting to listen to %s:%d\n", cfg.CamServer.HostName, cfg.CamServer.Port)

	tcpListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.CamServer.HostName, cfg.CamServer.Port))
	if err != nil {
		fmt.Println("[startCamServer] - Error starting server:", err)
		return
	}
	defer tcpListener.Close()

	for {
		select {
		case <-closeCamServerCh:
			fmt.Printf("CamServer is shutting down and no longer accepting connections.\n")
			return
		default:

			// timeout to avoid infinite lock at Accept and be able to handle closeCamServer channel
			tcpListener.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second))

			tcpConnection, err := tcpListener.Accept()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				fmt.Println("[startCamServer] - Error accepting connection:", err)
				continue
			}

			loginRequest, err := handleClientLoginRequest(tcpConnection, decrypter, cfg)
			if err != nil {
				fmt.Println("[startCamServer] - Error handling client login request:", err)
			}

			if loginRequest.IsValid {
				fmt.Printf("Request Received: clientOs %d; protocolVersion: %d; accountNumber: %d; character %s; password %s; otcv8: \n\tstrlen %d\n\tstr: %s\n\tversion: %d\n", loginRequest.ClientOs, loginRequest.ProtocolVersion, loginRequest.AccountNumber, loginRequest.Character, loginRequest.Password, loginRequest.OTCv8StringLength, loginRequest.OTCv8String, loginRequest.OTCv8Version)

				client := &Client{
					conn:      tcpConnection,
					fileId:    loginRequest.Character,
					XteaKey:   loginRequest.XteaKey,
					cancelCh:  closeCamServerCh,
					commandCh: make(chan string),
				}

				wg.Add(1)
				go handleCamFileStreaming(wg, client)
				// go handleClientInputPackets()
			}

		}
	}
}

func main() {

	var wg sync.WaitGroup
	stopCh := make(chan bool)

	// Capture SIGINT and SIGTERM for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChan // Block until a shutdown signal is received
		fmt.Println("Shutdown signal received")
		close(stopCh) // Notify server to stop accepting new connections
	}()

	config, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v", err)
	}

	rsaDecrypter, err := crypt.NewRSADecrypter(config.RSAKeyFile)
	if err != nil {
		fmt.Println("Error loading private key:", err)
		os.Exit(1)
	}

	fmt.Println("Starting Cam Server goroutine...")
	wg.Add(1)
	go startCamServer(stopCh, &wg, rsaDecrypter, &config)

	wg.Wait()
	fmt.Println("Server shutdown gracefully")

}

func handleClientLoginRequest(conn net.Conn, decrypter *crypt.RSA, cfg *config.Config) (LoginRequest, error) {
	//defer conn.Close()

	packet := packet.NewIncoming(INCOMING_PACKET_SIZE)
	var request LoginRequest
	request.IsValid = false

	reqLen, err := conn.Read(packet.PeekBuffer())
	if err != nil {
		return request, fmt.Errorf("[handleClient] - error reading: %w", err)
	}
	packet.Resize(reqLen)

	packet.GetUint16() // message size
	packet.GetUint8()

	request.ClientOs = packet.GetUint16()
	request.ProtocolVersion = packet.GetUint16()

	decryptedMsg, err := decrypter.DecryptNoPadding(packet.PeekBuffer())
	if err != nil {
		return request, fmt.Errorf("[parseLogin] - error while decrypting packet: %w", err)
	}

	copy(packet.PeekBuffer(), decryptedMsg)

	if packet.GetUint8() != 0 {
		return request, fmt.Errorf("[parseLogin] - error decrypted packet's first byte is not zero")
	}

	request.XteaKey[0] = packet.GetUint32()
	request.XteaKey[1] = packet.GetUint32()
	request.XteaKey[2] = packet.GetUint32()
	request.XteaKey[3] = packet.GetUint32()

	packet.GetUint8()
	request.AccountNumber = packet.GetUint32()
	request.Character = packet.GetString()
	request.Password = packet.GetString()

	request.OTCv8StringLength = packet.GetUint16()
	if request.OTCv8StringLength == 5 {
		request.OTCv8String = packet.GetStringSlice(int(request.OTCv8StringLength))
		if request.OTCv8String == "OTCv8" {
			request.OTCv8Version = packet.GetUint16()
		}
	}

	// remoteIpAddress, err := utils.GetRemoteIpAddr(conn)
	// if err != nil {
	// 	fmt.Printf("[handleClient] - could not get remote IP address: %s\n", err)
	// 	return
	// }

	///TODO: add validations: empty account number, wrong client version, account locked, ip banned and wrong account id

	request.IsValid = true
	return request, nil
}

func handleCamFileStreaming(wg *sync.WaitGroup, client *Client) {
	defer wg.Done()

	file, err := os.Open("Test_2_25-10-2024-18-36-45.cam")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	chunk := make([]byte, chunkSize)
	var readOffset int64 = 0
	isProcessingFile := true

	for isProcessingFile {
		select {
		case <-client.cancelCh:
			fmt.Printf("CamServer is shutting down and closing file %s\n", file.Name())
			return

		default:
			lines, err := retrieveLines(file, chunk, &readOffset)

			var previousTimestamp int64
			for _, line := range lines {
				select {

				case <-client.cancelCh:
					fmt.Printf("CamServer is shutting down and closing file %s\n", file.Name())
					return

				default:

					err := processAndSendPacket(client, line, &previousTimestamp)
					if err != nil {
						fmt.Printf("error while sending data: %s", err)
					}
				}
			}

			// Break the loop on EOF, but after processing the remaining data
			if err != nil {
				if err == io.EOF {
					fmt.Println("EOF reached")
					///TODO: send logout packet
				} else {
					fmt.Println("Error reading file:", err)
				}
				isProcessingFile = false
			}
		}

	}
}

func retrieveLines(file *os.File, chunk []byte, readOffset *int64) ([][]byte, error) {
	//checar se o arquivo esta aberto

	bytesRead, err := file.ReadAt(chunk, *readOffset)
	var lines [][]byte

	// Process the chunk if any bytes were read
	if bytesRead > 0 {
		*readOffset += int64(bytesRead)

		// Find newlines in the chunk and split by them
		var lastNewline int
		for i := 0; i < bytesRead; i++ {
			if chunk[i] == '\n' {
				// Process the line between lastNewline and current newline position
				line := chunk[lastNewline : i+1] // Include the '\n'
				lines = append(lines, line)
				//fmt.Printf("Processed line: %s", line)
				lastNewline = i + 1
			}
		}

		// If there is an incomplete line at the end, adjust readOffset
		if lastNewline < bytesRead {
			// Adjust the readOffset to re-read the incomplete line in the next cycle
			*readOffset -= int64(bytesRead - lastNewline)
		}
	}

	// Break the loop on EOF, but after processing the remaining data
	if err != nil {
		if err == io.EOF {
			fmt.Println("EOF reached")
		} else {
			fmt.Println("Error reading file:", err)
		}
	}

	return lines, err
}

func processAndSendPacket(client *Client, rawPacket []byte, previousTimestamp *int64) error {

	fields := strings.Fields(string(rawPacket))
	if len(fields) < 3 {
		return fmt.Errorf("invalid data format")
	}

	direction := fields[0]
	if direction != "<" && direction != ">" {
		return fmt.Errorf("invalid packet direction")
	}

	if direction != "<" {
		return nil
	}

	// Parse the timestamp
	timestamp, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %v", err)
	}

	// If previous timestamp exists, calculate delay
	if *previousTimestamp != 0 {
		delay := time.Duration(timestamp - *previousTimestamp)
		time.Sleep(delay * time.Millisecond)
	}
	*previousTimestamp = timestamp // Update previous timestamp

	hexString := fields[2]

	byteData, err := hex.DecodeString(hexString)
	if err != nil {
		return fmt.Errorf("error decoding hex string: %v", err)
	}

	outgoingPacket := packet.NewOutgoing(len(byteData))
	outgoingPacket.AddBytes(byteData)
	outgoingPacket.XteaEncrypt(client.XteaKey)
	outgoingPacket.HeaderAddSize()

	dataToSend := outgoingPacket.Get()

	// Send the byte data over the TCP connection
	_, err = client.conn.Write(dataToSend)
	if err != nil {
		return fmt.Errorf("error sending data over TCP connection: %v", err)
	}

	//fmt.Printf("Sent %d bytes over TCP: %x\n", len(dataToSend), dataToSend)
	return nil
}
