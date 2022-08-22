package smp

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/rigado/ble"
	"github.com/rigado/ble/linux/hci"
	"github.com/rigado/ble/sliceops"
)

const (
	JustWorks = iota
	NumericComp
	Passkey
	Oob
)

var pairingTypeStrings = map[int]string{
	JustWorks:   "Just Works",
	NumericComp: "Numeric Comparison",
	Passkey:     "Passkey Entry",
	Oob:         "OOB Data",
}

type pairingContext struct {
	authData     ble.AuthData
	localConfirm []byte

	remoteAddr    []byte
	remoteRandom  []byte
	remoteConfirm []byte

	localAddr          []byte
	localRandom        []byte
	scRemoteDHKeyCheck []byte

	shortTermKey []byte

	scMacKey []byte
	scDHKey  []byte
	bond     hci.BondInfo

	ble.Logger
	scRemotePubKey crypto.PublicKey

	scECDHKeys *ECDHKeys

	passKeyIteration int

	pairingType int
	state       PairingState
	request     hci.SmpConfig
	response    hci.SmpConfig

	legacy         bool
	localAddrType  byte
	remoteAddrType byte
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

func (p *pairingContext) checkPasskeyConfirm() error {
	// make the keys work as expected
	kbx := MarshalPublicKeyX(p.scRemotePubKey)
	kax := MarshalPublicKeyX(p.scECDHKeys.public)
	nb := p.remoteRandom
	i := p.passKeyIteration
	key := p.authData.Passkey

	//this gets the bit of the passkey for the current iteration
	z := 0x80 | (byte)((key&(1<<uint(i)))>>uint(i))

	//Cb =f4(PKbx,PKax, Nb, rb)
	calcConf, err := smpF4(kbx, kax, nb, z)
	if err != nil {
		return err
	}

	//p.Debugf("i: %d, z: %x, c: %v, cc: %v, ra: %v, rb: %v", iteration, z,
	//	hex.EncodeToString(p.remoteConfirm),
	//	hex.EncodeToString(calcConf),
	//	hex.EncodeToString(p.localRandom),
	//	hex.EncodeToString(p.remoteRandom))

	if !bytes.Equal(p.remoteConfirm, calcConf) {
		return fmt.Errorf("passkey confirm mismatch %d, exp %v got %v",
			i, hex.EncodeToString(p.remoteConfirm), hex.EncodeToString(calcConf))
	}

	return nil
}

// todo: key should be set at the beginning
func (p *pairingContext) generatePassKeyConfirm() ([]byte, []byte) {
	kbx := MarshalPublicKeyX(p.scRemotePubKey)
	kax := MarshalPublicKeyX(p.scECDHKeys.public)
	nai := make([]byte, 16)
	_, err := rand.Read(nai)
	if err != nil {

	}

	i := p.passKeyIteration
	z := 0x80 | (byte)((p.authData.Passkey&(1<<uint(i)))>>uint(i))

	calcConf, err := smpF4(kax, kbx, nai, z)
	if err != nil {
		p.Errorf("generatePasskeyConfirm: %v", err)
	}

	//p.Debugf("passkey confirm %d: z: %x, conf: %v", iteration, z, hex.EncodeToString(calcConf))

	return calcConf, nai
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
	//F6(MacKey, Na, Nb, ra, IOcapA, A, B)
	la := p.localAddr
	la = append(la, p.localAddrType)
	rAddr := p.remoteAddr
	rAddr = append(rAddr, p.remoteAddrType)
	na := p.localRandom
	nb := p.remoteRandom

	ioCap := sliceops.SwapBuf([]byte{p.response.AuthReq, p.response.OobFlag, p.response.IoCap})

	ra := make([]byte, 16)
	if p.pairingType == Passkey {
		keyBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(keyBytes, uint32(p.authData.Passkey))
		ra[12] = keyBytes[0]
		ra[13] = keyBytes[1]
		ra[14] = keyBytes[2]
		ra[15] = keyBytes[3]

		//swap to little endian
		ra = sliceops.SwapBuf(ra)
	} else if p.pairingType == Oob {
		ra = p.authData.OOBData
		//todo: does this need to be swapped?
	}

	dhKeyCheck, err := smpF6(p.scMacKey, nb, na, ra, ioCap, rAddr, la)
	if err != nil {
		return err
	}

	if !bytes.Equal(p.scRemoteDHKeyCheck, dhKeyCheck) {
		return fmt.Errorf("dhKeyCheck failed: expected %x, calculated %x",
			p.scRemoteDHKeyCheck, dhKeyCheck)
	}

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
	ra := p.remoteAddr
	sRand := p.remoteRandom

	k := make([]byte, 16)
	if p.pairingType == Passkey {
		k = getLegacyParingTK(p.authData.Passkey)
	}
	c1, err := smpC1(k, sRand, preq, pres,
		p.localAddrType,
		p.remoteAddrType,
		la,
		ra,
	)
	if err != nil {
		return err
	}

	sConfirm := p.remoteConfirm

	if !bytes.Equal(sConfirm, c1) {
		return fmt.Errorf("sConfirm does not match: exp %s calc %s",
			hex.EncodeToString(sConfirm), hex.EncodeToString(c1))
	}

	return nil
}
