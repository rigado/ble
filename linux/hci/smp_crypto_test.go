package hci

import (
	"bytes"
	"crypto/aes"
	"encoding/hex"
	"testing"

	"github.com/enceve/crypto/cmac"
)

func Test_ConfirmCheck(t *testing.T) {
	// < ACL Data TX: Handle 0 flags 0x00 dlen 69                                                                                                                           #4004 7602.154267
	// SMP: Pairing Public Key (0x0c) len 64
	// X: 2924dce60c38fdffe4bfa07134ea4cf238904695d7b8512b7c73ad3af2d1e789
	// Y: b9b7293371c2ede8cec34a8d2de8038bacac3b520fbb52c53aefe2c67e8b3661
	// > HCI Event: Number of Completed Packets (0x13) plen 5                                                                                                               #4005 7602.162356
	// Num handles: 1
	// Handle: 0
	// Count: 1
	// > ACL Data RX: Handle 0 flags 0x02 dlen 69                                                                                                                           #4006 7602.169384
	// SMP: Pairing Public Key (0x0c) len 64
	// X: 88287228a0d516fa458abc3a3264a0db65a92b8e8a53343e866eaed4b461b9c5
	// Y: 47fee8404d3a3a753e17a759ed747b7458bc5452bd4c8e69c636eeda851fb3a8
	// > ACL Data RX: Handle 0 flags 0x02 dlen 21                                                                                                                           #4007 7602.169405
	// SMP: Pairing Confirm (0x03) len 16
	// Confim value: a6c760d1be58d9b859e9823df9ab1c97
	// < ACL Data TX: Handle 0 flags 0x00 dlen 21                                                                                                                           #4008 7602.171134
	// SMP: Pairing Random (0x04) len 16
	// Random value: 86cebc859da1d2b9a0d408210db4c986
	// > HCI Event: Number of Completed Packets (0x13) plen 5                                                                                                               #4009 7602.175330
	// Num handles: 1
	// Handle: 0
	// Count: 1
	// > ACL Data RX: Handle 0 flags 0x02 dlen 21                                                                                                                           #4010 7602.183332
	// SMP: Pairing Random (0x04) len 16
	// Random value: e194607e5c588d24e6e22b5470f0b3c3

	//helper func
	s2h := func(swap bool, s string) []byte {
		b, err := hex.DecodeString(s)
		if err != nil {
			t.Fatal("s2h error!")
		}

		if swap {
			return swapBuf(b)
		}
		return b
	}

	lxy := s2h(false, "2924dce60c38fdffe4bfa07134ea4cf238904695d7b8512b7c73ad3af2d1e789b9b7293371c2ede8cec34a8d2de8038bacac3b520fbb52c53aefe2c67e8b3661")
	rxy := s2h(false, "88287228a0d516fa458abc3a3264a0db65a92b8e8a53343e866eaed4b461b9c547fee8404d3a3a753e17a759ed747b7458bc5452bd4c8e69c636eeda851fb3a8")

	rrand := s2h(false, "e194607e5c588d24e6e22b5470f0b3c3")
	rconf := s2h(false, "a6c760d1be58d9b859e9823df9ab1c97")

	// expLtk := s2h(false, "f56269cf8c376f2e8862a5d5275bf0ba")

	// var testZ = uint8(0x00)

	lk, ok := UnmarshalPublicKey(lxy)
	if !ok {
		t.Log("key err", lk)
	}
	rk, ok := UnmarshalPublicKey(rxy)
	if !ok {
		t.Log("key err", rk)
	}

	MarshalPublicKeyXY(lk)
	MarshalPublicKeyXY(rk)

	c := pairingContext{
		localKeys: &Keys{public: lk},
		// localRandom:   s2h("86cebc859da1d2b9a0d408210db4c986"),
		remoteConfirm: rconf,
		remoteRandom:  rrand,
		remotePubKey:  rk,
		// localPubKey:   lx,
	}

	err := c.checkConfirm()
	if err != nil {
		t.Fatal(err)
	}
}

