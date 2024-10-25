package packet

import (
	"encoding/binary"
)

type Incoming struct {
	buffer   []byte
	position int
}

func NewIncoming(size int) *Incoming {
	return &Incoming{
		buffer:   make([]byte, size),
		position: 0,
	}
}

func (p *Incoming) size() int {
	return len(p.buffer[p.position:])
}

func (p *Incoming) Resize(size int) {
	p.buffer = p.buffer[:size]
}

func (p *Incoming) skipBytes(n int) {
	if p.position+n > p.size() {
		panic("skipping more bytes than size")
	}
	p.position += n
}

func (p *Incoming) GetUint8() uint8 {
	result := p.buffer[p.position]
	p.position += 1
	return result
}

func (p *Incoming) peekUint8() uint8 {
	return p.buffer[p.position]
}

func (p *Incoming) GetUint16() uint16 {
	result := binary.LittleEndian.Uint16(p.buffer[p.position:])
	p.position += 2
	return result
}

func (p *Incoming) peekUint16() uint16 {
	return binary.LittleEndian.Uint16(p.buffer[p.position:])
}

func (p *Incoming) GetUint32() uint32 {
	result := binary.LittleEndian.Uint32(p.buffer[p.position:])
	p.position += 4
	return result
}

func (p *Incoming) peekUint32() uint32 {
	return binary.LittleEndian.Uint32(p.buffer[p.position:])
}

func (p *Incoming) GetString() string {
	stringLength := int(p.GetUint16())
	result := string(p.buffer[p.position:(p.position + stringLength)])
	p.position += stringLength
	return result
}

func (p *Incoming) GetStringSlice(length int) string {
	result := string(p.buffer[p.position:(p.position + length)])
	p.position += length
	return result
}

func (p *Incoming) PeekBuffer() []byte {
	return p.buffer[p.position:]
}
