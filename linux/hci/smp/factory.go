package smp

import "github.com/rigado/ble/linux/hci"

type factory struct {
	bm hci.BondManager
}

func NewSmpFactory(bm hci.BondManager) *factory {
	return &factory{bm}
}

func (f *factory) Create(config hci.SmpConfig) hci.SmpManager {
	return NewSmpManager(config, f.bm)
}

func (f *factory) SetBondManager(bm hci.BondManager) {
	f.bm = bm
}