// func Test_DHKeyCheck(t *testing.T) {

// 	//does not work
// 	SmpInit()

// 	//helper func
// 	s2h := func(swap bool, s string) []byte {
// 		b, err := hex.DecodeString(s)
// 		if err != nil {
// 			t.Fatal("s2h error!")
// 		}

// 		if swap {
// 			return swapBuf(b)
// 		}
// 		return b
// 	}

// 	// lpk := s2h(false, "")
// 	lxy := s2h(false, "b65b4000dfe0a771f8374fd385f65af2b0660c47d7de55e4dd04d38b9504da2e4088842c5ba6984150cb60e9948b3200e2f7d26310b9515c66353d99581b10df")
// 	rxy := s2h(false, "b6c75a2f3f6893933dd454285f9077c0df96a0b1e2df98fa6767d6d757ee29c15505c9debc1d9b2641f7f4aeb129bcf11707a155b98f43f77b2e6e041bba137d")

// 	lrand := s2h(false, "64ad15c69f9a7430c43abe7bd6cccbda")
// 	rrand := s2h(false, "b3fb312bd2941f21484564dd6d25fe29")
// 	rconf := s2h(false, "36257146a0097de0c50ab1d39cb5712f")

// 	expLtk := s2h(false, "f56269cf8c376f2e8862a5d5275bf0ba")

// 	// var testZ = uint8(0x00)

// 	lk, ok := UnmarshalPublicKey(lxy)
// 	if !ok {
// 		t.Log("key err", lk)
// 	}
// 	rk, ok := UnmarshalPublicKey(rxy)
// 	if !ok {
// 		t.Log("key err", rk)
// 	}

// 	MarshalPublicKeyXY(lk)
// 	MarshalPublicKeyXY(rk)

// 	c := pairingContext{
// 		localKeys:     &Keys{public: lk},
// 		localRandom:   lrand,
// 		remoteConfirm: rconf,
// 		remoteRandom:  rrand,
// 		remotePubKey:  rk,
// 		// localPubKey:   lx,
// 	}

// 	err := c.checkConfirm()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = c.calcMacLtk()
// 	if err != nil {
// 		t.Fatal()
// 	}

// 	if !bytes.Equal(c.ltk, expLtk) {
// 		t.Fatal(hex.EncodeToString(c.ltk), hex.EncodeToString(expLtk))
// 	}
// }

// func Test_DHKeyGenerate(t *testing.T) {
// 	// 	<info> nrf_ble_lesc: private keys
// 	// 2DE59161C05BBC0F0DA476E85B2D4B8F512F9690F5FF63985233955FA1D3E505
// 	// <info> nrf_ble_lesc: public keys
// 	// A452F050425EA025776511E43CD1F2276C53953A7BF27046E6E0C23ABA9271DFE9EE5F2E334CD90673331D84635E76574F14052630FC58572D2454B547FDC228
// 	// <info> app: BLE_GAP_EVT_LESC_DHKEY_REQUEST
// 	// <info> nrf_ble_lesc: Calling sd_ble_gap_lesc_dhkey_reply on conn_handle: 0
// 	// 93796F44E2963CE0176190A5A65AA883E4D6ADEEAC51FBA46507774E8AE84BDC

// 	// ###### flip the roles here for remote/local
// 	// < ACL Data TX: Handle 0 flags 0x00 dlen 69                                                            #3897 3193.185557
// 	//       SMP: Pairing Public Key (0x0c) len 64
// 	//         X: aa5a22757540e76ec8ef14f5197a39d8274b7252b322cf88942e78c6aa4bcf8f
// 	//         Y: 9ac358f1dc593a2af6baef246d29a1c9ad80cd7539f09bef76cd21e1e59d93a8
// 	// > HCI Event: Number of Completed Packets (0x13) plen 5                                                #3898 3193.230402
// 	//         Num handles: 1
// 	//         Handle: 0
// 	//         Count: 1

