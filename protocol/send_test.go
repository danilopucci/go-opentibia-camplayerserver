package protocol

import (
	"bytes"
	"go-opentibia-camplayerserver/packet"
	"net"
	"testing"
	"time"
)

type MockConn struct {
	writtenData []byte
	err         error
}

func (m *MockConn) Write(data []byte) (int, error) {
	m.writtenData = data
	if m.err != nil {
		return 0, m.err
	}
	return len(data), nil
}

func (m *MockConn) Read(b []byte) (int, error)         { return 0, nil }
func (m *MockConn) Close() error                       { return nil }
func (m *MockConn) LocalAddr() net.Addr                { return nil }
func (m *MockConn) RemoteAddr() net.Addr               { return nil }
func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestSendRawData(t *testing.T) {
	rawData := []byte{0x01, 0x02, 0x03}
	xteaKey := [4]uint32{0x1, 0x2, 0x3, 0x4}
	mockConn := &MockConn{}

	SendRawData(mockConn, xteaKey, &rawData)

	expectedData := []byte{0x08, 0x00, 0x5c, 0xb8, 0x3e, 0x2c, 0xc8, 0x1f, 0x36, 0x7d}

	if !bytes.Equal(mockConn.writtenData, expectedData) {
		t.Errorf("Expected %v, but got %v", expectedData, mockConn.writtenData)
	}
}

func TestSendClientError(t *testing.T) {
	xteaKey := [4]uint32{0x1, 0x2, 0x3, 0x4}
	errorData := "Test error"
	mockConn := &MockConn{}

	SendClientError(mockConn, xteaKey, errorData)

	expectedData := []byte{0x10, 0x00, 0x5d, 0x2b, 0x35, 0x14, 0x6f, 0xe5, 0x65, 0x81, 0x1d, 0x7c, 0x20, 0x7f, 0x3f, 0xdd, 0x13, 0x5e}

	if !bytes.Equal(mockConn.writtenData, expectedData) {
		t.Errorf("Expected %v, but got %v", expectedData, mockConn.writtenData)
	}
}

func TestSendTextMessage(t *testing.T) {
	xteaKey := [4]uint32{0x1, 0x2, 0x3, 0x4}
	message := "Hello, world!"
	messageType := MessageType(1)
	mockConn := &MockConn{}

	SendTextMessage(mockConn, xteaKey, message, messageType)

	expectedData := []byte{0x18, 0x00, 0x3e, 0xfb, 0x4d, 0x03, 0x48, 0x78, 0xfd, 0x39, 0xf3, 0xdb, 0xf6, 0x42, 0x91, 0x11, 0xf7, 0xf0, 0x54, 0xc0, 0xa2, 0x22, 0x54, 0x6c, 0xc7, 0x39}

	if !bytes.Equal(mockConn.writtenData, expectedData) {
		t.Errorf("Expected %v, but got %v", expectedData, mockConn.writtenData)
	}
}

func TestSendData(t *testing.T) {
	xteaKey := [4]uint32{0x1, 0x2, 0x3, 0x4}
	mockConn := &MockConn{}
	packet := packet.NewOutgoing(10)
	packet.AddUint8(0xFF) // example data

	err := SendData(mockConn, xteaKey, packet)
	if err != nil {
		t.Fatalf("Expected no error, but got %v", err)
	}

	expectedData := []byte{0x08, 0x00, 0x30, 0x60, 0x3f, 0x01, 0xf7, 0x8d, 0x16, 0xd1}

	if !bytes.Equal(mockConn.writtenData, expectedData) {
		t.Errorf("Expected %v, but got %v", expectedData, mockConn.writtenData)
	}
}
