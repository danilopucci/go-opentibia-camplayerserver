package protocol

import (
	"go-opentibia-camplayerserver/client"
	"go-opentibia-camplayerserver/packet"
	"strings"
)

type SpeakClass = uint8

const (
	TALKTYPE_SAY          SpeakClass = 1
	TALKTYPE_WHISPER      SpeakClass = 2
	TALKTYPE_YELL         SpeakClass = 3
	TALKTYPE_PRIVATE      SpeakClass = 4
	TALKTYPE_CHANNEL_Y    SpeakClass = 5 // Yellow
	TALKTYPE_RVR_CHANNEL  SpeakClass = 6
	TALKTYPE_RVR_ANSWER   SpeakClass = 7
	TALKTYPE_RVR_CONTINUE SpeakClass = 8
	TALKTYPE_BROADCAST    SpeakClass = 9
	TALKTYPE_CHANNEL_R1   SpeakClass = 10 // Red - #c text
	TALKTYPE_PRIVATE_RED  SpeakClass = 11 // @name@text
	TALKTYPE_CHANNEL_O    SpeakClass = 12 // orange
	TALKTYPE_CHANNEL_R2   SpeakClass = 14 // red anonymous - #d text
	TALKTYPE_MONSTER_YELL SpeakClass = 16
	TALKTYPE_MONSTER_SAY  SpeakClass = 17
)

func ParsePacket(c *client.Client, packet *packet.Incoming) {

	opCode := packet.GetUint8()

	switch opCode {
	case 0x14:
		c.CommandCh <- "logout"
		return
	case 0x6F:
		c.CommandCh <- "speedUp"
		return
	case 0x70:
		c.CommandCh <- "moveFoward"
		return
	case 0x71:
		c.CommandCh <- "speedDown"
		return
	case 0x72:
		c.CommandCh <- "moveBackward"
		return

	case 0x96:
		message := strings.ToLower(ParseSay(packet))

		if len(message) > 0 && message[0] == '/' {
			switch message {

			case "/pause":
				c.CommandCh <- "pause"
				return

			case "/stop":
				c.CommandCh <- "stop"
				return
			}
		}

		c.CommandCh <- "talk"
		return
	}
}

func ParseSay(packet *packet.Incoming) string {

	speakClass := packet.GetUint8()

	switch speakClass {

	case TALKTYPE_PRIVATE, TALKTYPE_PRIVATE_RED, TALKTYPE_RVR_ANSWER:
		packet.GetString()

	case TALKTYPE_CHANNEL_Y, TALKTYPE_CHANNEL_R1, TALKTYPE_CHANNEL_R2:
		packet.GetUint16()
	default:
	}

	return packet.GetString()
}
