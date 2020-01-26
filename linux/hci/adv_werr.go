package hci

import (
	"fmt"
	"net"

	"github.com/rigado/ble"
)

func (a *Advertisement) localNameWErr() (string, error) {
	if a.p == nil {
		return "", fmt.Errorf("nil packet")
	}
	return a.p.LocalName(), nil
}

func (a *Advertisement) manufacturerDataWErr() ([]byte, error) {
	if a.p == nil {
		return nil, fmt.Errorf("nil packet")
	}
	return a.p.ManufacturerData(), nil
}

func (a *Advertisement) serviceDataWErr() ([]ble.ServiceData, error) {
	if a.p == nil {
		return nil, fmt.Errorf("nil packet")
	}
	return a.p.ServiceData(), nil
}

func (a *Advertisement) servicesWErr() ([]ble.UUID, error) {
	if a.p == nil {
		return nil, fmt.Errorf("nil packet")
	}
	return a.p.UUIDs(), nil
}

func (a *Advertisement) overflowServiceWErr() ([]ble.UUID, error) {
	if a.p == nil {
		return nil, fmt.Errorf("nil packet")
	}
	return a.p.UUIDs(), nil
}

func (a *Advertisement) txPowerLevelWErr() (int, error) {
	if a.p == nil {
		return 0, fmt.Errorf("nil packet")
	}

	pwr, _ := a.p.TxPower()
	return pwr, nil
}

func (a *Advertisement) solicitedServiceWErr() ([]ble.UUID, error) {
	if a.p == nil {
		return nil, fmt.Errorf("nil packet")
	}
	return a.p.ServiceSol(), nil
}

func (a *Advertisement) connectableWErr() (bool, error) {
	t, err := a.eventTypeWErr()
	if err != nil {
		return false, err
	}

	c := (t == evtTypAdvDirectInd) || (t == evtTypAdvInd)
	return c, nil
}

func (a *Advertisement) rssiWErr() (int, error) {
	r, err := a.e.RSSIWErr(a.i)
	return int(r), err
}

func (a *Advertisement) addrWErr() (ble.Addr, error) {
	b, err := a.e.AddressWErr(a.i)
	if err != nil {
		return nil, err
	}

	addr := ble.NewAddr(
		net.HardwareAddr([]byte{b[5], b[4], b[3], b[2], b[1], b[0]}).String())
	at, err := a.e.AddressTypeWErr(a.i)
	if err != nil {
		return nil, err
	}
	if at == 1 {
		return RandomAddress{addr}, nil
	}
	return addr, nil
}

func (a *Advertisement) eventTypeWErr() (uint8, error) {
	return a.e.EventTypeWErr(a.i)
}

func (a *Advertisement) addressTypeWErr() (uint8, error) {
	return a.e.AddressTypeWErr(a.i)
}

func (a *Advertisement) dataWErr() ([]byte, error) {
	return a.e.DataWErr(a.i)
}

func (a *Advertisement) scanResponseWErr() ([]byte, error) {
	if a.sr == nil {
		return nil, nil
	}
	return a.sr.dataWErr()
}
