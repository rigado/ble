package dev

import (
	"github.com/rigado/ble"
	"github.com/rigado/ble/linux"
)

// DefaultDevice ...
func DefaultDevice(opts ...ble.Option) (d ble.Device, err error) {
	return linux.NewDevice(opts...)
}
