package connection

import "github.com/rigado/ble/linux/hci"

func (c *Conn) PutPacket(b []byte) {
	c.chInPkt <- b
}

func (c *Conn) Role() uint8 {
	return c.param.Role()
}

func (c *Conn) BufferPool() hci.BufferPool {
	return c.txBuffer
}

func (c *Conn) CloseInputChannel() {
	close(c.chInPkt)
}

func (c *Conn) SetClosed() {
	close(c.chDone)
}
