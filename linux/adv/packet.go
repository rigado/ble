package adv

import (
	"encoding/binary"
	"fmt"

	"github.com/go-ble/ble"
	"github.com/pkg/errors"
)

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
	m, err := decode(b)
	if err != nil {
		return nil, errors.Wrap(err, "pdu decode")
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

func (p *Packet) getUUIDsByType(typ byte, u []ble.UUID, w int) []ble.UUID {
	var k string
	switch typ {
	case types.uuid16comp:
		k = keys.uuid16comp
	case types.uuid16inc:
		k = keys.uuid16inc
	case types.uuid32comp:
		k = keys.uuid32comp
	case types.uuid32inc:
		k = keys.uuid32inc
	case types.uuid128comp:
		k = keys.uuid128comp
	case types.uuid128inc:
		k = keys.uuid128inc
	default:
		fmt.Printf("invalid type %v for UUIDs", typ)
		return u
	}

	v, ok := p.m[k].([]interface{})
	if !ok {
		return u
	}

	//v should be [][]byte
	for _, vv := range v {
		b, ok := vv.([]byte)
		if !ok {
			continue
		}
		u = append(u, b)
	}
	return u
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
	if b, ok := p.m[keys.namecomp].([]byte); ok {
		return string(b)
	}

	//DSC: nameshort/complete both use the same key
	// if b, ok := p.m[keys.nameshort].([]byte); ok {
	// 	return string(b)
	// }

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
	var u []ble.UUID
	u = p.getUUIDsByType(someUUID16, u, 2)
	u = p.getUUIDsByType(allUUID16, u, 2)
	u = p.getUUIDsByType(someUUID32, u, 4)
	u = p.getUUIDsByType(allUUID32, u, 4)
	u = p.getUUIDsByType(someUUID128, u, 16)
	u = p.getUUIDsByType(allUUID128, u, 16)
	return u
}

// ServiceSol ...
func (p *Packet) ServiceSol() []ble.UUID {
	var u []ble.UUID
	if b, ok := p.m[keys.sol16].([]byte); ok {
		u = uuidList(u, b, 2)
	}
	if b, ok := p.m[keys.sol32].([]byte); ok {
		u = uuidList(u, b, 4)
	}
	if b, ok := p.m[keys.sol128].([]byte); ok {
		u = uuidList(u, b, 16)
	}
	return u
}

// ServiceData ...
func (p *Packet) ServiceData() []ble.ServiceData {
	var s []ble.ServiceData

	if b, ok := p.m[keys.svc16].([]byte); ok {
		s = serviceDataList(s, b, 2)
	}
	if b, ok := p.m[keys.svc32].([]byte); ok {
		s = serviceDataList(s, b, 4)
	}
	if b, ok := p.m[keys.svc128].([]byte); ok {
		s = serviceDataList(s, b, 16)
	}
	return s
}

// ManufacturerData returns the ManufacturerData field if it presents.
func (p *Packet) ManufacturerData() []byte {
	v, _ := p.m[keys.mfgdata].([]byte)
	return v
}

// Utility function for creating a list of uuids.
func uuidList(u []ble.UUID, d []byte, w int) []ble.UUID {
	for len(d) > 0 {
		u = append(u, ble.UUID(d[:w]))
		d = d[w:]
	}
	return u
}

func serviceDataList(sd []ble.ServiceData, d []byte, w int) []ble.ServiceData {
	serviceData := ble.ServiceData{
		UUID: ble.UUID(d[:w]),
		Data: make([]byte, len(d)-w),
	}
	copy(serviceData.Data, d[w:])
	return append(sd, serviceData)
}
