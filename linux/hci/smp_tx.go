package hci

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
)

func buildPairingReq(p smpConfig) []byte {
	return []byte{pairingRequest, p.ioCap, p.oobFlag, p.authReq, p.maxKeySz, p.initKeyDist, p.respKeyDist}
}

func buildPairingRsp(p smpConfig) []byte {
	return []byte{pairingResponse, p.ioCap, p.oobFlag, p.authReq, p.maxKeySz, p.initKeyDist, p.respKeyDist}
}

func (c *Conn) smpSendPairingRequest() error {
	//todo: create a new pairing context function
	ra := make([]byte, 0)
	for _, v := range c.param.PeerAddress() {
		ra = append(ra, v)
	}
	ra = append(ra, c.param.PeerAddressType())

	laBE := c.LocalAddr().Bytes()
	la := make([]byte, 0, 7)
	la = append(la, uint8(0))
	for _, v := range laBE {
		la = append(la, v)
	}
	la = swapBuf(la)

	// todo get local addr!!!
	c.pairing = &pairingContext{
		localKeys: nil,
		//94:54:93:2F:5D:62 (mac of -00169)
		// localAddr:  []byte{0x62, 0x5d, 0x2f, 0x93, 0x54, 0x94, 0}, //type at the end!!!

		localAddr:  la, //type at the end!!!
		remoteAddr: ra,
	}

	cmd := buildPairingReq(smp.config)
	return c.sendSMP(pdu(cmd))
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

	c.pairing.localKeys = kp

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

func (c *Conn) smpSendMConfirm(rsp smpConfig) error {
	preq := buildPairingReq(smp.config)
	pres := buildPairingRsp(rsp)
	c.pairing.legacyPairingResponse = pres

	r := make([]byte, 16)
	_, err := rand.Read(r)
	if err != nil {
		return err
	}
	c.pairing.localRandom = r

	la, ok := c.pairing.localAddr.([]byte)
	if !ok {
		return fmt.Errorf("invalid local address type")
	}

	ra, ok := c.pairing.remoteAddr.([]byte)
	if !ok {
		return fmt.Errorf("invalid remote address type")
	}
	fmt.Println("preq: ", hex.EncodeToString(preq))
	fmt.Println("pres: ", hex.EncodeToString(pres))
	fmt.Println("la: ", hex.EncodeToString(la))

	c1, err := smpC1(make([]byte, 16), r, preq, pres,
		la[6],
		ra[6],
		la[:6],
		ra[:6],
		)
	if err != nil {
		return err
	}

	out := append([]byte{pairingConfirm}, c1...)
	return c.sendSMP(pdu(out))
}

func (c *Conn) smpSendMRandom() error {
	r, ok := c.pairing.localRandom.([]byte)
	if !ok {
		return fmt.Errorf("invalid type for local random")
	}
	out := append([]byte{pairingRandom}, r...)
	return c.sendSMP(pdu(out))
}
