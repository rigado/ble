package hci

import (
	"net"
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
	if a.p != nil {
		return a.p
	}
	return adv.NewRawPacket(a.Data(), a.ScanResponse())
}

// LocalName returns the LocalName of the remote peripheral.
func (a *Advertisement) LocalName() string {
	return a.packets().LocalName()
}

// ManufacturerData returns the ManufacturerData of the advertisement.
func (a *Advertisement) ManufacturerData() []byte {
	return a.packets().ManufacturerData()
}

// ServiceData returns the service data of the advertisement.
func (a *Advertisement) ServiceData() []ble.ServiceData {
	return a.packets().ServiceData()
}

// Services returns the service UUIDs of the advertisement.
func (a *Advertisement) Services() []ble.UUID {
	return a.packets().UUIDs()
}

// OverflowService returns the UUIDs of overflowed service.
func (a *Advertisement) OverflowService() []ble.UUID {
	return a.packets().UUIDs()
}

// TxPowerLevel returns the tx power level of the remote peripheral.
func (a *Advertisement) TxPowerLevel() int {
	pwr, _ := a.packets().TxPower()
	return pwr
}

// SolicitedService returns UUIDs of solicited services.
func (a *Advertisement) SolicitedService() []ble.UUID {
	return a.packets().ServiceSol()
}

// Connectable indicates weather the remote peripheral is connectable.
func (a *Advertisement) Connectable() bool {
	return a.EventType() == evtTypAdvDirectInd || a.EventType() == evtTypAdvInd
}

// RSSI returns RSSI signal strength.
func (a *Advertisement) RSSI() int {
	return int(a.e.RSSI(a.i))
}

// Addr returns the address of the remote peripheral.
func (a *Advertisement) Addr() ble.Addr {
	b := a.e.Address(a.i)
	addr := net.HardwareAddr([]byte{b[5], b[4], b[3], b[2], b[1], b[0]})
	if a.e.AddressType(a.i) == 1 {
		return RandomAddress{addr}
	}
	return addr
}

// EventType returns the event type of Advertisement.
// This is linux specific.
func (a *Advertisement) EventType() uint8 {
	return a.e.EventType(a.i)
}

// AddressType returns the address type of the Advertisement.
// This is linux specific.
func (a *Advertisement) AddressType() uint8 {
	return a.e.AddressType(a.i)
}

// Data returns the advertising data of the packet.
// This is linux specific.
func (a *Advertisement) Data() []byte {
	return a.e.Data(a.i)
}

// ScanResponse returns the scan response of the packet, if it presents.
// This is linux specific.
func (a *Advertisement) ScanResponse() []byte {
	if a.sr == nil {
		return nil
	}
	return a.sr.Data()
}

func (a *Advertisement) ToMap() *ble.AdvertisementMap {
	m := make(ble.AdvertisementMap)
	keys := ble.AdvertisementMapKeys

	addr := a.Addr().String()
	if len(addr) == 0 {
		//ignore
		return nil
	}

	m[keys.MAC] = strings.Replace(addr, ":", "", -1)
	m[keys.EventType] = a.EventType()
	m[keys.Connectable] = a.Connectable()

	//require rssi!
	if v := a.RSSI(); v != 0 {
		m[keys.RSSI] = v
	} else {
		m[keys.RSSI] = -128
	}

	if v := a.LocalName(); len(v) != 0 {
		m[keys.Name] = v
	}

	if v := a.ManufacturerData(); v != nil {
		m[keys.MFG] = v
	}

	if v := a.Services(); v != nil {
		m[keys.Services] = v
	}

	if v := a.ServiceData(); v != nil {
		m[keys.ServiceData] = v
	}

	if v := a.SolicitedService(); v != nil {
		m[keys.Solicited] = v
	}

	return &m
}
