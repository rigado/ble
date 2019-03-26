package evt

func (e CommandComplete) NumHCICommandPackets() uint8 {
	v, _ := e.NumHCICommandPacketsWErr()
	return v
}

func (e CommandComplete) CommandOpcode() uint16 {
	v, _ := e.CommandOpcodeWErr()
	return v
}

func (e CommandComplete) ReturnParameters() []byte {
	v, _ := e.ReturnParametersWErr()
	return v
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
	v, _ := e.NumberOfHandlesWErr()
	return v
}

func (e NumberOfCompletedPackets) ConnectionHandle(i int) uint16 {
	v, _ := e.ConnectionHandleWErr(i)
	return v
}

func (e NumberOfCompletedPackets) HCNumOfCompletedPackets(i int) uint16 {
	v, _ := e.HCNumOfCompletedPacketsWErr(i)
	return v
}
func (e LEAdvertisingReport) SubeventCode() uint8 {
	v, _ := e.SubeventCodeWErr()
	return v
}

func (e LEAdvertisingReport) NumReports() uint8 {
	v, _ := e.NumReportsWErr()
	return v
}

func (e LEAdvertisingReport) EventType(i int) uint8 {
	v, _ := e.EventTypeWErr(i)
	return v
}

func (e LEAdvertisingReport) AddressType(i int) uint8 {
	v, _ := e.AddressTypeWErr(i)
	return v
}

func (e LEAdvertisingReport) Address(i int) [6]byte {
	v, _ := e.AddressWErr(i)
	return v
}

func (e LEAdvertisingReport) LengthData(i int) uint8 {
	v, _ := e.LengthDataWErr(i)
	return v
}

func (e LEAdvertisingReport) Data(i int) []byte {
	v, _ := e.DataWErr(i)
	return v
}

func (e LEAdvertisingReport) RSSI(i int) int8 {
	v, _ := e.RSSIWErr(i)
	return v
}
