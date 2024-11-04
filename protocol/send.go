package protocol

import (
	"fmt"
	"go-opentibia-camplayerserver/packet"
	"net"
)

const DEFAULT_PACKET_SIZE = 1024

type MessageType uint8

const (
	MESSAGE_STATUS_CONSOLE_YELLOW MessageType = 0x01
	MESSAGE_STATUS_CONSOLE_LBLUE  MessageType = 0x04
	MESSAGE_STATUS_CONSOLE_ORANGE MessageType = 0x11
	MESSAGE_STATUS_WARNING        MessageType = 0x12 //Red message in game window and in the console
	MESSAGE_EVENT_ADVANCE         MessageType = 0x13 //White message in game window and in the console
	MESSAGE_EVENT_DEFAULT         MessageType = 0x14 //White message at the bottom of the game window and in the console
	MESSAGE_STATUS_DEFAULT        MessageType = 0x15 //White message at the bottom of the game window and in the console
	MESSAGE_INFO_DESCR            MessageType = 0x16 //Green message in game window and in the console
	MESSAGE_STATUS_SMALL          MessageType = 0x17 //White message at the bottom of the game window"
	MESSAGE_STATUS_CONSOLE_BLUE   MessageType = 0x18
	MESSAGE_STATUS_CONSOLE_RED    MessageType = 0x19
)

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

func SendTextMessage(conn net.Conn, xteaKey [4]uint32, message string, messageType MessageType) {
	packet := packet.NewOutgoing(1 + 2 + len(message)) // message type + string length + string
	packet.AddUint8(0xB4)
	packet.AddUint8(uint8(messageType))
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
