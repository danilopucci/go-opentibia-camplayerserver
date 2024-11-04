package cam

import (
	"compress/gzip"
	"os"
	"strings"
	"testing"
)

// Helper function to create a sample gzip file
func createGzipFile(data string) (*os.File, error) {
	tmpFile, err := os.CreateTemp("", "sample.gz")
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()

	gzWriter := gzip.NewWriter(tmpFile)
	_, err = gzWriter.Write([]byte(data))
	if err != nil {
		return nil, err
	}
	gzWriter.Close()

	return tmpFile, nil
}

func TestNewCamFileReader(t *testing.T) {
	reader := NewCamFileReader()
	if reader == nil {
		t.Errorf("Expected NewCamFileReader to return a non-nil reader")
	}
}

func TestCamFileReader_OpenRegularFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "sample.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write some data
	tmpFile.WriteString("< 1627391000 48656c6c6f0a") // Example data

	reader := NewCamFileReader()
	err = reader.Open(tmpFile.Name())
	if err != nil {
		t.Errorf("Expected no error when opening file, but got %v", err)
	}

	reader.Close()
}

func TestCamFileReader_OpenGzipFile(t *testing.T) {
	tmpFile, err := createGzipFile("< 1627391000 48656c6c6f0a") // Example compressed data
	if err != nil {
		t.Fatalf("Failed to create temp gzip file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	reader := NewCamFileReader()
	err = reader.Open(tmpFile.Name())
	if err != nil {
		t.Errorf("Expected no error when opening gzip file, but got %v", err)
	}

	reader.Close()
}

func TestCamFileReader_NextPacket(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "sample.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write some data
	tmpFile.WriteString("< 1627391000 48656c6c6f\n> 1627391001 776f726c640a") // Example data

	reader := NewCamFileReader()
	err = reader.Open(tmpFile.Name())
	if err != nil {
		t.Errorf("Expected no error when opening file, but got %v", err)
	}

	reader.packetsBucket, _ = reader.retrieveLines() // Load lines into packetsBucket
	reader.packetsBucketIndex = 0

	packet, err := reader.NextPacket()
	if err != nil {
		t.Fatalf("Expected no error from NextPacket, got %v", err)
	}

	expectedTimestamp := int64(1627391000)
	if packet.Timestamp != expectedTimestamp {
		t.Errorf("Expected timestamp %d, but got %d", expectedTimestamp, packet.Timestamp)
	}

	expectedData := "Hello"
	if string(packet.Data) != expectedData {
		t.Errorf("Expected data %s, but got %s", expectedData, packet.Data)
	}
}

func TestParseCamPacket(t *testing.T) {
	tests := []struct {
		input       string
		expectedErr string
	}{
		{"< 1627391000 48656c6c6f", ""},
		{"> 1627391001 776f726c64", ""},
		{"! 1627391001 776f726c64", "parse error: invalid packet type"},
		{"< invalid_timestamp 48656c6c6f", "parse error: invalid timestamp format"},
		{"< 1627391000 invalid_hex", "parse error: error decoding hex string"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseCamPacket(&tt.input)
			if err != nil {
				if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("Expected error containing %s, but got %v", tt.expectedErr, err)
				}
			} else if tt.expectedErr != "" {
				t.Errorf("Expected error %s but got none", tt.expectedErr)
			}
		})
	}
}
