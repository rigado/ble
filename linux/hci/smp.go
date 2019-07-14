package hci

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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
	handler func(*Conn, pdu) ([]byte, error)
}

func (c *Conn) sendSMP(p pdu) error {
	buf := bytes.NewBuffer(make([]byte, 0))
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(p))); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, cidSMP); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, p); err != nil {
		return err
	}
	_, err := c.writePDU(buf.Bytes())
	// fmt.Printf("smp tx %v, err %v\n", fmt.Sprintf("[%X]", buf.Bytes()), err)
	return err
}

func (c *Conn) handleSMP(p pdu) error {
	fmt.Printf("enter handleSMP ====================\n")
	defer fmt.Printf("exit handleSMP ====================\n")

	// fmt.Println("smp", "rx", fmt.Sprintf("[%X]", p))

	payload := p.payload()
	code := payload[0]
	data := payload[1:]
	v, ok := dispatcher[code]
	if !ok {
		logger.Error("smp", "unhandled smp code %v", code)
		return c.sendSMP([]byte{pairingFailed, 0x05})
	}

	fmt.Println("smp", "rx type:", v.desc)
	if v.handler != nil {
		//todo!!
		// fmt.Println("dispatching to smp handler...")
		_, err := v.handler(c, data)
		if err != nil {
			log.Println(err)
			return err
		}

		if c.pairing != nil {
			fmt.Printf("%+v\n", *c.pairing)
		}

		return nil
		// return c.sendSMP(r)
	}

	fmt.Println("no smp handler...")
	// FIXME: work around to the lack of SMP implementation - always return non-supported.
	// C.5.1 Pairing Not Supported by Slave
	return nil //c.sendSMP([]byte{pairingFailed, 0x05})
}

func (c *Conn) Bond() error {
	return c.smpSendPairingRequest()
}

func (c *Conn) EnableEncryption() error {
	bondFile := filepath.Join(os.Getenv("SNAP_DATA"), "bonds.json")
	fileData, err := ioutil.ReadFile(bondFile)
	if err != nil {
		return fmt.Errorf("no bond information to load")
	}

	var bonds bondInfo
	if len(fileData) > 0 {
		err = json.Unmarshal(fileData, &bonds)
		if err != nil {
			return fmt.Errorf("failed to unmarshal current bond info: %s", err)
		}
	}

	da := c.RemoteAddr().Bytes()
	da = swapBuf(da)
	addr := hex.EncodeToString(da)
	for _, bond := range bonds.Bonds {
		if bond.Address == addr {
			ltk, err := hex.DecodeString(bond.LongTermKey)
			if err != nil {
				return fmt.Errorf("failed to decode long term key: %s", err)
			}

			ediv, err := hex.DecodeString(bond.EncryptionDiversifier)
			if err != nil {
				return fmt.Errorf("failed to decode ediv: %s", err)
			}

			rv, err := hex.DecodeString(bond.RandomValue)
			if err != nil {
				return fmt.Errorf("failed to decode random value: %s", err)
			}

			c.pairing = &pairingContext{
				ltk: ltk,
				ediv: binary.LittleEndian.Uint16(ediv),
				rand: binary.LittleEndian.Uint64(rv),
			}
		}
	}

	if c.pairing == nil {
		return fmt.Errorf("no encryption information found")
	}

	err = c.encrypt()
	if err != nil {
		return fmt.Errorf("failed to start encryption: %s", err)
	}

	return nil
}

func isLegacy(authReq byte) bool {
	if authReq & 0x08 == 0x08 {
		return false
	}

	return true
}
