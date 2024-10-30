package protocol

import (
	"fmt"
	"go-opentibia-camplayerserver/packet"
	"net"
)

const DEFAULT_PACKET_SIZE = 1024

func SendRawData(conn net.Conn, xteaKey [4]uint32, rawData *[]byte) {
	packet := packet.NewOutgoing(len(*rawData))
	packet.AddBytes(*rawData)

	SendData(conn, xteaKey, packet)
}

func SendClientError(conn net.Conn, xteaKey [4]uint32, errorData string) {
	packet := packet.NewOutgoing(DEFAULT_PACKET_SIZE)
	packet.AddUint8(0x0A)
	packet.AddString(errorData)

	SendData(conn, xteaKey, packet)
}

func SendTextMessage(conn net.Conn, xteaKey [4]uint32, message string, messageType string) {
	packet := packet.NewOutgoing(1 + 2 + len(message)) // message type + string length + string
	packet.AddUint8(0xB4)
	packet.AddUint8(0x17)
	packet.AddString(message)

	SendData(conn, xteaKey, packet)
}

func SendData(conn net.Conn, xteaKey [4]uint32, packet *packet.Outgoing) error {
	packet.XteaEncrypt(xteaKey)
	packet.HeaderAddSize()

	dataToSend := packet.Get()

	_, err := conn.Write(dataToSend)
	if err != nil {
		return fmt.Errorf("failed to send data: %v", err)
	}

	return nil
}
