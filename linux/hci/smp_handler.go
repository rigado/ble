package hci

import (
	"encoding/hex"
	"fmt"
)

type smpConfig struct {
	ioCap, oobFlag, authReq, maxKeySz, initKeyDist, respKeyDist byte
}

type smpState struct {
	config smpConfig
	keys   *Keys
}

var smp *smpState

var dispatcher = map[byte]smpDispatcher{
	pairingRequest:          smpDispatcher{"pairing request", smpOnPairingRequest},
	pairingResponse:         smpDispatcher{"pairing response", smpOnPairingResponse},
	pairingConfirm:          smpDispatcher{"pairing confirm", smpOnPairingConfirm},
	pairingRandom:           smpDispatcher{"pairing random", smpOnPairingRandom},
	pairingFailed:           smpDispatcher{"pairing failed", smpOnPairingFailed},
	encryptionInformation:   smpDispatcher{"encryption info", nil},
	masterIdentification:    smpDispatcher{"master id", nil},
	identityInformation:     smpDispatcher{"id info", nil},
	identityAddrInformation: smpDispatcher{"id addr info", nil},
	signingInformation:      smpDispatcher{"signing info", nil},
	securityRequest:         smpDispatcher{"security req", smpOnSecurityRequest},
	pairingPublicKey:        smpDispatcher{"pairing pub key", smpOnPairingPublicKey},
	pairingDHKeyCheck:       smpDispatcher{"pairing dhkey check", smpOnDHKeyCheck},
	pairingKeypress:         smpDispatcher{"pairing keypress", nil},
}

func SmpInit() error {
	c := smpConfig{
		ioCap:       0x03,          //no input/output
		oobFlag:     0,             //no oob
		authReq:     (1<<0 | 1<<3), //bond+sc
		maxKeySz:    16,
		respKeyDist: 1,
	}

	k, err := GenerateKeys()
	if err != nil {
		return err
	}

	smp = &smpState{c, k}
	return nil
}

func smpOnPairingRequest(c *Conn, in pdu) ([]byte, error) {
	if len(in) < 6 {
		return nil, fmt.Errorf("%v, invalid length %v", hex.EncodeToString(in), len(in))
	}

	rx := smpConfig{}
	rx.ioCap = in[0]
	rx.oobFlag = in[1]
	rx.authReq = in[2]
	rx.maxKeySz = in[3]
	rx.initKeyDist = in[4]
	rx.respKeyDist = in[5]

	fmt.Printf("pair req: %+v\n", rx)

	//reply with pairing resp

	return nil, nil
}

func smpOnPairingResponse(c *Conn, in pdu) ([]byte, error) {
	if len(in) < 6 {
		return nil, fmt.Errorf("%v, invalid length %v", hex.EncodeToString(in), len(in))
	}

	rx := smpConfig{}
	rx.ioCap = in[0]
	rx.oobFlag = in[1]
	rx.authReq = in[2]
	rx.maxKeySz = in[3]
	rx.initKeyDist = in[4]
	rx.respKeyDist = in[5]
	fmt.Printf("pair rsp: %+v\n", rx)

	//send pub key
	return nil, c.smpSendPublicKey()

}

func smpOnPairingConfirm(c *Conn, in pdu) ([]byte, error) {
	if c.pairing == nil {
		return nil, fmt.Errorf("no pairing context")
	}

	if len(in) != 16 {
		return nil, fmt.Errorf("invalid length")
	}

	fmt.Println("pairing confirm:", hex.EncodeToString(in))
	c.pairing.remoteConfirm = []byte(in)

	return nil, c.smpSendPairingRandom()
}

func smpOnPairingRandom(c *Conn, in pdu) ([]byte, error) {
	if c.pairing == nil {
		return nil, fmt.Errorf("no pairing context")
	}

	if len(in) != 16 {
		return nil, fmt.Errorf("invalid length")
	}

	fmt.Println("pairing random:", hex.EncodeToString(in))
	c.pairing.remoteRandom = []byte(in)

	//conf check
	err := c.pairing.checkConfirm()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println("pairing confirm ok!")

	// TODO
	// here we would do the compare from g2(...) but this is just works only for now
	// move on to auth stage 2 (2.3.5.6.5) calc mackey, ltk
	err = c.pairing.calcMacLtk()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println("mac ltk ok!")

	//send dhkey check
	err = c.smpSendDHKeyCheck()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return nil, nil
}

func smpOnPairingPublicKey(c *Conn, in pdu) ([]byte, error) {
	if c.pairing == nil {
		return nil, fmt.Errorf("no pairing context")
	}

	if len(in) != 64 {
		return nil, fmt.Errorf("invalid length")
	}

	pubk, ok := UnmarshalPublicKey(in)

	if !ok {
		return nil, fmt.Errorf("key error")
	}

	c.pairing.remotePubKey = pubk
	return nil, nil
}

func smpOnDHKeyCheck(c *Conn, in pdu) ([]byte, error) {
	if c.pairing == nil {
		return nil, fmt.Errorf("no pairing context")
	}

	fmt.Println("dhkey check")

	//todo: checkDHKeyCheck not implemented
	c.pairing.remoteDHKeyCheck = []byte(in)
	err := c.pairing.checkDHKeyCheck()
	if err != nil {
		//dhkeycheck failed!
		return nil, err
	}

	//encrypt!
	err = c.encrypt()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func smpOnPairingFailed(c *Conn, in pdu) ([]byte, error) {
	fmt.Println("pairing failed")
	c.pairing = nil
	return nil, nil
}

func smpOnSecurityRequest(c *Conn, in pdu) ([]byte, error) {
	if len(in) < 1 {
		return nil, fmt.Errorf("%v, invalid length %v", hex.EncodeToString(in), len(in))
	}

	// do something...
	rx := smpConfig{}
	rx.authReq = in[0]

	fmt.Printf("sec req: %+v\n", rx)

	// if known, encrypt, otherwise pair

	return nil, nil
}
