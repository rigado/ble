package hci

import (
	"errors"
	"strings"

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
func (a *Advertisement) packets() *adv.Packet {
	p, _ := a.packetsWErr()
	return p
}

// LocalName returns the LocalName of the remote peripheral.
func (a *Advertisement) LocalName() string {
	v, _ := a.localNameWErr()
	return v
}

// ManufacturerData returns the ManufacturerData of the advertisement.
func (a *Advertisement) ManufacturerData() []byte {
	v, _ := a.manufacturerDataWErr()
	return v
}

// ServiceData returns the service data of the advertisement.
func (a *Advertisement) ServiceData() []ble.ServiceData {
	v, _ := a.serviceDataWErr()
	return v
}

// Services returns the service UUIDs of the advertisement.
func (a *Advertisement) Services() []ble.UUID {
	v, _ := a.servicesWErr()
	return v
}

// OverflowService returns the UUIDs of overflowed service.
func (a *Advertisement) OverflowService() []ble.UUID {
	v, _ := a.overflowServiceWErr()
	return v
}

// TxPowerLevel returns the tx power level of the remote peripheral.
func (a *Advertisement) TxPowerLevel() int {
	v, _ := a.txPowerLevelWErr()
	return v
}

// SolicitedService returns UUIDs of solicited services.
func (a *Advertisement) SolicitedService() []ble.UUID {
	v, _ := a.solicitedServiceWErr()
	return v
}

// Connectable indicates weather the remote peripheral is connectable.
func (a *Advertisement) Connectable() bool {
	v, _ := a.connectableWErr()
	return v
}

// RSSI returns RSSI signal strength.
func (a *Advertisement) RSSI() int {
	v, _ := a.rssiWErr()
	return v
}

// Addr returns the address of the remote peripheral.
func (a *Advertisement) Addr() ble.Addr {
	v, _ := a.addrWErr()
	return v
}

// EventType returns the event type of Advertisement.
// This is linux specific.
func (a *Advertisement) EventType() uint8 {
	v, _ := a.eventTypeWErr()
	return v
}

// AddressType returns the address type of the Advertisement.
// This is linux specific.
func (a *Advertisement) AddressType() uint8 {
	v, _ := a.addressTypeWErr()
	return v
}

// Data returns the advertising data of the packet.
// This is linux specific.
func (a *Advertisement) Data() []byte {
	v, _ := a.dataWErr()
	return v
}

// ScanResponse returns the scan response of the packet, if it presents.
// This is linux specific.
func (a *Advertisement) ScanResponse() []byte {
	v, _ := a.ScanResponseWErr()
	return v
}

func (a *Advertisement) ToMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	keys := ble.AdvertisementMapKeys

	addr, err := a.addrWErr()
	if err != nil {
		return nil, errors.Wrap(err, keys.MAC)
	}
	m[keys.MAC] = strings.Replace(addr.String(), ":", "", -1)

	et, err := a.eventTypeWErr()
	if err != nil {
		return nil, errors.Wrap(err, keys.EventType)
	}
	m[keys.EventType] = et

	c, err := a.connectableWErr()
	if err != nil {
		return nil, errors.Wrap(err, keys.Connectable)
	}
	m[keys.Connectable] = c

	r, err := a.rssiWErr()
	if err != nil {
		return nil, errors.Wrap(err, keys.RSSI)
	}
	if r != 0 {
		m[keys.RSSI] = r
	} else {
		m[keys.RSSI] = -128
	}

	//build the packets and bail before we try picking stuff out
	pp, err := a.packetsWErr()
	if err != nil {
		return nil, errors.Wrap(err, "pdu")
	}

	ln, err := a.localNameWErr()
	if err != nil {
		return nil, errors.Wrap(err, keys.Name)
	}
	if len(ln) != 0 {
		m[keys.Name] = ln
	}

	md, err := a.manufacturerDataWErr()
	if err != nil {
		return nil, errors.Wrap(err, keys.MFG)
	}
	if md != nil {
		m[keys.MFG] = md
	}

	v, err := a.servicesWErr()
	if err != nil {
		return nil, errors.Wrap(err, keys.Services)
	}
	if v != nil {
		m[keys.Services] = v
	}

	sd, err := a.serviceDataWErr()
	if err != nil {
		return nil, errors.Wrap(err, keys.ServiceData)
	}
	if sd != nil {
		m[keys.ServiceData] = sd
	}

	ss, err := a.solicitedServiceWErr()
	if err != nil {
		return nil, errors.Wrap(err, keys.Solicited)
	}
	if ss != nil {
		m[keys.Solicited] = ss
	}

	return m, nil
}
