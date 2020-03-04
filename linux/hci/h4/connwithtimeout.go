package h4

import (
	"net"
	"time"
)

type connWithTimeout struct {
	c       net.Conn
	timeout time.Duration
}

func (cwt *connWithTimeout) Read(b []byte) (int, error) {
	// with deadline
	cwt.c.SetReadDeadline(time.Now().Add(cwt.timeout))
	return cwt.c.Read(b)
}

func (cwt *connWithTimeout) Write(b []byte) (int, error) {
	// with deadline
	cwt.c.SetWriteDeadline(time.Now().Add(cwt.timeout))
	return cwt.c.Write(b)
}

func (cwt *connWithTimeout) Close() error {
	return cwt.c.Close()
}
