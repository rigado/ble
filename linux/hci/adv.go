package hci

import (
	"net"
	"strings"

	errors "github.com/pkg/errors"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux/adv"
	"github.com/go-ble/ble/linux/hci/evt"
)

// RandomAddress is a Random Device Address.
type RandomAddress struct {
	ble.Addr
}

// [Vol 6, Part B, 4.4.2] [Vol 3, Part C, 11]
const (
	evtTypAdvInd        = 0x00 // Connectable undirected advertising (ADV_IND).
	evtTypAdvDirectInd  = 0x01 // Connectable directed advertising (ADV_DIRECT_IND).
	evtTypAdvScanInd    = 0x02 // Scannable undirected advertising (ADV_SCAN_IND).
	evtTypAdvNonconnInd = 0x03 // Non connectable undirected advertising (ADV_NONCONN_IND).
	evtTypScanRsp       = 0x04 // Scan Response (SCAN_RSP).
)

func newAdvertisement(e evt.LEAdvertisingReport, i int) *Advertisement {
	return &Advertisement{e: e, i: i}
}

// Advertisement implements ble.Advertisement and other functions that are only
// available on Linux.
type Advertisement struct {
	e  evt.LEAdvertisingReport
	i  int
	sr *Advertisement

	// cached packets.
	p *adv.Packet
}

// setScanResponse ssociate sca response to the existing advertisement.
func (a *Advertisement) setScanResponse(sr *Advertisement) {
	a.sr = sr
	a.p = nil // clear the cached.
}

// packets returns the combined advertising packet and scan response (if presents)
func (a *Advertisement) packets() (*adv.Packet, error) {
	if a.p != nil {
		return a.p, nil
	}

	p1, err := a.Data()
	if err != nil {
		return nil, err
	}

	p2, err := a.ScanResponse()
	if err != nil {
		return nil, err
	}

	return adv.NewRawPacket(p1, p2), nil
}

// LocalName returns the LocalName of the remote peripheral.
func (a *Advertisement) LocalName() (string, error) {
	p, err := a.packets()
	if err != nil {
		return "", err
	}

	return p.LocalName(), nil
}

// ManufacturerData returns the ManufacturerData of the advertisement.
func (a *Advertisement) ManufacturerData() ([]byte, error) {
	p, err := a.packets()
	if err != nil {
		return nil, err
	}
	return p.ManufacturerData(), nil
}

// ServiceData returns the service data of the advertisement.
func (a *Advertisement) ServiceData() ([]ble.ServiceData, error) {
	p, err := a.packets()
	if err != nil {
		return nil, err
	}
	return p.ServiceData(), nil
}

// Services returns the service UUIDs of the advertisement.
func (a *Advertisement) Services() ([]ble.UUID, error) {
	p, err := a.packets()
	if err != nil {
		return nil, err
	}
	return p.UUIDs(), nil
}

// OverflowService returns the UUIDs of overflowed service.
func (a *Advertisement) OverflowService() ([]ble.UUID, error) {
	p, err := a.packets()
	if err != nil {
		return nil, err
	}
	return p.UUIDs(), nil
}

// TxPowerLevel returns the tx power level of the remote peripheral.
func (a *Advertisement) TxPowerLevel() (int, error) {
	p, err := a.packets()
	if err != nil {
		return 0, err
	}

	pwr, _ := p.TxPower()
	return pwr, nil
}

// SolicitedService returns UUIDs of solicited services.
func (a *Advertisement) SolicitedService() ([]ble.UUID, error) {
	p, err := a.packets()
	if err != nil {
		return nil, err
	}
	return p.ServiceSol(), nil
}

// Connectable indicates weather the remote peripheral is connectable.
func (a *Advertisement) Connectable() (bool, error) {
	t, err := a.EventType()
	if err != nil {
		return false, err
	}

	c := (t == evtTypAdvDirectInd) || (t == evtTypAdvInd)
	return c, nil
}

// RSSI returns RSSI signal strength.
func (a *Advertisement) RSSI() (int, error) {
	r, err := a.e.RSSI(a.i)
	return int(r), err
}

// Addr returns the address of the remote peripheral.
func (a *Advertisement) Addr() (ble.Addr, error) {
	b, err := a.e.Address(a.i)
	if err != nil {
		return nil, err
	}

	addr := net.HardwareAddr([]byte{b[5], b[4], b[3], b[2], b[1], b[0]})
	at, err := a.e.AddressType(a.i)
	if err != nil {
		return nil, err
	}
	if at == 1 {
		return RandomAddress{addr}, nil
	}
	return addr, nil
}

// EventType returns the event type of Advertisement.
// This is linux specific.
func (a *Advertisement) EventType() (uint8, error) {
	return a.e.EventType(a.i)
}

// AddressType returns the address type of the Advertisement.
// This is linux specific.
func (a *Advertisement) AddressType() (uint8, error) {
	return a.e.AddressType(a.i)
}

// Data returns the advertising data of the packet.
// This is linux specific.
func (a *Advertisement) Data() ([]byte, error) {
	return a.e.Data(a.i)
}

// ScanResponse returns the scan response of the packet, if it presents.
// This is linux specific.
func (a *Advertisement) ScanResponse() ([]byte, error) {
	if a.sr == nil {
		return nil, nil
	}
	return a.sr.Data()
}

func (a *Advertisement) ToMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	keys := ble.AdvertisementMapKeys

	addr, err := a.Addr()
	if err != nil {
		return nil, errors.Wrap(err, keys.MAC)
	}
	m[keys.MAC] = strings.Replace(addr.String(), ":", "", -1)

	et, err := a.EventType()
	if err != nil {
		return nil, errors.Wrap(err, keys.EventType)
	}
	m[keys.EventType] = et

	c, err := a.Connectable()
	if err != nil {
		return nil, errors.Wrap(err, keys.Connectable)
	}
	m[keys.Connectable] = c

	r, err := a.RSSI()
	if err != nil {
		return nil, errors.Wrap(err, keys.RSSI)
	}
	if r != 0 {
		m[keys.RSSI] = r
	} else {
		m[keys.RSSI] = -128
	}

	//build the packets and bail before we try picking stuff out
	pp, err := a.packets()
	if err != nil {
		return nil, errors.Wrap(err, "pdu")
	}

	ln, err := a.LocalName()
	if err != nil {
		return nil, errors.Wrap(err, keys.Name)
	}
	if len(ln) != 0 {
		m[keys.Name] = ln
	}

	md, err := a.ManufacturerData()
	if err != nil {
		return nil, errors.Wrap(err, keys.MFG)
	}
	if md != nil {
		m[keys.MFG] = md
	}

	v, err := a.Services()
	if err != nil {
		return nil, errors.Wrap(err, keys.Services)
	}
	if v != nil {
		m[keys.Services] = v
	}

	sd, err := a.ServiceData()
	if err != nil {
		return nil, errors.Wrap(err, keys.ServiceData)
	}
	if sd != nil {
		m[keys.ServiceData] = sd
	}

	ss, err := a.SolicitedService()
	if err != nil {
		return nil, errors.Wrap(err, keys.Solicited)
	}
	if ss != nil {
		m[keys.Solicited] = ss
	}

	return m, nil
}
