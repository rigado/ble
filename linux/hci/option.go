package hci

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-ble/ble/linux/hci/cmd"
)

// SetDeviceID sets HCI device ID.
func (h *HCI) SetDeviceID(id int) error {
	h.id = id
	return nil
}

// SetDialerTimeout sets dialing timeout for Dialer.
func (h *HCI) SetDialerTimeout(d time.Duration) error {
	h.dialerTmo = d
	return nil
}

// SetListenerTimeout sets dialing timeout for Listener.
func (h *HCI) SetListenerTimeout(d time.Duration) error {
	h.listenerTmo = d
	return nil
}

// SetConnParams overrides default connection parameters.
func (h *HCI) SetConnParams(param cmd.LECreateConnection) error {
	h.params.connParams = param
	return nil
}

func (h *HCI) EnableSecurity(bm interface{}) error {
	bondManager, ok := bm.(BondManager)
	if !ok {
		return fmt.Errorf("unknown bond manager type")
	}
	h.smpEnabled = true
	if h.smp != nil {
		h.smp.SetBondManager(bondManager)
	}
	return nil
}

// SetScanParams overrides default scanning parameters.
func (h *HCI) SetScanParams(param cmd.LESetScanParameters) error {
	h.params.scanParams = param
	return nil
}

// SetAdvParams overrides default advertising parameters.
func (h *HCI) SetAdvParams(param cmd.LESetAdvertisingParameters) error {
	h.params.advParams = param
	return nil
}

// SetPeripheralRole is not supported
func (h *HCI) SetPeripheralRole() error {
	return errors.New("Not supported")
}

// SetCentralRole is not supported
func (h *HCI) SetCentralRole() error {
	return errors.New("Not supported")
}

// SetAdvHandlerSync overrides default advertising handler behavior (async)
func (h *HCI) SetAdvHandlerSync(sync bool) error {
	h.advHandlerSync = sync
	return nil
}

// SetErrorHandler ...
func (h *HCI) SetErrorHandler(handler func(error)) error {
	h.errorHandler = handler
	return nil
}
