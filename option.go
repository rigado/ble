package ble

import (
	"time"

	"github.com/rigado/ble/linux/hci/cmd"
)

// DeviceOption is an interface which the device should implement to allow using configuration options
type DeviceOption interface {
	SetDialerTimeout(time.Duration) error
	SetListenerTimeout(time.Duration) error
	SetConnParams(cmd.LECreateConnection) error
	SetScanParams(cmd.LESetScanParameters) error
	SetAdvParams(cmd.LESetAdvertisingParameters) error
	SetPeripheralRole() error
	SetCentralRole() error
	SetAdvHandlerSync(bool) error
	SetErrorHandler(handler func(error)) error
	EnableSecurity(interface{}) error

	SetTransportHCISocket(id int) error
	SetTransportH4Socket(addr string, timeout time.Duration) error
	SetTransportH4Uart(path string) error
}

// An Option is a configuration function, which configures the device.
type Option func(DeviceOption) error

// DEPRECATED: legacy stuff
func OptDeviceID(id int) Option {
	return OptTransportHCISocket(id)
}

// OptDialerTimeout sets dialing timeout for Dialer.
func OptDialerTimeout(d time.Duration) Option {
	return func(opt DeviceOption) error {
		opt.SetDialerTimeout(d)
		return nil
	}
}

// OptListenerTimeout sets dialing timeout for Listener.
func OptListenerTimeout(d time.Duration) Option {
	return func(opt DeviceOption) error {
		opt.SetListenerTimeout(d)
		return nil
	}
}

// OptConnParams overrides default connection parameters.
func OptConnParams(param cmd.LECreateConnection) Option {
	return func(opt DeviceOption) error {
		opt.SetConnParams(param)
		return nil
	}
}

// OptScanParams overrides default scanning parameters.
func OptScanParams(param cmd.LESetScanParameters) Option {
	return func(opt DeviceOption) error {
		opt.SetScanParams(param)
		return nil
	}
}

// OptAdvParams overrides default advertising parameters.
func OptAdvParams(param cmd.LESetAdvertisingParameters) Option {
	return func(opt DeviceOption) error {
		opt.SetAdvParams(param)
		return nil
	}
}

// OptPeripheralRole configures the device to perform Peripheral tasks.
func OptPeripheralRole() Option {
	return func(opt DeviceOption) error {
		opt.SetPeripheralRole()
		return nil
	}
}

// OptCentralRole configures the device to perform Central tasks.
func OptCentralRole() Option {
	return func(opt DeviceOption) error {
		opt.SetCentralRole()
		return nil
	}
}

// OptAdvHandlerSync sets sync adv handling
func OptAdvHandlerSync(sync bool) Option {
	return func(opt DeviceOption) error {
		opt.SetAdvHandlerSync(sync)
		return nil
	}
}

// OptErrorHandler sets error handler
func OptErrorHandler(handler func(error)) Option {
	return func(opt DeviceOption) error {
		opt.SetErrorHandler(handler)
		return nil
	}
}

// OptEnableSecurity enables bonding with devices
func OptEnableSecurity(bondManager interface{}) Option {
	return func(opt DeviceOption) error {
		opt.EnableSecurity(bondManager)
		return nil
	}
}

// OptTransportHCISocket set hci socket transport
func OptTransportHCISocket(id int) Option {
	return func(opt DeviceOption) error {
		opt.SetTransportHCISocket(id)
		return nil
	}
}

// OptTransportH4Socket set h4 socket transport
func OptTransportH4Socket(addr string, timeout time.Duration) Option {
	return func(opt DeviceOption) error {
		opt.SetTransportH4Socket(addr, timeout)
		return nil
	}
}

// OptTransportH4Uart set h4 uart transport
func OptTransportH4Uart(path string) Option {
	return func(opt DeviceOption) error {
		opt.SetTransportH4Uart(path)
		return nil
	}
}