// 	// > ACL Data RX: Handle 0 flags 0x02 dlen 69                                                            #3899 3193.280714
// 	//       SMP: Pairing Public Key (0x0c) len 64
// 	//         X: a452f050425ea025776511e43cd1f2276c53953a7bf27046e6e0c23aba9271df
// 	//         Y: e9ee5f2e334cd90673331d84635e76574f14052630fc58572d2454b547fdc228

// 	//helper func
// 	s2h := func(swap bool, s string) []byte {
// 		b, err := hex.DecodeString(s)
// 		if err != nil {
// 			t.Fatal("s2h error!")
// 		}

// 		if swap {
// 			return swapBuf(b)
// 		}
// 		return b
// 	}

// 	lprvk := s2h(false, "2DE59161C05BBC0F0DA476E85B2D4B8F512F9690F5FF63985233955FA1D3E505")
// 	rxy := s2h(false, "aa5a22757540e76ec8ef14f5197a39d8274b7252b322cf88942e78c6aa4bcf8f9ac358f1dc593a2af6baef246d29a1c9ad80cd7539f09bef76cd21e1e59d93a8")
// 	lprvkk, ok := ecdh.UnmarshalPrivateKey(lprvk)
// 	if !ok {
// 		t.Log("prv key err")
// 	}

// 	rk, ok := UnmarshalPublicKey(rxy)
// 	if !ok {
// 		t.Log("key err", rk)
// 	}

// 	MarshalPublicKeyXY(rk)

// 	c := pairingContext{
// 		localKeys:    &Keys{private: lprvkk},
// 		remotePubKey: rk,
// 	}

// 	expdhk := s2h(false, "93796F44E2963CE0176190A5A65AA883E4D6ADEEAC51FBA46507774E8AE84BDC")
// 	err := c.generateDHKey()
// 	if err != nil {
// 		t.Fatal()
// 	}

// 	if !bytes.Equal(c.dhkey, expdhk) {
// 		t.Fatalf("\ngot %v\nexp %v", c.dhkey, expdhk)
// 	}
// }

func Test_f5(t *testing.T) {
	s2h := func(s string) []byte {
		b, err := hex.DecodeString(s)
		if err != nil {
			t.Fatal("s2h error!")
		}
		return b
	}

	na := s2h("fa9d22d0f2ecfbf7960a76aa9925f18f")
	nb := s2h("b30214a4b530db3fcb65e88164321de2")
	aa := []byte{0x94, 0x54, 0x93, 0x93, 0x54, 0x94}
	a := append(aa, 0)
	bb := []byte{0x32, 0x49, 0xba, 0x7a, 0x74, 0xc5}
	b := append(bb, 1)
	dhk := s2h("93796F44E2963CE0176190A5A65AA883E4D6ADEEAC51FBA46507774E8AE84BDC")

	mk, ltk, err := smpF5(dhk, na, nb, a, b)
	if err != nil {
		t.Fatal()
	}

	eltk := s2h("3ea2200172d747c1102854108cfcda87")
	if !bytes.Equal(eltk, ltk) {
		t.Fatalf("\ngot %v\nexp %v", hex.EncodeToString(ltk), hex.EncodeToString(eltk))
	}

	t.Log(mk, ltk)
}

