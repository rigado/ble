package smp

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/rigado/ble/linux/hci"
	"github.com/rigado/ble/sliceops"
)

func buildPairingReq(p hci.SmpConfig) []byte {
	return []byte{pairingRequest, p.IoCap, p.OobFlag, p.AuthReq, p.MaxKeySize, p.InitKeyDist, p.RespKeyDist}
}

func buildPairingRsp(p hci.SmpConfig) []byte {
	return []byte{pairingResponse, p.IoCap, p.OobFlag, p.AuthReq, p.MaxKeySize, p.InitKeyDist, p.RespKeyDist}
}

type transport struct {
	pairing  *pairingContext
	writePDU func([]byte) (int, error)

	bondManager hci.BondManager
	encrypter   hci.Encrypter

	nopFunc func() error //workaround stuff

	result chan error
}

func NewSmpTransport(ctx *pairingContext, bm hci.BondManager, e hci.Encrypter, writePDU func([]byte) (int, error), nopFunc func() error) *transport {
	return &transport{ctx, writePDU, bm, e, nopFunc, make(chan error)}
}

func (t *transport) SetContext(ctx *pairingContext) {
	t.pairing = ctx
}

func (t *transport) StartPairing(to time.Duration) error {
	t.pairing.state = WaitPairingResponse
	err := t.sendPairingRequest()
	if err != nil {
		t.pairing.state = Error
		return err
	}

	return nil
}

func (t *transport) saveBondInfo() error {
	addr := hex.EncodeToString(t.pairing.remoteAddr)
	return t.bondManager.Save(addr, t.pairing.bond)
}

func (t *transport) send(pdu []byte) error {
	buf := bytes.NewBuffer(make([]byte, 0))
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(pdu))); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, hci.CidSMP); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, pdu); err != nil {
		return err
	}
	_, err := t.writePDU(buf.Bytes())
	return err
}

func (t *transport) sendPairingRequest() error {
	//todo: create a new pairing context function
	ra := t.pairing.remoteAddr
	ra = append(ra, t.pairing.remoteAddrType)

	laBE := t.pairing.localAddr
	la := make([]byte, 0, 7)
	la = append(la, t.pairing.localAddrType)
	for _, v := range laBE {
		la = append(la, v)
	}
	la = sliceops.SwapBuf(la)

	cmd := buildPairingReq(t.pairing.request)
	return t.send(cmd)
}

func (t *transport) sendPublicKey() error {
	if t.pairing.scECDHKeys == nil {
		keys, err := GenerateKeys()
		if err != nil {
			fmt.Println("error generating secure keys:", err)
		}
		t.pairing.scECDHKeys = keys
	}

	k := MarshalPublicKeyXY(t.pairing.scECDHKeys.public)

	t.pairing.state = WaitPublicKey
	out := append([]byte{pairingPublicKey}, k...)
	err := t.send(out)

	if err != nil {
		return err
	}

	return nil
}

func (t *transport) sendPairingRandom() error {
	if t.pairing == nil {
		return fmt.Errorf("no pairing context")
	}

	if t.pairing.localRandom == nil {
		r := make([]byte, 16)
		_, err := rand.Read(r)
		if err != nil {
			return err
		}
		t.pairing.localRandom = r
	}

	t.pairing.state = WaitRandom
	out := append([]byte{pairingRandom}, t.pairing.localRandom...)

	return t.send(out)
}

func (t *transport) sendDHKeyCheck() error {
	if t.pairing == nil {
		return fmt.Errorf("no pairing context")
	}

	log.Printf("send dhkey check")
	p := t.pairing

	//Ea = f6 (MacKey, Na, Nb, rb, IOcapA, A, B)
	la := append(p.localAddr, p.localAddrType)
	ra := append(p.remoteAddr, p.remoteAddrType)
	na := p.localRandom
	nb := p.remoteRandom

	ioCap := sliceops.SwapBuf([]byte{t.pairing.request.AuthReq, t.pairing.request.OobFlag, t.pairing.request.IoCap})

	rb := make([]byte, 16)
	if t.pairing.pairingType == Passkey {
		keyBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(keyBytes, uint32(t.pairing.authData.Passkey))
		rb[12] = keyBytes[0]
		rb[13] = keyBytes[1]
		rb[14] = keyBytes[2]
		rb[15] = keyBytes[3]

		//swap to little endian
		rb = sliceops.SwapBuf(rb)
	} else if t.pairing.pairingType == Oob {
		rb = t.pairing.authData.OOBData
		//todo: does this need to be swapped?
	}

	ea, err := smpF6(t.pairing.scMacKey, na, nb, rb, ioCap, la, ra)
	if err != nil {
		return err
	}

	t.pairing.state = WaitDhKeyCheck
	out := append([]byte{pairingDHKeyCheck}, ea...)
	return t.send(out)
}

func (t *transport) sendMConfirm() error {
	if t.pairing == nil {
		return fmt.Errorf("no pairing context")
	}

	preq := buildPairingReq(t.pairing.request)
	pres := buildPairingRsp(t.pairing.response)

	r := make([]byte, 16)
	_, err := rand.Read(r)
	if err != nil {
		return err
	}
	t.pairing.localRandom = r

	la := t.pairing.localAddr
	lat := t.pairing.localAddrType
	ra := t.pairing.remoteAddr
	rat := t.pairing.remoteAddrType

	k := make([]byte, 16)
	if t.pairing.pairingType == Passkey {
		k = getLegacyParingTK(t.pairing.authData.Passkey)
	}

	c1, err := smpC1(k, r, preq, pres,
		lat, rat, la, ra,
	)
	if err != nil {
		return err
	}

	t.pairing.state = WaitConfirm
	out := append([]byte{pairingConfirm}, c1...)
	return t.send(out)
}
