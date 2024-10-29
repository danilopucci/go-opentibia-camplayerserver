package cam

import (
	"encoding/hex"
	"fmt"
	"go-opentibia-camplayerserver/client"
	"go-opentibia-camplayerserver/protocol"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const chunkSize = 28192

type CamPacket struct {
	Timestamp int64
	Type      string
	Data      []byte
}

func HandleCamFileStreaming(wg *sync.WaitGroup, c *client.Client, filePath string) {
	defer wg.Done()
	defer c.Conn.Close()

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	chunk := make([]byte, chunkSize)
	readOffset := int64(0)

	var packetsBucket []string
	currentIndexFromPacketsBucket := int(0)
	fileLine := 0
	previousTimestamp := int64(0)
	nextProcessPacketTimestamp := time.Now()

	processPacketSleep := 5 * time.Millisecond

	for {
		select {
		case <-c.CancelCh:
			fmt.Printf("CamServer is shutting down and closing file %s\n", file.Name())
			return

		case command := <-c.CommandCh:
			fmt.Printf("received command %s\n", command)
			continue

		default:

			if time.Now().Before(nextProcessPacketTimestamp) {
				time.Sleep(processPacketSleep)
				continue
			}

			if currentIndexFromPacketsBucket >= len(packetsBucket) {
				packetsBucket, _ = retrieveLines(file, chunk, &readOffset)
				currentIndexFromPacketsBucket = 0

				if len(packetsBucket) == 0 {
					fmt.Printf("Finished to play cam file %s, closing Connection in few seconds\n", file.Name())
					time.Sleep(5 * time.Second)
					return
				}
			}

			camPacket, err := parseCamPacket(&packetsBucket[currentIndexFromPacketsBucket])
			fileLine += 1
			currentIndexFromPacketsBucket += 1

			fmt.Printf("HandleCamFileStreaming - currentPacketIndexFromBucket %d; fileLine %d\n", currentIndexFromPacketsBucket, fileLine)

			if err != nil {
				fmt.Printf("error while parsing CamPacket: %s in file: %s; line: %d; data: %s", err, file.Name(), fileLine, packetsBucket[currentIndexFromPacketsBucket])
				continue
			}

			if camPacket.Type != "<" {
				continue
			}

			if previousTimestamp != 0 {
				delay := time.Duration(camPacket.Timestamp - previousTimestamp)
				nextProcessPacketTimestamp = time.Now().Add(delay * time.Millisecond)
			}
			previousTimestamp = camPacket.Timestamp

			protocol.SendRawData(c.Conn, c.XteaKey, &camPacket.Data)
		}
	}
}

func retrieveLines(file *os.File, chunk []byte, readOffset *int64) ([]string, error) {
	bytesRead, err := file.ReadAt(chunk, *readOffset)
	var lines []string

	if bytesRead > 0 {
		*readOffset += int64(bytesRead)

		var lastNewlineOffset int
		for i := 0; i < bytesRead; i++ {
			if chunk[i] == '\n' {
				// Process the line between lastNewlineOffset and current newline position
				line := string(chunk[lastNewlineOffset : i+1]) // +1 to include the '\n'
				lines = append(lines, line)
				lastNewlineOffset = i + 1
			}
		}

		// If there is an incomplete line at the end, discard the incomplete line at this read (by adjusting readOffset)
		if lastNewlineOffset < bytesRead {
			*readOffset -= int64(bytesRead - lastNewlineOffset)
		}
	}

	if err != nil {
		if err != io.EOF {
			fmt.Println("Error reading file:", err)
		}
	}
	return lines, err
}

func parseCamPacket(rawData *string) (CamPacket, error) {
	var camPacket CamPacket
	fields := strings.Fields(*rawData)

	if len(fields) < 3 {
		return camPacket, fmt.Errorf("invalid data format")
	}

	packetType := fields[0]
	if packetType != "<" && packetType != ">" {
		return camPacket, fmt.Errorf("invalid packet type (%s)", packetType)
	}

	timestamp, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return camPacket, fmt.Errorf("invalid timestamp format: %v", err)
	}

	hexString := fields[2]
	camPacket.Data = make([]byte, len(hexString)/2)
	if _, err := hex.Decode(camPacket.Data, []byte(hexString)); err != nil {
		return camPacket, fmt.Errorf("error decoding hex string: %v", err)
	}

	camPacket.Timestamp = timestamp
	camPacket.Type = packetType

	return camPacket, nil
}
