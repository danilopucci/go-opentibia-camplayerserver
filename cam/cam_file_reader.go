package cam

import (
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	gzipReader         *gzip.Reader
	fileBuffer         []byte
	fileLine           int64
	packetsBucketIndex int
	packetsBucket      []string
}

const (
	// each cam file line has the following digits: 1 control, 2 spaces, up to 19 of timestamp, up to 131070 digits (65355 packet bytes)
	// 256KB is a good number, despite being an oversized number, it is still a small amount of memory
	FILE_READ_CHUNK_SIZE = 262144 // 256 KB
)

func NewCamFileReader() *CamFileReader {
	return &CamFileReader{
		fileBuffer:         make([]byte, 0, FILE_READ_CHUNK_SIZE),
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

	extension := filepath.Ext(filePath)
	fmt.Printf("%s\n", extension)
	if extension == ".gz" {
		c.gzipReader, err = gzip.NewReader(c.file)
		if err != nil {
			return fmt.Errorf("error creating gzip reader: %w", err)
		}
	}

	c.fileLine = 1
	return nil
}

func (c *CamFileReader) Close() {
	if c.gzipReader != nil {
		c.gzipReader.Close()
	}
	if c.file != nil {
		c.file.Close()
	}
}

func (c *CamFileReader) Reset() error {
	// Reset the file pointer to the beginning
	if _, err := c.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to start of file: %w", err)
	}

	// Reset the gzip reader if the file is gzip-compressed
	if c.gzipReader != nil {
		if err := c.gzipReader.Reset(c.file); err != nil {
			return fmt.Errorf("error resetting gzip reader: %w", err)
		}
	}

	// Reset fields to their initial state
	c.fileBuffer = make([]byte, 0, FILE_READ_CHUNK_SIZE)
	c.fileLine = 1
	c.packetsBucketIndex = 0
	c.packetsBucket = nil

	return nil
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

func (c *CamFileReader) LastPacket() (CamPacket, error) {
	var lastLine string

	for {
		lines, err := c.retrieveLines()
		if len(lines) > 0 {
			lastLine = lines[len(lines)-1] // Keep the last line of each read
		}

		if err == io.EOF {
			// End of file, return the last line as a packet
			if lastLine == "" {
				return CamPacket{}, io.EOF
			}
			return parseCamPacket(&lastLine)
		}

		if err != nil {
			return CamPacket{}, fmt.Errorf("error reading file: %w", err)
		}
	}
}

func (c *CamFileReader) retrieveLines() ([]string, error) {
	var bytesRead int
	var err error
	var lines []string
	rawData := make([]byte, FILE_READ_CHUNK_SIZE)

	if c.gzipReader != nil {
		bytesRead, err = c.gzipReader.Read(rawData)
	} else {
		bytesRead, err = c.file.Read(rawData)
	}

	if bytesRead > 0 {
		if len(c.fileBuffer) == 0 {
			c.fileBuffer = rawData[:bytesRead]
		} else {
			// Use append when fileBuffer already has data (e.g., leftover from previous read)
			c.fileBuffer = append(c.fileBuffer, rawData[:bytesRead]...)
		}

		var lastNewlineOffset int
		for i := 0; i < len(c.fileBuffer); i++ {
			if c.fileBuffer[i] == '\n' {
				line := string(c.fileBuffer[lastNewlineOffset : i+1]) // +1 to include the '\n'
				lines = append(lines, line)
				lastNewlineOffset = i + 1
			}
		}

		// Keep only the incomplete line (if any) in fileBuffer
		if lastNewlineOffset < len(c.fileBuffer) {
			c.fileBuffer = c.fileBuffer[lastNewlineOffset:]
		} else {
			c.fileBuffer = c.fileBuffer[:0]
		}
	}

	if err == io.EOF && len(c.fileBuffer) > 0 {
		lines = append(lines, string(c.fileBuffer))
		c.fileBuffer = c.fileBuffer[:0]
	}

	if err != nil && err != io.EOF {
		fmt.Println("Error reading file:", err)
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
