package smp

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/go-ble/ble/linux/hci"
	"log"
)

func buildPairingReq(p hci.SmpConfig) []byte {
	return []byte{pairingRequest, p.IoCap, p.OobFlag, p.AuthReq, p.MaxKeySize, p.InitKeyDist, p.RespKeyDist}
}

func buildPairingRsp(p hci.SmpConfig) []byte {
	return []byte{pairingResponse, p.IoCap, p.OobFlag, p.AuthReq, p.MaxKeySize, p.InitKeyDist, p.RespKeyDist}
}

type transport struct {
	pairing     *pairingContext
	writePDU    func([]byte) (int, error)
	bondManager hci.BondManager
	encrypter   hci.Encrypter
}

func NewSmpTransport(ctx *pairingContext, bm hci.BondManager, e hci.Encrypter, writePDU func([]byte) (int, error)) *transport {
	return &transport{ctx, writePDU, bm, e}
}

func (t *transport) SetContext(ctx *pairingContext) {
	t.pairing = ctx
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
	la = swapBuf(la)

	cmd := buildPairingReq(t.pairing.request)
	return t.send(cmd)
}

func (t *transport) sendPublicKey() error {
	kp := t.pairing.scECDHKeys

	k := MarshalPublicKeyXY(kp.public)
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

	out := append([]byte{pairingRandom}, t.pairing.localRandom...)

	return t.send(out)
}

func (t *transport) sendDHKeyCheck() error {
	if t.pairing == nil {
		return fmt.Errorf("no pairing context")
	}

	log.Printf("send dhkey check")
	p := t.pairing

	//Ea = f6 (MacKey, Na, Nb, 0, IOcapA, A, B)
	la := append(p.localAddr, p.localAddrType)
	ra := append(p.remoteAddr, p.remoteAddrType)
	na := p.localRandom
	nb := p.remoteRandom

	ioCap := swapBuf([]byte{t.pairing.request.AuthReq, t.pairing.request.OobFlag, t.pairing.request.IoCap})

	ea, err := smpF6(t.pairing.scMacKey, na, nb, make([]byte, 16), ioCap, la, ra)
	if err != nil {
		return err
	}

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

	c1, err := smpC1(make([]byte, 16), r, preq, pres,
		lat,
		rat,
		la,
		ra,
	)
	if err != nil {
		return err
	}

	out := append([]byte{pairingConfirm}, c1...)
	return t.send(out)
}
