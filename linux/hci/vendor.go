package hci

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type CustomCommand struct {
	Payload interface{}
	opCode  int
	length  int
}

func (c *CustomCommand) OpCode() int {
	return c.opCode
}

func (c *CustomCommand) Len() int {
	return c.length
}

func (c *CustomCommand) Marshal(b []byte) error {

	buf := bytes.NewBuffer(b)
	buf.Reset()
	if buf.Cap() < c.Len() {
		return io.ErrShortBuffer
	}

	return binary.Write(buf, binary.LittleEndian, c.Payload)
}

func (c *CustomCommand) String() string {
	ogf := (c.opCode & 0xFC00) >> 10
	ocf := c.opCode & 0x3FF

	return fmt.Sprintf("Custom Command (0x%02x|0x%04x); Payload (%02x)", ogf, ocf, c.Payload)
}

func (h *HCI) SendVendorSpecificCommand(op uint16, length uint8, v interface{}) error {
	if length > maxHciPayload {
		return fmt.Errorf("invalid length %v; max hci payload length is %v", length, maxHciPayload)
	}

	opcode := (ogfVendorSpecificDebug << ogfBitShift) | op

	c := &CustomCommand{
		opCode:  int(opcode),
		length:  int(length),
		Payload: v,
	}

	err := h.Send(c, nil)
	if err != nil {
		return err
	}

	return nil
}
