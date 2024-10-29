package main

import (
	"encoding/hex"
	"fmt"
	"go-opentibia-camplayerserver/protocol"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CamPacket struct {
	Timestamp int64
	Type      string
	Data      []byte
}

func HandleCamFileStreaming(wg *sync.WaitGroup, client *Client, filePath string) {
	defer wg.Done()
	defer client.conn.Close()

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	chunk := make([]byte, chunkSize)
	readOffset := int64(0)
	isProcessingFile := true

	var lines []string
	pendingLinesToProcess := int(0)
	currentProcessingLine := int(0)
	//var retrieveLinesError error
	fileLine := 0
	previousTimestamp := int64(0)

	for isProcessingFile {
		select {
		case <-client.cancelCh:
			fmt.Printf("CamServer is shutting down and closing file %s\n", file.Name())
			return

		case command := <-client.commandCh:
			fmt.Printf("received command %s\n", command)
			continue

		default:
			if pendingLinesToProcess <= 0 {
				lines, _ = retrieveLines(file, chunk, &readOffset)
				currentProcessingLine = 0
				pendingLinesToProcess = len(lines) - currentProcessingLine

				if pendingLinesToProcess == 0 {
					fmt.Printf("Finished to play cam file %s, closing connection in few seconds\n", file.Name())
					time.Sleep(5 * time.Second)
					isProcessingFile = false
					continue
				}
			}

			camPacket, err := parseCamPacket(&lines[currentProcessingLine])
			fileLine += 1
			currentProcessingLine += 1
			pendingLinesToProcess -= 1

			fmt.Printf("HandleCamFileStreaming - pendingLinesToProcess %d; currentProcessingLine %d; pendingLinesToProcess %d, fileLine %d\n", pendingLinesToProcess, currentProcessingLine, pendingLinesToProcess, fileLine)

			if err != nil {
				fmt.Printf("error while parsing CamPacket: %s in file: %s; line: %d; data: %s", err, file.Name(), fileLine, lines[currentProcessingLine])
				continue
			}

			if camPacket.Type != "<" {
				continue
			}

			if previousTimestamp != 0 {
				delay := time.Duration(camPacket.Timestamp - previousTimestamp)
				time.Sleep(delay * time.Millisecond)
			}
			previousTimestamp = camPacket.Timestamp

			protocol.SendRawData(client.conn, client.XteaKey, &camPacket.Data)

			// if retrieveLinesError != nil {
			// 	if retrieveLinesError == io.EOF && len(lines) == 0 {
			// 		fmt.Printf("Finished to play cam file %s, closing connection in few seconds\n", file.Name())
			// 		time.Sleep(5 * time.Second)
			// 	} else {
			// 		fmt.Printf("Error reading file %s: %s\n", file.Name(), retrieveLinesError)
			// 	}
			// 	isProcessingFile = false
			// }
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
