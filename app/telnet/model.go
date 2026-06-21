package telnet

import (
	"net"
	"time"
)

const crlf = "\r\n"

type Telnet struct {
	user    string
	pass    string
	conn    net.Conn
	timeout time.Duration
}
