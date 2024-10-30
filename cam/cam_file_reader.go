package cam

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type ParseError struct {
	Message string
}

func (e *ParseError) Error() string {
	return e.Message
}

type CamFileReader struct {
	file               *os.File
	fileBuffer         []byte
	filePosition       int64
	fileLine           int64
	packetsBucketIndex int
	packetsBucket      []string
}

const (
	FILE_READ_CHUNK_SIZE = 28192
)

func NewCamFileReader() *CamFileReader {
	return &CamFileReader{
		fileBuffer:         make([]byte, FILE_READ_CHUNK_SIZE),
		filePosition:       0,
		fileLine:           0,
		packetsBucketIndex: 0,
	}
}

func (c *CamFileReader) Open(filePath string) error {
	var err error
	c.file, err = os.Open(filePath)

	if err != nil {
		fmt.Println("Error opening file:", err)
		return fmt.Errorf("error while openning the file %s: %w", filePath, err)
	}

	c.fileLine = 1
	return nil
}

func (c *CamFileReader) Close() {
	c.file.Close()
}

func (c *CamFileReader) Filename() string {
	return c.file.Name()
}

func (c *CamFileReader) NextPacket() (CamPacket, error) {
	var camPacket CamPacket

	if c.packetsBucketIndex >= len(c.packetsBucket) {
		c.packetsBucket, _ = c.retrieveLines()
		c.packetsBucketIndex = 0

		if len(c.packetsBucket) == 0 {
			return camPacket, io.EOF
		}
	}

	rawData := c.packetsBucket[c.packetsBucketIndex]
	camPacket, err := parseCamPacket(&rawData)
	c.fileLine += 1
	c.packetsBucketIndex += 1

	if err != nil {
		if parseErr, ok := err.(*ParseError); ok {
			return camPacket, &ParseError{
				fmt.Sprintf("%s on line %d in file %s; data: %s", parseErr.Message, c.fileLine-1, c.file.Name(), rawData),
			}
		}
		return camPacket, err
	}

	return camPacket, nil
}

func (c *CamFileReader) retrieveLines() ([]string, error) {
	bytesRead, err := c.file.ReadAt(c.fileBuffer, c.filePosition)
	var lines []string

	if bytesRead > 0 {
		c.filePosition += int64(bytesRead)

		var lastNewlineOffset int
		for i := 0; i < bytesRead; i++ {
			if c.fileBuffer[i] == '\n' {
				// Process the line between lastNewlineOffset and current newline position
				line := string(c.fileBuffer[lastNewlineOffset : i+1]) // +1 to include the '\n'
				lines = append(lines, line)
				lastNewlineOffset = i + 1
			}
		}

		// If there is an incomplete line at the end, discard the incomplete line at this read (by adjusting filePosition)
		if lastNewlineOffset < bytesRead {
			c.filePosition -= int64(bytesRead - lastNewlineOffset)
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
		return camPacket, &ParseError{"parse error: invalid data format"}
	}

	packetType := fields[0]
	if packetType != "<" && packetType != ">" {
		return camPacket, &ParseError{fmt.Sprintf("parse error: invalid packet type (%s)", packetType)}
	}

	timestamp, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return camPacket, &ParseError{fmt.Sprintf("parse error: invalid timestamp format: %v", err)}
	}

	hexString := fields[2]
	camPacket.Data = make([]byte, len(hexString)/2)
	if _, err := hex.Decode(camPacket.Data, []byte(hexString)); err != nil {
		return camPacket, &ParseError{fmt.Sprintf("parse error: error decoding hex string: %v", err)}
	}

	camPacket.Timestamp = timestamp
	camPacket.Type = packetType

	return camPacket, nil
}
