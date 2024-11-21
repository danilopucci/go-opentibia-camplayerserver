package cam

import (
	"errors"
	"fmt"
	"go-opentibia-camplayerserver/client"
	"go-opentibia-camplayerserver/protocol"
	"io"
	"math"
	"sync"
	"time"
)

type CamPacket struct {
	Timestamp int64
	Type      string
	Data      []byte
}

var ErrParse = errors.New("parse error")

const (
	MINIMUM_PLAY_SPEED = 0.25
	MAXIMUM_PLAY_SPEED = 64
)

type CamStats struct {
	currentTime float64
	duration    float64
	speed       float64
	date        string
}

func (c *CamStats) Format() string {
	durationStr := "?"
	if c.duration > 0 {
		durationStr = fmt.Sprintf("%.1f", c.duration)
	}

	speedStr := "Paused"
	if c.speed > 0 {
		speedStr = fmt.Sprintf("Speed: %.2fx", c.speed)
	}

	return fmt.Sprintf("%.1f/%s | %s", c.currentTime, durationStr, speedStr)
}

func (c *CamStats) IncreaseSpeed() {
	if c.speed <= 0 {
		c.speed = 1
	} else {
		c.speed = math.Min(c.speed*2, MAXIMUM_PLAY_SPEED)
	}
}

func (c *CamStats) DecreaseSpeed() {
	c.speed = math.Max(c.speed/2, MINIMUM_PLAY_SPEED)
}

func (c *CamStats) Speed(speed float64) {
	c.speed = speed
}

func HandleCamFileStreaming(wg *sync.WaitGroup, c *client.Client, filePath string) {
	defer wg.Done()
	defer c.Conn.Close()

	camFileReader := NewCamFileReader()

	err := camFileReader.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer camFileReader.Close()

	var camStats CamStats
	camStats.speed = 1.0

	lastPacket, err := camFileReader.LastPacket()
	camFileReader.Reset()

	if err == nil {
		camStats.duration = float64(lastPacket.Timestamp) / 1000.0
	}

	previousTimestamp := int64(0)
	nextProcessPacketTimestamp := time.Now()
	nextBeatcountTimestamp := time.Now()

	poolInterval := 5 * time.Millisecond
	beatCountInterval := 100 * time.Millisecond

	welcomeMessageSent := false
	welcomeMessage := "Welcome"

	for {
		select {
		case <-c.CancelCh:
			fmt.Printf("CamServer is shutting down and closing file %s\n", camFileReader.Filename())
			return

		case command := <-c.CommandCh:
			switch command {

			case "speedUp":
				camStats.IncreaseSpeed()

			case "speedDown":
				camStats.DecreaseSpeed()

			case "pause":
				camStats.Speed(0)

			case "logout":
				fmt.Printf("CamServer is shutting down and closing file %s\n", camFileReader.Filename())
				return
			}

			fmt.Printf("received command %s\n", command)
			continue

		default:

			if time.Now().After(nextProcessPacketTimestamp) && camStats.speed > 0 {
				camPacket, err := camFileReader.NextPacket()

				if err != nil {
					if errors.Is(err, io.EOF) {
						fmt.Printf("Finished to play cam file %s, closing Connection in few seconds\n", camFileReader.Filename())
						time.Sleep(5 * time.Second)
						return
					} else if parseErr := new(ParseError); errors.As(err, &parseErr) {
						fmt.Printf("%v", parseErr)
						continue
					}

					fmt.Printf("Unexpected error: %v\n", err)
					return
				}

				if camPacket.Type != "<" {
					continue
				}

				if previousTimestamp != 0 {
					delay := time.Duration(float64(camPacket.Timestamp-previousTimestamp) / camStats.speed)
					nextProcessPacketTimestamp = time.Now().Add(delay * time.Millisecond)
				}
				previousTimestamp = camPacket.Timestamp

				camStats.currentTime = float64(camPacket.Timestamp) / 1000.0

				protocol.SendRawData(c.Conn, c.XteaKey, &camPacket.Data)

			}

			if time.Now().After(nextBeatcountTimestamp) {
				protocol.SendTextMessage(c.Conn, c.XteaKey, camStats.Format(), protocol.MESSAGE_STATUS_SMALL)

				if !welcomeMessageSent {
					protocol.SendTextMessage(c.Conn, c.XteaKey, welcomeMessage, protocol.MESSAGE_STATUS_CONSOLE_BLUE)
					welcomeMessageSent = true
				}

				nextBeatcountTimestamp = time.Now().Add(beatCountInterval)
			}

			time.Sleep(poolInterval)
		}
	}
}
