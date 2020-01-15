package smp

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/go-ble/ble/linux/hci"
)

//todo: track this state in the pairing context
var isPassKeyPairing = false
var passKeyIteration = 0
var pairingKey = 123456

//func smpOnPairingRequest(c *Conn, in pdu) ([]byte, error) {
//	if len(in) < 6 {
//		return nil, fmt.Errorf("%v, invalid length %v", hex.EncodeToString(in), len(in))
//	}
//
//	rx := smpConfig{}
//	rx.ioCap = in[0]
//	rx.oobFlag = in[1]
//	rx.authReq = in[2]
//	rx.maxKeySz = in[3]
//	rx.initKeyDist = in[4]
//	rx.respKeyDist = in[5]
//
//	return nil, nil
//}

func smpOnPairingResponse(t *transport, in pdu) ([]byte, error) {
	if len(in) < 6 {
		return nil, fmt.Errorf("%v, invalid length %v", hex.EncodeToString(in), len(in))
	}

	rx := hci.SmpConfig{}
	rx.IoCap = in[0]
	rx.OobFlag = in[1]
	rx.AuthReq = in[2]
	rx.MaxKeySize = in[3]
	rx.InitKeyDist = in[4]
	rx.RespKeyDist = in[5]
	t.pairing.response = rx

	isPassKeyPairing = false
	passKeyIteration = 0

	if isLegacy(rx.AuthReq) {
		t.pairing.legacy = true
		return nil, t.sendMConfirm()
	} else {
		//secure connections
		return nil, t.sendPublicKey()
	}
}

func smpOnPairingConfirm(t *transport, in pdu) ([]byte, error) {
	if t.pairing == nil {
		return nil, fmt.Errorf("no pairing context")
	}

	if len(in) != 16 {
		return nil, fmt.Errorf("invalid length")
	}

	t.pairing.remoteConfirm = in

	err := t.sendPairingRandom()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func smpOnPairingRandom(t *transport, in pdu) ([]byte, error) {
	if t.pairing == nil {
		return nil, fmt.Errorf("no pairing context")
	}

	if len(in) != 16 {
		return nil, fmt.Errorf("invalid length")
	}

	t.pairing.remoteRandom = []byte(in)

	//conf check
	if t.pairing.legacy {
		return onLegacyRandom(t)
	}

	return onSecureRandom(t)
}

func onSecureRandom(t *transport) ([]byte, error) {
	if isPassKeyPairing {
		err := t.pairing.checkPasskeyConfirm(passKeyIteration, pairingKey)
		if err != nil {
			return nil, err
		}

		passKeyIteration++

		if passKeyIteration < 20 {
			continuePassKeyPairing(t)
			return nil, nil
		}
	} else {
		err := t.pairing.checkConfirm()
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		fmt.Println("pairing confirm ok!")
	}

	// TODO
	// here we would do the compare from g2(...) but this is just works only for now
	// move on to auth stage 2 (2.3.5.6.5) calc mackey, ltk
	err := t.pairing.calcMacLtk()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println("mac ltk ok!")

	//send dhkey check
	err = t.sendDHKeyCheck(pairingKey)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return nil, nil
}

func onLegacyRandom(t *transport) ([]byte, error) {
	err := t.pairing.checkLegacyConfirm()
	if err != nil {
		return nil, err
	}

	fmt.Println("remote confirm OK!")

	lRand := t.pairing.localRandom
	rRand := t.pairing.remoteRandom

	//calculate STK
	stk, err := smpS1(make([]byte, 16), rRand, lRand)
	t.pairing.shortTermKey = stk

	return nil, t.encrypter.Encrypt()
}

func smpOnPairingPublicKey(t *transport, in pdu) ([]byte, error) {
	if t.pairing == nil {
		return nil, fmt.Errorf("no pairing context")
	}

	if len(in) != 64 {
		return nil, fmt.Errorf("invalid length")
	}

	pubk, ok := UnmarshalPublicKey(in)

	if !ok {
		return nil, fmt.Errorf("key error")
	}

	t.pairing.scRemotePubKey = pubk

	//check to see if we should start a passkey pairing procedure
	//this should actually be done according to the spec
	//todo: need a way to get the passkey into the library
	if t.pairing.response.IoCap == 0x00 {
		//device has a display, so lets do passkey pairing
		//todo: this check is kind of bogus
		startPassKeyPairing(t)
	}
	return nil, nil
}

func smpOnDHKeyCheck(t *transport, in pdu) ([]byte, error) {
	if t.pairing == nil {
		return nil, fmt.Errorf("no pairing context")
	}

	//todo: checkDHKeyCheck not implemented
	t.pairing.scRemoteDHKeyCheck = []byte(in)
	err := t.pairing.checkDHKeyCheck()
	if err != nil {
		//dhkeycheck failed!
		return nil, err
	}

	err = t.saveBondInfo()
	if err != nil {
		return nil, err
	}

	//encrypt!
	return nil, t.encrypter.Encrypt()
}

func smpOnPairingFailed(t *transport, in pdu) ([]byte, error) {
	fmt.Println("pairing failed")
	t.pairing = nil
	return nil, nil
}

func smpOnSecurityRequest(t *transport, in pdu) ([]byte, error) {
	if len(in) < 1 {
		return nil, fmt.Errorf("%v, invalid length %v", hex.EncodeToString(in), len(in))
	}

	ra := hex.EncodeToString(t.pairing.remoteAddr)
	bi, err := t.bondManager.Find(ra)
	fmt.Println(err)
	if err == nil {
		t.pairing.bond = bi
		return nil, t.encrypter.Encrypt()
	}

	//todo: clean this up
	rx := hci.SmpConfig{}
	rx.AuthReq = in[0]

	//match the incoming request parameters
	t.pairing.request.AuthReq = rx.AuthReq
	//no bonding information stored, so trigger a bond
	return nil, t.sendPairingRequest()
}

func smpOnEncryptionInformation(t *transport, in pdu) ([]byte, error) {
	//need to save the ltk, ediv, and rand to a file
	t.pairing.bond = hci.NewBondInfo([]byte(in), 0, 0, true)

	return nil, nil
}

func smpOnMasterIdentification(t *transport, in pdu) ([]byte, error) {
	data := []byte(in)
	ediv := binary.LittleEndian.Uint16(data[:2])
	randVal := binary.LittleEndian.Uint64(data[2:])

	ltk := t.pairing.bond.LongTermKey()
	t.pairing.bond = hci.NewBondInfo(ltk, ediv, randVal, true)

	//todo: move this somewhere more useful
	return nil, t.saveBondInfo()
}

func startPassKeyPairing(t *transport) {
	isPassKeyPairing = true
	passKeyIteration = 0

	continuePassKeyPairing(t)
}

func continuePassKeyPairing(t *transport) {
	confirm, random := t.pairing.generatePassKeyConfirm(passKeyIteration, 123456)
	t.pairing.localRandom = random
	out := append([]byte{pairingConfirm}, confirm...)
	t.send(out)
}
