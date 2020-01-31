package connection

import (
	"github.com/rigado/ble/linux/hci"
	"github.com/rigado/ble/linux/hci/evt"
)

type factory struct {}

func (f factory) Create(h hci.Controller, e evt.LEConnectionComplete) hci.Connection {
	return New(h, e)
}

func NewFactory() hci.ConnectionFactory {
	return &factory{}
}
