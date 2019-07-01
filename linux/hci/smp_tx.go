package hci

import (
	"crypto/rand"
	"fmt"
	"log"
)

func (c *Conn) smpSendPairingRequest() error {
	p := smp.config
	b := []byte{pairingRequest, p.ioCap, p.oobFlag, p.authReq, p.maxKeySz, p.initKeyDist, p.respKeyDist}
	return c.sendSMP(pdu(b))
}

func (c *Conn) smpSendPublicKey() error {
	kp := smp.keys

	k := MarshalPublicKeyXY(kp.public)
	out := append([]byte{pairingPublicKey}, k...)
	err := c.sendSMP(pdu(out))

	if err != nil {
		return err
	}

	if c.pairing != nil {
		fmt.Printf("discarding pairing context: %+v\n", *c.pairing)
	}

	ra := make([]byte, 0)
	for _, v := range c.param.PeerAddress() {
		ra = append(ra, v)
	}
	ra = append(ra, c.param.PeerAddressType())

	// todo get local addr!!!
	c.pairing = &pairingContext{
		localKeys: kp,
		//94:54:93:2F:5D:62 (mac of -00169)
		// localAddr:  []byte{0x62, 0x5d, 0x2f, 0x93, 0x54, 0x94, 0}, //type at the end!!!

		// mac of evk on laptop
		localAddr:  []byte{0x94, 0x54, 0x93, 0x93, 0x54, 0x94, 0}, //type at the end!!!
		remoteAddr: ra,
	}

	return nil
}

func (c *Conn) smpSendPairingRandom() error {
	if c.pairing == nil {
		return fmt.Errorf("no pairing context")
	}

	log.Printf("send pairing rand")

	r := make([]byte, 16)
	_, err := rand.Read(r)
	if err != nil {
		return err
	}

	c.pairing.localRandom = r
	out := append([]byte{pairingRandom}, r...)

	return c.sendSMP(pdu(out))
}

func (c *Conn) smpSendDHKeyCheck() error {
	if c.pairing == nil {
		return fmt.Errorf("no pairing context")
	}

	log.Printf("send dhkey check")
	p := c.pairing

	//Ea = f6 (MacKey, Na, Nb, 0, IOcapA, A, B)
	la := p.localAddr.([]byte)
	ra := p.remoteAddr.([]byte)
	na := p.localRandom.([]byte)
	nb := p.remoteRandom.([]byte)
	ioCap := swapBuf([]byte{smp.config.authReq, smp.config.oobFlag, smp.config.ioCap})

	ea, err := smpF6(p.macKey, na, nb, make([]byte, 16), ioCap, la, ra)
	if err != nil {
		return err
	}

	out := append([]byte{pairingDHKeyCheck}, ea...)
	return c.sendSMP(pdu(out))
}

func (c *Conn) smpStartEncryption() error {
	if c.pairing == nil {
		return fmt.Errorf("no pairing context")
	}
	log.Printf("start encryption")
	return nil
}
