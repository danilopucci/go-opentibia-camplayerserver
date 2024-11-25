package main

import (
	"encoding/binary"
	"fmt"
	"go-opentibia-camplayerserver/cam"
	"go-opentibia-camplayerserver/client"
	"go-opentibia-camplayerserver/config"
	"go-opentibia-camplayerserver/crypt"
	"go-opentibia-camplayerserver/packet"
	"go-opentibia-camplayerserver/protocol"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

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

func startCamServer(closeCamServerCh <-chan struct{}, wg *sync.WaitGroup, decrypter *crypt.RSA, cfg *config.Config) {
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

			if tcpConn, ok := tcpConnection.(*net.TCPConn); ok {
				tcpConn.SetNoDelay(true)
			}

			loginRequest, err := handleClientLoginRequest(tcpConnection, decrypter, cfg)
			if err != nil {
				fmt.Println("[startCamServer] - Error handling client login request:", err)
			}

			if loginRequest.IsValid {
				fmt.Printf("Request Received: clientOs %d; protocolVersion: %d; accountNumber: %d; character %s; password %s; otcv8: \n\tstrlen %d\n\tstr: %s\n\tversion: %d\n", loginRequest.ClientOs, loginRequest.ProtocolVersion, loginRequest.AccountNumber, loginRequest.Character, loginRequest.Password, loginRequest.OTCv8StringLength, loginRequest.OTCv8String, loginRequest.OTCv8Version)

				client := &client.Client{
					Conn:      tcpConnection,
					FileId:    loginRequest.Character,
					XteaKey:   loginRequest.XteaKey,
					CancelCh:  closeCamServerCh,
					CommandCh: make(chan string),
				}

				wg.Add(1)
				go cam.HandleCamFileStreaming(wg, client, "Test_2_25-10-2024-18-36-45.cam")
				wg.Add(1)
				go handleClientInputPackets(wg, client)
			}

		}
	}
}

func main() {

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

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

	const INCOMING_PACKET_SIZE = 65535
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

func handleClientInputPackets(wg *sync.WaitGroup, c *client.Client) {
	defer wg.Done()

	for {
		select {
		case <-c.CancelCh:
			fmt.Println("Client disconnected or cancelled command")
			return

		default:
			c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second))

			header := make([]byte, packet.HEADER_LENGTH)
			if _, err := io.ReadFull(c.Conn, header); err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				return
			}

			// parse header: it has only the packet length
			packetLength := int(binary.LittleEndian.Uint16(header))

			packet := packet.NewIncoming(packetLength)
			if _, err := io.ReadFull(c.Conn, packet.PeekBuffer()); err != nil {
				continue
			}

			if err := packet.XteaDecrypt(c.XteaKey); err != nil {
				fmt.Printf("Error during XteaDecrypt: %v\n", err)
				continue
			}

			protocol.ParsePacket(c, packet)
		}
	}
}
