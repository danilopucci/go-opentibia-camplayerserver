package client

import "net"

type Client struct {
	Conn      net.Conn
	FileId    string
	CancelCh  <-chan struct{}
	CommandCh chan string // Channel for receiving commands
	XteaKey   [4]uint32
}
