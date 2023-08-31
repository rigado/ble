package hci

import (
	"bytes"
	"encoding/binary"
)

// SignalCommandReject is the code of Command Reject signaling packet.
const SignalCommandReject = 0x01

// CommandReject implements Command Reject (0x01) [Vol 3, Part A, 4.1].
type CommandReject struct {
	Reason uint16
	Data   []byte
}

// Code returns the event code of the command.
func (s CommandReject) Code() int { return 0x01 }

// Marshal serializes the command parameters into binary form.
func (s *CommandReject) Marshal() []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, s)
	return buf.Bytes()
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (s *CommandReject) Unmarshal(b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, s)
}

// SignalL2CAPConnectionRequest is the code of L2CAP Connection Request signaling packet.
const SignalL2CAPConnectionRequest = 0x02

// L2CAPConnectionRequest implements L2CAP Connection Request (0x02) [Vol 3, Part A, 4.2].
type L2CAPConnectionRequest struct {
	PSM       uint16
	SourceCID uint16
}

// Code returns the event code of the command.
func (s L2CAPConnectionRequest) Code() int { return 0x02 }

// Marshal serializes the command parameters into binary form.
func (s *L2CAPConnectionRequest) Marshal() []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, s)
	return buf.Bytes()
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (s *L2CAPConnectionRequest) Unmarshal(b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, s)
}

// SignalL2CAPConnectionResponse is the code of L2CAP Connection Response signaling packet.
const SignalL2CAPConnectionResponse = 0x03

// L2CAPConnectionResponse implements L2CAP Connection Response (0x03) [Vol 3, Part A, 4.3].
type L2CAPConnectionResponse struct {
	DestinationCID uint16
	SourceCID      uint16
	MTU            uint16
	Result         uint16
	Status         uint16
}

// Code returns the event code of the command.
func (s L2CAPConnectionResponse) Code() int { return 0x03 }

// Marshal serializes the command parameters into binary form.
func (s *L2CAPConnectionResponse) Marshal() []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, s)
	return buf.Bytes()
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (s *L2CAPConnectionResponse) Unmarshal(b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, s)
}

// SignalDisconnectRequest is the code of Disconnect Request signaling packet.
const SignalDisconnectRequest = 0x06

// DisconnectRequest implements Disconnect Request (0x06) [Vol 3, Part A, 4.6].
type DisconnectRequest struct {
	DestinationCID uint16
	SourceCID      uint16
}

// Code returns the event code of the command.
func (s DisconnectRequest) Code() int { return 0x06 }

// Marshal serializes the command parameters into binary form.
func (s *DisconnectRequest) Marshal() []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, s)
	return buf.Bytes()
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (s *DisconnectRequest) Unmarshal(b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, s)
}

// SignalDisconnectResponse is the code of Disconnect Response signaling packet.
const SignalDisconnectResponse = 0x07

// DisconnectResponse implements Disconnect Response (0x07) [Vol 3, Part A, 4.7].
type DisconnectResponse struct {
	DestinationCID uint16
	SourceCID      uint16
}

// Code returns the event code of the command.
func (s DisconnectResponse) Code() int { return 0x07 }

// Marshal serializes the command parameters into binary form.
func (s *DisconnectResponse) Marshal() []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, s)
	return buf.Bytes()
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (s *DisconnectResponse) Unmarshal(b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, s)
}

// SignalConnectionParameterUpdateRequest is the code of Connection Parameter Update Request signaling packet.
const SignalConnectionParameterUpdateRequest = 0x12

// ConnectionParameterUpdateRequest implements Connection Parameter Update Request (0x12) [Vol 3, Part A, 4.20].
type ConnectionParameterUpdateRequest struct {
	IntervalMin       uint16
	IntervalMax       uint16
	SlaveLatency      uint16
	TimeoutMultiplier uint16
}

// Code returns the event code of the command.
func (s ConnectionParameterUpdateRequest) Code() int { return 0x12 }

// Marshal serializes the command parameters into binary form.
func (s *ConnectionParameterUpdateRequest) Marshal() []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, s)
	return buf.Bytes()
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (s *ConnectionParameterUpdateRequest) Unmarshal(b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, s)
}

// SignalConnectionParameterUpdateResponse is the code of Connection Parameter Update Response signaling packet.
const SignalConnectionParameterUpdateResponse = 0x13

// ConnectionParameterUpdateResponse implements Connection Parameter Update Response (0x13) [Vol 3, Part A, 4.21].
type ConnectionParameterUpdateResponse struct {
	Result uint16
}

