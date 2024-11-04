package protocol

import (
	"go-opentibia-camplayerserver/client"
	"go-opentibia-camplayerserver/packet"
	"testing"
)

// Helper function to create a mock Incoming packet with predefined bytes
func newMockIncoming(data []byte) *packet.Incoming {
	p := packet.NewIncoming(len(data))
	copy(p.PeekBuffer(), data)
	return p
}

// Helper function to create a mock Client
func newMockClient() *client.Client {
	return &client.Client{
		CommandCh: make(chan string, 1), // Buffer to avoid blocking
	}
}

func TestParsePacket(t *testing.T) {
	tests := []struct {
		name      string
		inputData []byte
		expected  string
	}{
		{"Logout Packet", []byte{0x14}, "logout"},
		{"Speed Up Packet", []byte{0x6F}, "speedUp"},
		{"Move Forward Packet", []byte{0x70}, "moveFoward"},
		{"Speed Down Packet", []byte{0x71}, "speedDown"},
		{"Move Backward Packet", []byte{0x72}, "moveBackward"},
		{"Say Packet", []byte{0x96, TALKTYPE_SAY, 0x02, 0x00, 'H', 'i'}, "talk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := newMockClient()
			mockPacket := newMockIncoming(tt.inputData)

			// Run the ParsePacket function
			ParsePacket(mockClient, mockPacket)

			// Check if the expected command was sent to CommandCh
			select {
			case cmd := <-mockClient.CommandCh:
				if cmd != tt.expected {
					t.Errorf("expected command %s, got %s", tt.expected, cmd)
				}
			default:
				t.Errorf("expected command %s but no command was sent", tt.expected)

			}
		})
	}
}

func TestParseSay(t *testing.T) {
	tests := []struct {
		name         string
		inputData    []byte
		expectedText string
	}{
		{"Private Say", []byte{TALKTYPE_PRIVATE, 0x04, 0x00, 'J', 'o', 'h', 'n', 0x02, 0x00, 'H', 'i'}, "Hi"},
		{"Channel Yell", []byte{TALKTYPE_CHANNEL_Y, 0x00, 0x00, 0x02, 0x00, 'H', 'i'}, "Hi"},
		{"Monster Say", []byte{TALKTYPE_MONSTER_SAY, 0x02, 0x00, 'H', 'i'}, "Hi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPacket := newMockIncoming(tt.inputData)

			result := ParseSay(mockPacket)
			if result != tt.expectedText {
				t.Errorf("expected Say result %s, got %s", tt.expectedText, result)
			}
		})
	}
}
