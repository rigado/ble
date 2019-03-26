package hci

import (
	"net"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux/adv"
)

func (a *Advertisement) packetsWErr() (*adv.Packet, error) {
	if a.p != nil {
		return a.p, nil
	}

	p1, err := a.dataWErr()
	if err != nil {
		return nil, err
	}

	p2, err := a.scanResponseWErr()
	if err != nil {
		return nil, err
	}

	return adv.NewRawPacket(p1, p2), nil
}

func (a *Advertisement) localNameWErr() (string, error) {
	p, err := a.packetsWErr()
	if err != nil {
		return "", err
	}

	return p.LocalName(), nil
}

func (a *Advertisement) manufacturerDataWErr() ([]byte, error) {
	p, err := a.packetsWErr()
	if err != nil {
		return nil, err
	}
	return p.ManufacturerData(), nil
}

func (a *Advertisement) serviceDataWErr() ([]ble.ServiceData, error) {
	p, err := a.packetsWErr()
	if err != nil {
		return nil, err
	}
	return p.ServiceData(), nil
}

func (a *Advertisement) servicesWErr() ([]ble.UUID, error) {
	p, err := a.packetsWErr()
	if err != nil {
		return nil, err
	}
	return p.UUIDs(), nil
}

func (a *Advertisement) overflowServiceWErr() ([]ble.UUID, error) {
	p, err := a.packetsWErr()
	if err != nil {
		return nil, err
	}
	return p.UUIDs(), nil
}

func (a *Advertisement) txPowerLevelWErr() (int, error) {
	p, err := a.packetsWErr()
	if err != nil {
		return 0, err
	}

	pwr, _ := p.TxPower()
	return pwr, nil
}

func (a *Advertisement) solicitedServiceWErr() ([]ble.UUID, error) {
	p, err := a.packetsWErr()
	if err != nil {
		return nil, err
	}
	return p.ServiceSol(), nil
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

	addr := net.HardwareAddr([]byte{b[5], b[4], b[3], b[2], b[1], b[0]})
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
