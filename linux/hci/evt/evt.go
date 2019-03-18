package evt

import "encoding/binary"

func (e CommandComplete) NumHCICommandPackets() uint8 {
	return getByte(e, 0, 0)
}
func (e CommandComplete) CommandOpcode() uint16 {
	return getUint16LE(e, 1, 0xffff)
}
func (e CommandComplete) ReturnParameters() []byte {
	b, ok := getBytes(e, 3, -1)
	if !ok {
		return []byte{}
	}

	return b
}

// Per-spec [Vol 2, Part E, 7.7.19], the packet structure should be:
//
//     NumOfHandle, HandleA, HandleB, CompPktNumA, CompPktNumB
//
// But we got the actual packet from BCM20702A1 with the following structure instead.
//
//     NumOfHandle, HandleA, CompPktNumA, HandleB, CompPktNumB
//              02,   40 00,       01 00,   41 00,       01 00

func (e NumberOfCompletedPackets) NumberOfHandles() uint8 {
	return getByte(e, 0, 0)
}
func (e NumberOfCompletedPackets) ConnectionHandle(i int) uint16 {
	si := 1 + (i * 4)
	return getUint16LE(e, si, 0xffff)
}
func (e NumberOfCompletedPackets) HCNumOfCompletedPackets(i int) uint16 {
	si := 1 + (i * 4) + 2
	return getUint16LE(e, si, 0)
}
func (e LEAdvertisingReport) SubeventCode() uint8 {
	return getByte(e, 0, 0xff)
}

func (e LEAdvertisingReport) NumReports() uint8 {
	return getByte(e, 1, 0)
}

func (e LEAdvertisingReport) EventType(i int) uint8 {
	return getByte(e, 2+i, 0xff)
}
func (e LEAdvertisingReport) AddressType(i int) uint8 {
	nr := e.NumReports()
	if nr == 0 {
		return 0xff
	}

	si := 2 + int(nr) + i
	return getByte(e, si, 0xff)
}
func (e LEAdvertisingReport) Address(i int) [6]byte {
	nr := e.NumReports()
	if nr == 0 {
		return [6]byte{}
	}

	si := 2 + int(nr)*2 + (6 * i)
	bb, ok := getBytes(e, si, 6)
	if !ok {
		return [6]byte{}
	}

	out := [6]byte{}
	copy(out[:], bb)
	return out
}

func (e LEAdvertisingReport) LengthData(i int) uint8 {
	nr := e.NumReports()
	if nr == 0 {
		return 0
	}

	si := 2 + int(nr)*8 + i
	return getByte(e, si, 0)
}

func (e LEAdvertisingReport) Data(i int) []byte {
	nr := e.NumReports()
	if nr == 0 {
		return nil
	}

	l := 0
	for j := 0; j < i; j++ {
		l += int(e.LengthData(j))
	}

	ld := e.LengthData(i)
	si := 2 + int(nr)*9 + l

	if ld == 0 {
		return nil
	}

	b, ok := getBytes(e, si, int(ld))
	if !ok {
		return nil
	}
	return b
}

func (e LEAdvertisingReport) RSSI(i int) int8 {
	nr := e.NumReports()
	if nr == 0 {
		return 0
	}

	l := 0
	for j := 0; j < int(nr); j++ {
		l += int(e.LengthData(j))
	}

	si := 2 + int(nr)*9 + l + i
	return int8(getByte(e, si, 0))
}

//get or default
func getByte(b []byte, i int, def byte) byte {
	bb, ok := getBytes(b, i, 1)
	if !ok {
		return def
	}
	return bb[0]
}

//get or default
func getUint16LE(b []byte, i int, def uint16) uint16 {
	bb, ok := getBytes(b, i, 2)
	if !ok {
		return def
	}
	return binary.LittleEndian.Uint16(bb)
}

func getBytes(bytes []byte, start int, count int) ([]byte, bool) {
	if bytes == nil || start >= len(bytes) {
		return nil, false
	}

	if count < 0 {
		return bytes[start:], true
	}

	end := start + count
	//end is non-inclusive
	if end > len(bytes) {
		return nil, false
	}

	return bytes[start:end], true
}
