package hci

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"

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

func newAdvertisement(e evt.LEAdvertisingReport, i int) (*Advertisement, error) {
	ad, err := e.DataWErr(i)
	if err != nil {
		return nil, err
	}
	p, err := adv.NewRawPacket(ad)
	if err != nil {
		a := make([]byte, 6)
		addrArray := e.Address(i)
		for i := 0; i < 6; i++ {
			a[i] = addrArray[i]
		}
		fmt.Println("address:", hex.EncodeToString(a))
		return nil, err
	}

	a := &Advertisement{e: e, i: i, p: p}
	return a, nil
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
func (a *Advertisement) setScanResponse(sr *Advertisement) error {

	ad, err := a.e.DataWErr(a.i)
	if err != nil {
		return err
	}

	srd, err := sr.e.DataWErr(sr.i)
	if err != nil {
		return err
	}

	//does this parse ok?
	p, err := adv.NewRawPacket(ad, srd)
	if err != nil {
		return errors.Wrap(err, "setScanResp")
	}

	a.sr = sr
	a.p = p

	return nil
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
	v, _ := a.scanResponseWErr()
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

	//join the adv data maps
	if a.p != nil {
		for k, v := range a.p.Map() {
			//some special processing requirements for certain keys
			//todo: this should be handled better in the parser
			if k == keys.Name {
				if bytes, ok := v.([]byte); ok {
					m[k] = string(bytes)
				} else {
					m[k] = v
				}
			} else if k == keys.TxPower {
				if bytes, ok := v.([]byte); ok {
					m[k] = int(bytes[0])
				} else {
					m[k] = v
				}
			} else {
				m[k] = v
			}
		}
	}

	return m, nil
}
