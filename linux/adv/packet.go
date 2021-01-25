package adv

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/rigado/ble"
	"github.com/rigado/ble/parser"
)

var keys = struct {
	flags       string
	services    string
	solicited   string
	serviceData string
	localName   string
	txpwr       string
	mfgdata     string
}{
	flags:       ble.AdvertisementMapKeys.Flags,
	services:    ble.AdvertisementMapKeys.Services,
	solicited:   ble.AdvertisementMapKeys.Solicited,
	serviceData: ble.AdvertisementMapKeys.ServiceData,
	localName:   ble.AdvertisementMapKeys.Name,
	txpwr:       ble.AdvertisementMapKeys.TxPower,
	mfgdata:     ble.AdvertisementMapKeys.MFG,
}

// Packet is an implemntation of ble.AdvPacket for crafting or parsing an advertising packet or scan response.
// Refer to Supplement to Bluetooth Core Specification | CSSv6, Part A.
type Packet struct {
	b []byte
	m map[string]interface{}
}

// Bytes returns the bytes of the packet.
func (p *Packet) Bytes() []byte {
	return p.b
}

// Len returns the length of the packet.
func (p *Packet) Len() int {
	return len(p.b)
}

func (p *Packet) Map() map[string]interface{} {
	return p.m
}

// NewPacket returns a new advertising Packet.
func NewPacket(fields ...Field) (*Packet, error) {
	p := &Packet{b: make([]byte, 0, MaxEIRPacketLength)}
	for _, f := range fields {
		if err := f(p); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// NewRawPacket returns a new advertising Packet.
func NewRawPacket(bytes ...[]byte) (*Packet, error) {
	//concatenate
	b := make([]byte, 0, MaxEIRPacketLength)
	for _, bb := range bytes {
		b = append(b, bb...)
	}

	//decode the bytes
	m, err := parser.Parse(b)
	err = errors.Wrapf(err, "pdu decode")
	switch {
	case err == nil:
		// ok
	case len(m) > 0:
		// some of the adv was ok, append the error
		m[ble.AdvertisementMapKeys.AdvertisementError] = err.Error()
	default:
		// nothing was ok parsed, exit
		return nil, err
	}

	p := &Packet{b: b, m: m}
	return p, nil
}

// Field is an advertising field which can be appended to a packet.
type Field func(p *Packet) error

// Append appends a field to the packet. It returns ErrNotFit if the field
// doesn't fit into the packet, and leaves the packet intact.
func (p *Packet) Append(f Field) error {
	return f(p)
}

// appends appends a field to the packet. It returns ErrNotFit if the field
// doesn't fit into the packet, and leaves the packet intact.
func (p *Packet) append(typ byte, b []byte) error {
	if p.Len()+1+1+len(b) > MaxEIRPacketLength {
		return ErrNotFit
	}
	p.b = append(p.b, byte(len(b)+1))
	p.b = append(p.b, typ)
	p.b = append(p.b, b...)
	return nil
}

// Raw appends the bytes to the current packet.
// This is helpful for creating new packet from existing packets.
func Raw(b []byte) Field {
	return func(p *Packet) error {
		if p.Len()+len(b) > MaxEIRPacketLength {
			return ErrNotFit
		}
		p.b = append(p.b, b...)
		return nil
	}
}

// IBeaconData returns an iBeacon advertising packet with specified parameters.
func IBeaconData(md []byte) Field {
	return func(p *Packet) error {
		return ManufacturerData(0x004C, md)(p)
	}
}

// IBeacon returns an iBeacon advertising packet with specified parameters.
func IBeacon(u ble.UUID, major, minor uint16, pwr int8) Field {
	return func(p *Packet) error {
		if u.Len() != 16 {
			return ErrInvalid
		}
		md := make([]byte, 23)
		md[0] = 0x02                               // Data type: iBeacon
		md[1] = 0x15                               // Data length: 21 bytes
		copy(md[2:], ble.Reverse(u))               // Big endian
		binary.BigEndian.PutUint16(md[18:], major) // Big endian
		binary.BigEndian.PutUint16(md[20:], minor) // Big endian
		md[22] = uint8(pwr)                        // Measured Tx Power
		return ManufacturerData(0x004C, md)(p)
	}
}

// Flags is a flags.
func Flags(f byte) Field {
	return func(p *Packet) error {
		return p.append(flags, []byte{f})
	}
}

// ShortName is a short local name.
func ShortName(n string) Field {
	return func(p *Packet) error {
		return p.append(shortName, []byte(n))
	}
}

// CompleteName is a compelete local name.
func CompleteName(n string) Field {
	return func(p *Packet) error {
		return p.append(completeName, []byte(n))
	}
}

// ManufacturerData is manufacturer specific data.
func ManufacturerData(id uint16, b []byte) Field {
	return func(p *Packet) error {
		d := append([]byte{uint8(id), uint8(id >> 8)}, b...)
		return p.append(manufacturerData, d)
	}
}

// AllUUID is one of the complete service UUID list.
func AllUUID(u ble.UUID) Field {
	return func(p *Packet) error {
		if u.Len() == 2 {
			return p.append(allUUID16, u)
		}
		if u.Len() == 4 {
			return p.append(allUUID32, u)
		}
		return p.append(allUUID128, u)
	}
}

// SomeUUID is one of the incomplete service UUID list.
func SomeUUID(u ble.UUID) Field {
	return func(p *Packet) error {
		if u.Len() == 2 {
			return p.append(someUUID16, u)
		}
		if u.Len() == 4 {
			return p.append(someUUID32, u)
		}
		return p.append(someUUID128, u)
	}
}

// ServiceData16 is service data for a 16bit service uuid
func ServiceData16(id uint16, b []byte) Field {
	return func(p *Packet) error {
		uuid := ble.UUID16(id)
		if err := p.append(allUUID16, uuid); err != nil {
			return err
		}
		return p.append(serviceData16, append(uuid, b...))
	}
}

// Flags returns the flags of the packet.
func (p *Packet) Flags() (flags byte, present bool) {
	if b, ok := p.m[keys.flags].([]byte); ok {
		return b[0], true
	}
	return 0, false
}

// LocalName returns the ShortName or CompleteName if it presents.
func (p *Packet) LocalName() string {
	if b, ok := p.m[keys.localName].([]byte); ok {
		return string(b)
	}
	return ""
}

// TxPower returns the TxPower, if it presents.
func (p *Packet) TxPower() (power int, present bool) {
	if b, ok := p.m[keys.txpwr].([]byte); ok {
		txpwr := int(int8(b[0]))
		return txpwr, true
	}
	return 0, false
}

// UUIDs returns a list of service UUIDs.
func (p *Packet) UUIDs() []ble.UUID {
	v, _ := p.m[keys.services].([]ble.UUID)
	return v
}

// ServiceSol ...
func (p *Packet) ServiceSol() []ble.UUID {
	v, _ := p.m[keys.solicited].([]ble.UUID)
	return v
}

// ServiceData ...
func (p *Packet) ServiceData() []ble.ServiceData {
	m, ok := p.m[keys.serviceData].(map[string]interface{})
	if !ok {
		return nil
	}

	// map -> array
	out := []ble.ServiceData{}
	for su, val := range m {

		arr, ok := val.([]interface{})
		if !ok {
			continue
		}

		for _, v := range arr {
			sd, ok := v.([]byte)
			if !ok {
				continue
			}
			u, err := ble.Parse(su)
			if err != nil {
				continue
			}
			out = append(out, ble.ServiceData{UUID: u, Data: sd})
		}
	}

	return out
}

// ManufacturerData returns the ManufacturerData field if it presents.
func (p *Packet) ManufacturerData() []byte {
	v, _ := p.m[keys.mfgdata].([]byte)
	return v
}
