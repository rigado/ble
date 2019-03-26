package evt

import (
	"encoding/binary"
	"fmt"
)

func (e CommandComplete) NumHCICommandPacketsWErr() (uint8, error) {
	return getByte(e, 0, 0)
}

func (e CommandComplete) CommandOpcodeWErr() (uint16, error) {
	return getUint16LE(e, 1, 0xffff)
}
func (e CommandComplete) ReturnParametersWErr() ([]byte, error) {
	return getBytes(e, 3, -1)
}

// Per-spec [Vol 2, Part E, 7.7.19], the packet structure should be:
//
//     NumOfHandle, HandleA, HandleB, CompPktNumA, CompPktNumB
//
// But we got the actual packet from BCM20702A1 with the following structure instead.
//
//     NumOfHandle, HandleA, CompPktNumA, HandleB, CompPktNumB
//              02,   40 00,       01 00,   41 00,       01 00

func (e NumberOfCompletedPackets) NumberOfHandlesWErr() (uint8, error) {
	return getByte(e, 0, 0)
}
func (e NumberOfCompletedPackets) ConnectionHandleWErr(i int) (uint16, error) {
	si := 1 + (i * 4)
	return getUint16LE(e, si, 0xffff)
}
func (e NumberOfCompletedPackets) HCNumOfCompletedPacketsWErr(i int) (uint16, error) {
	si := 1 + (i * 4) + 2
	return getUint16LE(e, si, 0)
}
func (e LEAdvertisingReport) SubeventCodeWErr() (uint8, error) {
	return getByte(e, 0, 0xff)
}

func (e LEAdvertisingReport) NumReportsWErr() (uint8, error) {
	return getByte(e, 1, 0)
}

func (e LEAdvertisingReport) EventTypeWErr(i int) (uint8, error) {
	return getByte(e, 2+i, 0xff)
}
func (e LEAdvertisingReport) AddressTypeWErr(i int) (uint8, error) {
	nr, err := e.NumReportsWErr()
	if err != nil {
		return 0, err
	}

	si := 2 + int(nr) + i
	return getByte(e, si, 0xff)
}
func (e LEAdvertisingReport) AddressWErr(i int) ([6]byte, error) {
	nr, err := e.NumReportsWErr()
	if err != nil {
		return [6]byte{}, err
	}

	si := 2 + int(nr)*2 + (6 * i)
	bb, err := getBytes(e, si, 6)
	if err != nil {
		return [6]byte{}, err
	}

	out := [6]byte{}
	copy(out[:], bb)
	return out, nil
}

func (e LEAdvertisingReport) LengthDataWErr(i int) (uint8, error) {
	nr, err := e.NumReportsWErr()
	if err != nil {
		return 0, err
	}

	si := 2 + int(nr)*8 + i
	return getByte(e, si, 0)
}

func (e LEAdvertisingReport) DataWErr(i int) ([]byte, error) {
	nr, err := e.NumReportsWErr()
	if err != nil {
		return nil, err
	}

	l := 0
	for j := 0; j < i; j++ {
		ll, err := e.LengthDataWErr(j)

		if err != nil {
			return nil, err
		}

		l += int(ll)
	}

	ll, err := e.LengthDataWErr(i)
	if err != nil {
		return nil, err
	}
	si := 2 + int(nr)*9 + l
	b, err := getBytes(e, si, int(ll))
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (e LEAdvertisingReport) RSSIWErr(i int) (int8, error) {
	nr, err := e.NumReportsWErr()
	if err != nil {
		return 0, err
	}

	l := 0
	for j := 0; j < int(nr); j++ {
		ll, err := e.LengthDataWErr(j)
		if err != nil {
			return 0, err
		}

		l += int(ll)
	}

	si := 2 + int(nr)*9 + l + i
	rssi, err := getByte(e, si, 0)
	return int8(rssi), err
}

//get or default
func getByte(b []byte, i int, def byte) (byte, error) {
	bb, err := getBytes(b, i, 1)
	if err != nil {
		return def, err
	}
	return bb[0], nil
}

//get or default
func getUint16LE(b []byte, i int, def uint16) (uint16, error) {
	bb, err := getBytes(b, i, 2)
	if err != nil {
		return def, err
	}
	return binary.LittleEndian.Uint16(bb), nil
}

func getBytes(bytes []byte, start int, count int) ([]byte, error) {
	if bytes == nil || start >= len(bytes) {
		return nil, fmt.Errorf("index error")
	}

	if count < 0 {
		return bytes[start:], nil
	}

	end := start + count
	//end is non-inclusive
	if end > len(bytes) {
		return nil, fmt.Errorf("index error")
	}

	return bytes[start:end], nil
}
