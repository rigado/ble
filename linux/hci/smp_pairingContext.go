package hci

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"fmt"
	"reflect"
)

type pairingContext struct {
	remoteAddr       interface{}
	remotePubKey     interface{}
	remoteConfirm    interface{}
	remoteRandom     interface{}
	remoteDHKeyCheck interface{}
	remoteMacKey     interface{}

	localKeys       *Keys
	localAddr       interface{}
	localConfirm    interface{}
	localRandom     interface{}
	localDHKeyCheck interface{}

	dhkey  []byte
	ltk    []byte
	macKey []byte
}

func (p *pairingContext) checkConfirm() error {
	if p == nil {
		return fmt.Errorf("context nil")
	}

	//Cb =f4(PKbx,PKax, Nb, 0 )
	expConf, ok := p.remoteConfirm.([]byte)
	if !ok {
		return fmt.Errorf("remoteConfirm type error, %v", reflect.TypeOf(p.remoteConfirm))
	}

	// make the keys work as expected
	kbx := MarshalPublicKeyX(p.remotePubKey.(crypto.PublicKey))
	kax := MarshalPublicKeyX(p.localKeys.public)
	nb := p.remoteRandom.([]byte)

	calcConf, err := smpF4(kbx, kax, nb, 0)
	if err != nil {
		return err
	}

	if !bytes.Equal(calcConf, expConf) {
		return fmt.Errorf("confirm mismatch, exp %v got %v", hex.EncodeToString(expConf), hex.EncodeToString(calcConf))
	}

	return nil
}

func (p *pairingContext) calcMacLtk() error {
	err := p.generateDHKey()
	if err != nil {
		return err
	}

	// MacKey || LTK = f5(DHKey, N_master, N_slave, BD_ADDR_master,BD_ADDR_slave)
	la := p.localAddr.([]byte)
	ra := p.remoteAddr.([]byte)
	na := p.localRandom.([]byte)
	nb := p.remoteRandom.([]byte)

	mk, ltk, err := smpF5(p.dhkey, na, nb, la, ra)
	if err != nil {
		return err
	}

	p.ltk = ltk
	p.macKey = mk

	fmt.Printf("mac ltk ok, %+v\n", *p)
	return nil
}

func (p *pairingContext) checkDHKeyCheck() error {

	return nil
}

func (p *pairingContext) generateDHKey() error {
	if p == nil || p.localKeys == nil {
		return fmt.Errorf("nil keys")
	}
	pub, ok := p.remotePubKey.(crypto.PublicKey)
	if !ok {
		return fmt.Errorf("type error")
	}
	prv := p.localKeys.private

	dk, err := GenerateSecret(prv, pub)
	if err != nil {
		return err
	}
	p.dhkey = dk
	return nil
}
