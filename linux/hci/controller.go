package hci

import "github.com/rigado/ble"

type Controller interface{
	RequestBufferPool() BufferPool
	RequestSmpManager(SmpConfig) (SmpManager, error)
	DispatchError(error)
	SocketWrite([]byte) (int, error)
	Send(Command, CommandRP) error
	Addr() ble.Addr
}
