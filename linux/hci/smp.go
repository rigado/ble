package hci

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	pairingRequest          = 0x01 // Pairing Request LE-U, ACL-U
	pairingResponse         = 0x02 // Pairing Response LE-U, ACL-U
	pairingConfirm          = 0x03 // Pairing Confirm LE-U
	pairingRandom           = 0x04 // Pairing Random LE-U
	pairingFailed           = 0x05 // Pairing Failed LE-U, ACL-U
	encryptionInformation   = 0x06 // Encryption Information LE-U
	masterIdentification    = 0x07 // Master Identification LE-U
	identityInformation     = 0x08 // Identity Information LE-U, ACL-U
	identityAddrInformation = 0x09 // Identity Address Information LE-U, ACL-U
	signingInformation      = 0x0A // Signing Information LE-U, ACL-U
	securityRequest         = 0x0B // Security Request LE-U
	pairingPublicKey        = 0x0C // Pairing Public Key LE-U
	pairingDHKeyCheck       = 0x0D // Pairing DHKey Check LE-U
	pairingKeypress         = 0x0E // Pairing Keypress Notification LE-U
)

type smpDispatcher struct {
	desc    string
	handler func(p pdu) error
}

var dispatcher = map[byte]smpDispatcher{
	pairingRequest:          smpDispatcher{"pairing request", nil},
	pairingResponse:         smpDispatcher{"pairing response", nil},
	pairingConfirm:          smpDispatcher{"pairing confirm", nil},
	pairingRandom:           smpDispatcher{"pairing random", nil},
	pairingFailed:           smpDispatcher{"pairing failed", nil},
	encryptionInformation:   smpDispatcher{"encryption info", nil},
	masterIdentification:    smpDispatcher{"master id", nil},
	identityInformation:     smpDispatcher{"id info", nil},
	identityAddrInformation: smpDispatcher{"id addr info", nil},
	signingInformation:      smpDispatcher{"signing info", nil},
	securityRequest:         smpDispatcher{"security req", nil},
	pairingPublicKey:        smpDispatcher{"pairing pub key", nil},
	pairingDHKeyCheck:       smpDispatcher{"pairing dhkey check", nil},
	pairingKeypress:         smpDispatcher{"pairing keypress", nil},
}

func (c *Conn) sendSMP(p pdu) error {
	buf := bytes.NewBuffer(make([]byte, 0))
	if err := binary.Write(buf, binary.LittleEndian, uint16(4+len(p))); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, cidSMP); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, p); err != nil {
		return err
	}
	_, err := c.writePDU(buf.Bytes())
	fmt.Println("smp", "tx", fmt.Sprintf("[%X]", buf.Bytes()))
	return err
}

func (c *Conn) handleSMP(p pdu) error {
	fmt.Println("smp", "rx", fmt.Sprintf("[%X]", p))

	code := p[0]
	v, ok := dispatcher[code]
	if !ok {
		logger.Error("smp", "unhandled smp code %v", code)
		return c.sendSMP([]byte{pairingFailed, 0x05})
	}

	fmt.Println("smp", "rx type:", v.desc)

	if v.handler != nil {
		//todo!!
		return v.handler(p)
	}

	// FIXME: work aound to the lack of SMP implementation - always return non-supported.
	// C.5.1 Pairing Not Supported by Slave
	return c.sendSMP([]byte{pairingFailed, 0x05})
}
