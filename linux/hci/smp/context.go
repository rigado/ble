package smp

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"fmt"
	"github.com/go-ble/ble/linux/hci"
)

type pairingContext struct {
	request hci.SmpConfig
	response hci.SmpConfig
	remoteAddr []byte
	remoteAddrType byte
	remoteRandom []byte
	remoteConfirm []byte

	localAddr []byte
	localAddrType byte
	localRandom []byte
	localConfirm []byte

	scECDHKeys *ECDHKeys
	scMacKey []byte
	scRemotePubKey crypto.PublicKey
	scDHKey []byte
	scRemoteDHKeyCheck []byte

	legacy bool
	shortTermKey []byte

	bond hci.BondInfo
}

func (p *pairingContext) checkConfirm() error {
	if p == nil {
		return fmt.Errorf("context nil")
	}

	//Cb =f4(PKbx,PKax, Nb, 0 )
	// make the keys work as expected
	kbx := MarshalPublicKeyX(p.scRemotePubKey)
	kax := MarshalPublicKeyX(p.scECDHKeys.public)
	nb := p.remoteRandom

	calcConf, err := smpF4(kbx, kax, nb, 0)
	if err != nil {
		return err
	}

	if !bytes.Equal(calcConf, p.remoteConfirm) {
		return fmt.Errorf("confirm mismatch, exp %v got %v",
			hex.EncodeToString(p.remoteConfirm), hex.EncodeToString(calcConf))
	}

	return nil
}

func (p *pairingContext) calcMacLtk() error {
	err := p.generateDHKey()
	if err != nil {
		return err
	}

	// MacKey || LTK = f5(DHKey, N_master, N_slave, BD_ADDR_master,BD_ADDR_slave)
	la := p.localAddr
	la = append(la, p.localAddrType)
	ra := p.remoteAddr
	ra = append(ra, p.remoteAddrType)
	na := p.localRandom
	nb := p.remoteRandom

	mk, ltk, err := smpF5(p.scDHKey, na, nb, la, ra)
	if err != nil {
		return err
	}

	p.bond = hci.NewBondInfo(ltk, 0, 0, false)
	p.scMacKey = mk

	return nil
}

func (p *pairingContext) checkDHKeyCheck() error {

	return nil
}

func (p *pairingContext) generateDHKey() error {
	if p == nil || p.scECDHKeys == nil {
		return fmt.Errorf("nil keys")
	}

	if p.scRemotePubKey == nil {
		return fmt.Errorf("missing remote public key")
	}

	prv := p.scECDHKeys.private

	dk, err := GenerateSecret(prv, p.scRemotePubKey)
	if err != nil {
		return err
	}
	p.scDHKey = dk
	return nil
}

func (p *pairingContext) checkLegacyConfirm() error {
	preq := buildPairingReq(p.request)
	pres := buildPairingRsp(p.response)
	la := p.localAddr
	ra:= p.remoteAddr
	sRand:= p.remoteRandom

	c1, err := smpC1(make([]byte, 16), sRand, preq, pres,
		p.localAddrType,
		p.remoteAddrType,
		la,
		ra,
	)
	if err != nil {
		return err
	}

	sConfirm:= p.remoteConfirm

	if !bytes.Equal(sConfirm, c1) {
		return fmt.Errorf("sConfirm does not match: exp %s calc %s",
			hex.EncodeToString(sConfirm), hex.EncodeToString(c1))
	}

	return nil
}