func Test_SMP_Crypto(t *testing.T) {

	const key = "2b7e151628aed2a6abf7158809cf4f3c"

	var testU = []byte{
		0xe6, 0x9d, 0x35, 0x0e, 0x48, 0x01, 0x03, 0xcc,
		0xdb, 0xfd, 0xf4, 0xac, 0x11, 0x91, 0xf4, 0xef,
		0xb9, 0xa5, 0xf9, 0xe9, 0xa7, 0x83, 0x2c, 0x5e,
		0x2c, 0xbe, 0x97, 0xf2, 0xd2, 0x03, 0xb0, 0x20,
	}

	var testV = []byte{
		0xfd, 0xc5, 0x7f, 0xf4, 0x49, 0xdd, 0x4f, 0x6b,
		0xfb, 0x7c, 0x9d, 0xf1, 0xc2, 0x9a, 0xcb, 0x59,
		0x2a, 0xe7, 0xd4, 0xee, 0xfb, 0xfc, 0x0a, 0x90,
		0x9a, 0xbb, 0xf6, 0x32, 0x3d, 0x8b, 0x18, 0x55,
	}

	var testX = []byte{
		0xab, 0xae, 0x2b, 0x71, 0xec, 0xb2, 0xff, 0xff,
		0x3e, 0x73, 0x77, 0xd1, 0x54, 0x84, 0xcb, 0xd5,
	}

	var testZ = uint8(0x00)

	var testExpF4 = []byte{
		0x2d, 0x87, 0x74, 0xa9, 0xbe, 0xa1, 0xed, 0xf1,
		0x1c, 0xbd, 0xa9, 0x07, 0xf1, 0x16, 0xc9, 0xf2,
	}

	var testW = []byte{
		0x98, 0xa6, 0xbf, 0x73, 0xf3, 0x34, 0x8d, 0x86,
		0xf1, 0x66, 0xf8, 0xb4, 0x13, 0x6b, 0x79, 0x99,
		0x9b, 0x7d, 0x39, 0x0a, 0xa6, 0x10, 0x10, 0x34,
		0x05, 0xad, 0xc8, 0x57, 0xa3, 0x34, 0x02, 0xec}
	var testN1 = []byte{
		0xab, 0xae, 0x2b, 0x71, 0xec, 0xb2, 0xff, 0xff,
		0x3e, 0x73, 0x77, 0xd1, 0x54, 0x84, 0xcb, 0xd5}
	var testN2 = []byte{
		0xcf, 0xc4, 0x3d, 0xff, 0xf7, 0x83, 0x65, 0x21,
		0x6e, 0x5f, 0xa7, 0x25, 0xcc, 0xe7, 0xe8, 0xa6}
	var testA1 = []byte{0xce, 0xbf, 0x37, 0x37, 0x12, 0x56, 0x00}
	var testA2 = []byte{0xc1, 0xcf, 0x2d, 0x70, 0x13, 0xa7, 0x00}
	var testExpLTK = []byte{
		0x38, 0x0a, 0x75, 0x94, 0xb5, 0x22, 0x05, 0x98,
		0x23, 0xcd, 0xd7, 0x69, 0x11, 0x79, 0x86, 0x69}
	var testExpMACKey = []byte{
		0x20, 0x6e, 0x63, 0xce, 0x20, 0x6a, 0x3f, 0xfd,
		0x02, 0x4a, 0x08, 0xa1, 0x76, 0xf1, 0x65, 0x29}

	var testWF6 = []byte{
		0x20, 0x6e, 0x63, 0xce, 0x20, 0x6a, 0x3f, 0xfd,
		0x02, 0x4a, 0x08, 0xa1, 0x76, 0xf1, 0x65, 0x29}
	var testN1F6 = []byte{
		0xab, 0xae, 0x2b, 0x71, 0xec, 0xb2, 0xff, 0xff,
		0x3e, 0x73, 0x77, 0xd1, 0x54, 0x84, 0xcb, 0xd5}
	var testN2F6 = []byte{
		0xcf, 0xc4, 0x3d, 0xff, 0xf7, 0x83, 0x65, 0x21,
		0x6e, 0x5f, 0xa7, 0x25, 0xcc, 0xe7, 0xe8, 0xa6}
	var testR = []byte{
		0xc8, 0x0f, 0x2d, 0x0c, 0xd2, 0x42, 0xda, 0x08,
		0x54, 0xbb, 0x53, 0xb4, 0x3b, 0x34, 0xa3, 0x12}
	var testIoCap = []byte{0x02, 0x01, 0x01}
	var testA1F6 = []byte{0xce, 0xbf, 0x37, 0x37, 0x12, 0x56, 0x00}
	// var testA2F6 = []byte{0xc1, 0xcf, 0x2d, 0x70, 0x13, 0xa7, 0x00}
	var expF6 = []byte{
		0x61, 0x8f, 0x95, 0xda, 0x09, 0x0b, 0x6c, 0xd2,
		0xc5, 0xe8, 0xd0, 0x9c, 0x98, 0x73, 0xc4, 0xe3}

	var testUG2 = []byte{
		0xe6, 0x9d, 0x35, 0x0e, 0x48, 0x01, 0x03, 0xcc,
		0xdb, 0xfd, 0xf4, 0xac, 0x11, 0x91, 0xf4, 0xef,
		0xb9, 0xa5, 0xf9, 0xe9, 0xa7, 0x83, 0x2c, 0x5e,
		0x2c, 0xbe, 0x97, 0xf2, 0xd2, 0x03, 0xb0, 0x20}
	var testVG2 = []byte{
		0xfd, 0xc5, 0x7f, 0xf4, 0x49, 0xdd, 0x4f, 0x6b,
		0xfb, 0x7c, 0x9d, 0xf1, 0xc2, 0x9a, 0xcb, 0x59,
		0x2a, 0xe7, 0xd4, 0xee, 0xfb, 0xfc, 0x0a, 0x90,
		0x9a, 0xbb, 0xf6, 0x32, 0x3d, 0x8b, 0x18, 0x55}
	var testXG2 = []byte{
		0xab, 0xae, 0x2b, 0x71, 0xec, 0xb2, 0xff, 0xff,
		0x3e, 0x73, 0x77, 0xd1, 0x54, 0x84, 0xcb, 0xd5}
	var testYG2 = []byte{
		0xcf, 0xc4, 0x3d, 0xff, 0xf7, 0x83, 0x65, 0x21,
		0x6e, 0x5f, 0xa7, 0x25, 0xcc, 0xe7, 0xe8, 0xa6}
	var expValG2 = uint32(0x2f9ed5ba % 1000000)

	//create cmac for known message
	mStr := "6bc1bee22e409f96e93d7e117393172a"
	expMac := "070a16b46b4d4144f79bdd9dd04a287c"

	m, err := hex.DecodeString(mStr)
	if err != nil {
		t.Fatal("failed to decode mStr:", err)
	}

	k, err := hex.DecodeString(key)
	if err != nil {
		t.Fatal("failed to decode key:", err)
	}
	mCipher, err := aes.NewCipher(k)
	if err != nil {
		t.Fatal("failed to generic cipher for m:", err)
	}

	mMac, err := cmac.New(mCipher)
	if err != nil {
		t.Fatal("failed to generic cmac for m:", err)
	}

	mMac.Write(m)

	actualMac := hex.EncodeToString(mMac.Sum(nil))
	if actualMac != expMac {
		t.Fatal("actual mac doesn't match expected:", actualMac)
	}

	//test smpF4
	f4Out, err := smpF4(testU, testV, testX, testZ)
	if err != nil {
		t.Fatal("f4 calc failed:", err)
	}

	if !bytes.Equal(f4Out, testExpF4) {
		t.Fatal("incorrect f4 output")
	}

	//test smpF5
	macKey, ltk, err := smpF5(testW, testN1, testN2, testA1, testA2)
	if err != nil {
		t.Fatal("f5 calc failed:", err)
	}

	if !bytes.Equal(macKey, testExpMACKey) {
		t.Fatal("incorrect f5 macKey:", hex.EncodeToString(macKey))
	}

	if !bytes.Equal(ltk, testExpLTK) {
		t.Fatal("incorrect f5 ltk:", hex.EncodeToString(ltk))
	}

	res, err := smpF6(testWF6, testN1F6, testN2F6, testR, testIoCap, testA1F6, testA2)
	if err != nil {
		t.Fatal("incorrect f6 operation:", err)
	}

	if !bytes.Equal(res, expF6) {
		t.Fatal("incorrect f6 output:", hex.EncodeToString(res))
	}

	val, err := smpG2(testUG2, testVG2, testXG2, testYG2)
	if err != nil {
		t.Fatal("failed to calc G2:", err)
	}

	if val != expValG2 {
		t.Fatal("incorrect G2 output:", val)
	}
}
