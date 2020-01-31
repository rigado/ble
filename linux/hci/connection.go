package hci

import (
	"github.com/rigado/ble"
	"github.com/rigado/ble/linux/hci/evt"
)

type Connection interface {
	ble.Conn
	PutPacket([]byte)
	Role() uint8
	BufferPool() BufferPool
	CloseInputChannel()
	SetClosed()
}

type ConnectionFactory interface {
	Create(Controller, evt.LEConnectionComplete) Connection
}