// Code returns the event code of the command.
func (s ConnectionParameterUpdateResponse) Code() int { return 0x13 }

// Marshal serializes the command parameters into binary form.
func (s *ConnectionParameterUpdateResponse) Marshal() []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, s)
	return buf.Bytes()
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (s *ConnectionParameterUpdateResponse) Unmarshal(b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, s)
}

// SignalLECreditBasedConnectionRequest is the code of LE Credit Based Connection Request signaling packet.
const SignalLECreditBasedConnectionRequest = 0x14

// LECreditBasedConnectionRequest implements LE Credit Based Connection Request (0x14) [Vol 3, Part A, 4.22].
type LECreditBasedConnectionRequest struct {
	LEPSM          uint16
	SourceCID      uint16
	MTU            uint16
	MPS            uint16
	InitialCredits uint16
}

// Code returns the event code of the command.
func (s LECreditBasedConnectionRequest) Code() int { return 0x14 }

// Marshal serializes the command parameters into binary form.
func (s *LECreditBasedConnectionRequest) Marshal() []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, s)
	return buf.Bytes()
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (s *LECreditBasedConnectionRequest) Unmarshal(b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, s)
}

// SignalLECreditBasedConnectionResponse is the code of LE Credit Based Connection Response signaling packet.
const SignalLECreditBasedConnectionResponse = 0x15

// LECreditBasedConnectionResponse implements LE Credit Based Connection Response (0x15) [Vol 3, Part A, 4.23].
type LECreditBasedConnectionResponse struct {
	DestinationCID    uint16
	MTU               uint16
	MPS               uint16
	InitialCreditsCID uint16
	Result            uint16
}

// Code returns the event code of the command.
func (s LECreditBasedConnectionResponse) Code() int { return 0x15 }

// Marshal serializes the command parameters into binary form.
func (s *LECreditBasedConnectionResponse) Marshal() []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, s)
	return buf.Bytes()
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (s *LECreditBasedConnectionResponse) Unmarshal(b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, s)
}

// SignalLEFlowControlCredit is the code of LE Flow Control Credit signaling packet.
const SignalLEFlowControlCredit = 0x16

// LEFlowControlCredit implements LE Flow Control Credit (0x16) [Vol 3, Part A, 4.24].
type LEFlowControlCredit struct {
	CID     uint16
	Credits uint16
}

// Code returns the event code of the command.
func (s LEFlowControlCredit) Code() int { return 0x16 }

// Marshal serializes the command parameters into binary form.
func (s *LEFlowControlCredit) Marshal() []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, s)
	return buf.Bytes()
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (s *LEFlowControlCredit) Unmarshal(b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, s)
}

// SignalL2CAPCreditBasedConnectionRequest is the code of L2CAP Credit Based Connection Request signaling packet.
const SignalL2CAPCreditBasedConnectionRequest = 0x17

// L2CAPCreditBasedConnectionRequest implements L2CAP Credit Based Connection Request (0x17) [Vol 3, Part A, 4.25].
type L2CAPCreditBasedConnectionRequest struct {
	SPSM           uint16
	MTU            uint16
	MPS            uint16
	InitialCredits uint16
	SourceCID      uint16
}

// Code returns the event code of the command.
func (s L2CAPCreditBasedConnectionRequest) Code() int { return 0x17 }

// Marshal serializes the command parameters into binary form.
func (s *L2CAPCreditBasedConnectionRequest) Marshal() []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, s)
	return buf.Bytes()
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (s *L2CAPCreditBasedConnectionRequest) Unmarshal(b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, s)
}

// SignalL2CAPCreditBasedConnectionResponse is the code of L2CAP Credit Based Connection Response signaling packet.
const SignalL2CAPCreditBasedConnectionResponse = 0x18

// L2CAPCreditBasedConnectionResponse implements L2CAP Credit Based Connection Response (0x18) [Vol 3, Part A, 4.26].
type L2CAPCreditBasedConnectionResponse struct {
	MTU               uint16
	MPS               uint16
	InitialCreditsCID uint16
	Result            uint16
	DestinationCID    uint16
}

// Code returns the event code of the command.
func (s L2CAPCreditBasedConnectionResponse) Code() int { return 0x18 }

// Marshal serializes the command parameters into binary form.
func (s *L2CAPCreditBasedConnectionResponse) Marshal() []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, s)
	return buf.Bytes()
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (s *L2CAPCreditBasedConnectionResponse) Unmarshal(b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, s)
}
