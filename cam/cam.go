package cam

import (
	"errors"
	"fmt"
	"go-opentibia-camplayerserver/client"
	"go-opentibia-camplayerserver/protocol"
	"io"
	"sync"
	"time"
)

type CamPacket struct {
	Timestamp int64
	Type      string
	Data      []byte
}

var ErrParse = errors.New("parse error")

type CamStats struct {
	currentTime float32
	duration    float32
	speed       float32
	date        string
}

func (c *CamStats) Format() string {
	return fmt.Sprintf("%.1f | Speed: %.2fx", c.currentTime, c.speed)
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

	previousTimestamp := int64(0)
	nextProcessPacketTimestamp := time.Now()
	nextBeatcountTimestamp := time.Now()

	poolInterval := 5 * time.Millisecond

	var camStats CamStats
	camStats.speed = 1.0

	for {
		select {
		case <-c.CancelCh:
			fmt.Printf("CamServer is shutting down and closing file %s\n", camFileReader.Filename())
			return

		case command := <-c.CommandCh:
			switch command {

			case "speedUp":
				camStats.speed *= 2
				if camStats.speed >= 64 {
					camStats.speed = 64
				}
				fmt.Printf("increased speed to %f\n", camStats.speed)

			case "speedDown":
				camStats.speed /= 2
				if camStats.speed <= 0.25 {
					camStats.speed = 0.25
				}
				fmt.Printf("decreased speed to %f\n", camStats.speed)
			}

			fmt.Printf("received command %s\n", command)
			continue

		default:

			if time.Now().After(nextProcessPacketTimestamp) {

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
					delay := time.Duration(float32(camPacket.Timestamp-previousTimestamp) / camStats.speed)
					nextProcessPacketTimestamp = time.Now().Add(delay * time.Millisecond)
				}
				previousTimestamp = camPacket.Timestamp

				camStats.currentTime = float32(camPacket.Timestamp) / 1000.0

				protocol.SendRawData(c.Conn, c.XteaKey, &camPacket.Data)

			}

			if time.Now().After(nextBeatcountTimestamp) {
				protocol.SendTextMessage(c.Conn, c.XteaKey, camStats.Format(), "1")
				nextBeatcountTimestamp = time.Now().Add(100 * time.Millisecond)
			}

			time.Sleep(poolInterval)
		}
	}
}
