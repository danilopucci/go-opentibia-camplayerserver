package packet

import (
	"testing"
	"unsafe"
)

func TestGetUint8(t *testing.T) {
	var packet Incoming
	packet.buffer = []byte{0x12, 0x34}

	sizeBefore := packet.size()
	var want uint8 = 0x12
	var got uint8 = packet.GetUint8()

	if got != want {
		t.Errorf("got %d, wanted %d", got, want)
	}

	sizeAfter := packet.size()
	expectedSize := sizeBefore - int(unsafe.Sizeof(uint8(0)))

	if sizeAfter != expectedSize {
		t.Errorf("expected packet size to be %d, and got %d", expectedSize, sizeAfter)
	}
}

func TestPeekUint8(t *testing.T) {
	var packet Incoming
	packet.buffer = []byte{0x12, 0x34}

	sizeBefore := packet.size()
	var want uint8 = 0x12
	var got uint8 = packet.peekUint8()

	if got != want {
		t.Errorf("got %d, wanted %d", got, want)
	}

	sizeAfter := packet.size()
	expectedSize := sizeBefore

	if sizeAfter != expectedSize {
		t.Errorf("expected packet size to be %d, and got %d", expectedSize, sizeAfter)
	}
}

func TestGetUint16(t *testing.T) {
	var packet Incoming
	packet.buffer = []byte{0x34, 0x12} // Little endian 0x1234

	sizeBefore := packet.size()
	var want uint16 = 0x1234
	var got uint16 = packet.GetUint16()

	if got != want {
		t.Errorf("got %d, wanted %d", got, want)
	}

	sizeAfter := packet.size()
	expectedSize := sizeBefore - int(unsafe.Sizeof(uint16(0)))

	if sizeAfter != expectedSize {
		t.Errorf("expected packet size to be %d, and got %d", expectedSize, sizeAfter)
	}
}

func TestPeekUint16(t *testing.T) {
	var packet Incoming
	packet.buffer = []byte{0x34, 0x12} // Little endian 0x1234

	sizeBefore := packet.size()
	var want uint16 = 0x1234
	var got uint16 = packet.peekUint16()

	if got != want {
		t.Errorf("got %d, wanted %d", got, want)
	}

	sizeAfter := packet.size()
	expectedSize := sizeBefore

	if sizeAfter != expectedSize {
		t.Errorf("expected packet size to be %d, and got %d", expectedSize, sizeAfter)
	}
}

func TestIncomingGetUint32(t *testing.T) {
	var packet Incoming
	packet.buffer = []byte{0x78, 0x56, 0x34, 0x12} // Little endian 0x12345678

	sizeBefore := packet.size()
	var want uint32 = 0x12345678
	var got uint32 = packet.GetUint32()

	if got != want {
		t.Errorf("got %d, wanted %d", got, want)
	}

	sizeAfter := packet.size()
	expectedSize := sizeBefore - int(unsafe.Sizeof(uint32(0)))

	if sizeAfter != expectedSize {
		t.Errorf("expected packet size to be %d, and got %d", expectedSize, sizeAfter)
	}
}

func TestIncomingPeekUint32(t *testing.T) {
	var packet Incoming
	packet.buffer = []byte{0x78, 0x56, 0x34, 0x12} // Little endian 0x12345678

	sizeBefore := packet.size()
	var want uint32 = 0x12345678
	var got uint32 = packet.peekUint32()

	if got != want {
		t.Errorf("got %d, wanted %d", got, want)
	}

	sizeAfter := packet.size()
	expectedSize := sizeBefore

	if sizeAfter != expectedSize {
		t.Errorf("expected packet size to be %d, and got %d", expectedSize, sizeAfter)
	}
}

func TestIncomingGetString(t *testing.T) {
	var packet Incoming
	packet.buffer = append([]byte{0x05, 0x00}, []byte("hello")...) // 0x05 for the string length, "hello" as the string

	sizeBefore := packet.size()
	want := "hello"
	got := packet.GetString()

	if got != want {
		t.Errorf("got %s, wanted %s", got, want)
	}

	sizeAfter := packet.size()
	expectedSize := sizeBefore - int(unsafe.Sizeof(uint16(0))) - len(want)

	if sizeAfter != expectedSize {
		t.Errorf("expected packet size to be %d, and got %d", expectedSize, sizeAfter)
	}
}

func TestIncomingGetMaxStringLength(t *testing.T) {
	var packet Incoming
	str := "this_is_a_very_long_string"
	strLen := uint16(len(str))
	packet.buffer = append([]byte{byte(strLen), 0x00}, []byte(str)...)

	want := str
	got := packet.GetString()

	if got != want {
		t.Errorf("got %s, wanted %s", got, want)
	}

	if packet.size() != 0 {
		t.Errorf("expected buffer to be empty after reading string, but got size %d", packet.size())
	}
}

func TestIncomingBufferOverflow(t *testing.T) {
	var packet Incoming
	packet.buffer = []byte{0x01}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected a panic due to buffer overflow, but got none")
		}
	}()

	packet.GetUint16() // Should panic because there's not enough data
}

func TestIncomingSkipBytes(t *testing.T) {
	var packet Incoming
	packet.buffer = []byte{0x01, 0x02, 0x03, 0x04}

	sizeBefore := packet.size()
	packet.skipBytes(2)

	uint8Data := packet.GetUint8()
	if uint8Data != 0x03 {
		t.Errorf("expected data to be 0x03 after skipping, got 0x%x", uint8Data)
	}

	sizeAfter := packet.size()
	expectedSize := sizeBefore - 3 // skipped 2, read 1

	if sizeAfter != expectedSize {
		t.Errorf("expected packet size to be %d, and got %d", expectedSize, sizeAfter)
	}
}

func TestIncomingSkipTooManyBytesShouldFail(t *testing.T) {
	var packet Incoming
	packet.buffer = []byte{0x01, 0x02, 0x03}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected a panic when skipping too many bytes, but got none")
		}
	}()

	packet.skipBytes(10) // Should panic
}

func TestIncomingEmptyBufferShouldFail(t *testing.T) {
	var packet Incoming
	packet.buffer = []byte{}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected a panic with an empty buffer, but got none")
		}
	}()

	packet.GetUint32() // Should panic due to empty buffer
}

func TestIncomingResizeSmaller(t *testing.T) {
	packet := NewIncoming(10)
	packet.Resize(5)

	if len(packet.buffer) != 5 {
		t.Errorf("expected buffer size to be 5, but got %d", len(packet.buffer))
	}
}

func TestIncomingResizeLargerShouldFail(t *testing.T) {
	packet := NewIncoming(5)
	packet.buffer = []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	// Expect a panic when resizing to a larger value
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected a panic when resizing to a larger value, but no panic occurred")
		}
	}()

	// Attempt to resize to a larger size, which should cause a panic
	packet.Resize(10)
}

func TestIncomingInit(t *testing.T) {
	packet := NewIncoming(10)

	if len(packet.buffer) != 10 {
		t.Errorf("expected buffer size to be 10, but got %d", len(packet.buffer))
	}
}
