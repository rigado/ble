package smp

import "github.com/go-ble/ble/linux/hci"

type factory struct {}

func NewSmpFactory() *factory {
	return &factory{}
}

func (f *factory) Create(config hci.SmpConfig, bm hci.BondManager) hci.SmpManager {
	return NewSmpManager(config, bm)
}
