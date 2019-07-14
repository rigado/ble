package hci

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	encryptionInformation:   smpDispatcher{"encryption info", smpOnEncryptionInformation},
	masterIdentification:    smpDispatcher{"master id", smpOnMasterIdentification},
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
		initKeyDist: 0,
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

	if !isLegacy(rx.authReq) {
		//secure connections
		//send pub key
		return nil, c.smpSendPublicKey()
	} else {
		//legacy pairing
		c.pairing.legacy = true
		return nil, c.smpSendMConfirm(rx)
	}
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

	if c.pairing.legacy {
		return nil, c.smpSendMRandom()
	} else {
		return nil, c.smpSendPairingRandom()
	}
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
	if c.pairing.legacy {
		err := c.pairing.checkLegacyConfirm()
		if err != nil {
			return nil, err
		}

		fmt.Println("remote confirm OK!")

		lRand, ok := c.pairing.localRandom.([]byte)
		if !ok {
			return nil, fmt.Errorf("invalid type for local random")
		}

		rRand := []byte(in)

		//calculate STK
		stk, err := smpS1(make([]byte, 16), rRand, lRand)
		c.pairing.stk = stk

		return nil, c.encrypt()
	}

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
	//try to encrypt
	//todo: this is very hacky
	err := c.EnableEncryption()
	if err != nil {
		fmt.Print(err)
		_ = c.Bond()
	}

	return nil, nil
}

type bondInfo struct {
	Bonds []remoteKeyInfo `json:"bonds"`
}

type remoteKeyInfo struct {
	Address string `json:"address"`
	LongTermKey string `json:"longTermKey"`
	EncryptionDiversifier string `json:"encryptionDiversifier"`
	RandomValue string `json:"randomValue"`
}

/*todo: this is a bit of a hack at the moment
	usually, enc info and master id come back to back
	should probably write ltk to file and then the ediv and rand
	after they are received */
var rki = remoteKeyInfo{}
func smpOnEncryptionInformation(c *Conn, in pdu) ([]byte, error) {
	//need to save the ltk, ediv, and rand to a file
	rki.Address = hex.EncodeToString(c.pairing.remoteAddr.([]byte)[:6])
	rki.LongTermKey = hex.EncodeToString([]byte(in))

	fmt.Print("got LTK message")
	return nil, nil
}

func smpOnMasterIdentification(c *Conn, in pdu) ([]byte, error) {
	fmt.Print("got master id message")
	data := []byte(in)
	rki.EncryptionDiversifier = hex.EncodeToString(data[:2])
	rki.RandomValue = hex.EncodeToString(data[2:])

	//todo: move this somewhere more useful

	//open local file
	bondFile := filepath.Join(os.Getenv("SNAP_DATA"), "bonds.json")
	_, err := os.Stat(bondFile)
	var f *os.File
	if os.IsNotExist(err) {
		f, err = os.Create(bondFile)
		if err != nil {
			return nil, fmt.Errorf("unable to create bond file: %s",err)
		}
		_ = f.Close()
	}

	fileData, err := ioutil.ReadFile(bondFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read bond file information: %s", err)
	}

	var bonds bondInfo
	if len(fileData) > 0 {
		err = json.Unmarshal(fileData, &bonds)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal current bond info: %s", err)
		}
	}

	if len(bonds.Bonds) == 0 {
		bonds.Bonds = make([]remoteKeyInfo, 0, 1)
	}

	bonds.Bonds = append(bonds.Bonds, rki)

	out, err := json.Marshal(bonds)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bonds to json: %s", err)
	}

	err = ioutil.WriteFile(bondFile, out, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to update bond information: %s", err)
	}

	//todo: send central LTK??

	//todo: return something useful
	return nil, nil
}
