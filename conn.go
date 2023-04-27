package ble

import (
	"context"
	"io"
	"time"
)

type EncryptionChangedInfo struct {
	Status  int
	Err     error
	Enabled bool
}

// Conn implements a L2CAP connection.
type Conn interface {
	io.ReadWriteCloser

	// Context returns the context that is used by this Conn.
	Context() context.Context

	// SetContext sets the context that is used by this Conn.
	SetContext(ctx context.Context)

	// LocalAddr returns local device's address.
	LocalAddr() Addr

	// RemoteAddr returns remote device's address.
	RemoteAddr() Addr

	// ReadRSSI returns the remote device's RSSI.
	ReadRSSI() (int8, error)

	// RxMTU returns the ATT_MTU which the local device is capable of accepting.
	RxMTU() int

	// SetRxMTU sets the ATT_MTU which the local device is capable of accepting.
	SetRxMTU(mtu int)

	// TxMTU returns the ATT_MTU which the remote device is capable of accepting.
	TxMTU() int

	// SetTxMTU sets the ATT_MTU which the remote device is capable of accepting.
	SetTxMTU(mtu int)

	// Disconnected returns a receiving channel, which is closed when the connection disconnects.
	Disconnected() <-chan struct{}

	Pair(AuthData, time.Duration) error

	StartEncryption(change chan EncryptionChangedInfo) error

	OpenLECreditBasedConnection(psm uint16) (LECreditBasedConnection, error)
	ConnectionHandle() uint8
}

type LECreditBasedConnection interface {
	Send(bb []byte) error
	Subscribe() (<-chan []byte, error)
	Unsubscribe() error
	Close() error
	Info() LECreditBasedConnectionInfo
}

type LECreditBasedConnectionInfo struct {
	LocalCID, RemoteCID uint16
	MTU, MPS            uint16
}
