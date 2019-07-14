package hci

import (
	"bytes"
	"crypto/elliptic"
	"encoding/hex"
	"testing"

	ecdh "github.com/wsddn/go-ecdh"
)

func Test_SMP_Key(t *testing.T) {
	k1, err := GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	k2, err := GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	e := ecdh.NewEllipticECDH(elliptic.P256())
	s1, _ := e.GenerateSharedSecret(k1.private, k2.public)
	s2, _ := e.GenerateSharedSecret(k2.private, k1.public)

	if !bytes.Equal(s1, s2) {
		t.Fatal()
	}

	// this is a dumped key from a real exchange
	hs := "c697669493e497655afb7be56e319d53d97a7d5e4b043cfb23c1978ea9433ea62a56c8fda27d8ed835b5af7a31574ad71aa06ee745bc85e36bfde05b66a28d7d"
	hb, err := hex.DecodeString(hs)
	if _, ok := UnmarshalPublicKey(hb); !ok {
		t.Fatal("unmarshal err")
	}
}

func Test_SMP_Legacy_Confirm(t *testing.T) {
	//request
	preq := []byte{pairingRequest, 0x03, 0x00, 0x09, 16, 0x05, 0x07}
	pres := []byte{pairingResponse, 0x03, 0x00, 0x01, 16, 0x01, 0x03}

	la := []byte{0x98, 0x5a, 0x2f, 0x93, 0x54, 0x94}
	lat := uint8(0x00)
	lrand, _ := hex.DecodeString("45e39d7a7bb5f81e979b516757ecb2dc")

	ra := []byte{0x98, 0xd3, 0x45, 0x85, 0x47, 0xd8}
	rat := uint8(0x01)
	rrand, _ := hex.DecodeString("e6d5505348fa4188acfb209860fd9524")

	expMConfirm, _ := hex.DecodeString("ff5985f3216bb8f0d9812e700a5a6477")
	expRConfirm, _ := hex.DecodeString("10e6a8b112adf45c47468c6ac0f31294")

	confirm, _ := smpC1(make([]byte,16), lrand, preq, pres, lat, rat, la, ra)
	if !bytes.Equal(expMConfirm, confirm) {
		t.Fatalf("failed to generate correct mConfirm value:\n%s\n%s",
			hex.EncodeToString(expMConfirm), hex.EncodeToString(confirm))
	}

	confirm, _ = smpC1(make([]byte, 16), rrand, preq, pres, lat, rat, la, ra)
	if !bytes.Equal(expRConfirm, confirm) {
		t.Fatalf("failed to generate correct rConfirm value:\n%s\n%s",
			hex.EncodeToString(expRConfirm), hex.EncodeToString(confirm))
	}
}

func Test_SMP_Legacy_Confirm2(t *testing.T) {
	//request
	preq := []byte{pairingRequest, 0x03, 0x00, 0x09, 16, 0x01, 0x01}
	pres := []byte{pairingResponse, 0x03, 0x00, 0x01, 16, 0x01, 0x01}

	la := []byte{0x98, 0x5a, 0x2f, 0x93, 0x54, 0x94}
	lat := uint8(0x00)
	lrand, _ := hex.DecodeString("5eb83e928a5ad801d99bd6b9cf339167")

	ra := []byte{0x98, 0xd3, 0x45, 0x85, 0x47, 0xd8}
	rat := uint8(0x01)
	rrand, _ := hex.DecodeString("8c101aba0f623e3450dfb817a1e0b425")

	expMConfirm, _ := hex.DecodeString("e2e6907164813041a28b1a399babe1d0")
	expRConfirm, _ := hex.DecodeString("0acb5b32c7f60851eaa96649b7effcce")

	confirm, _ := smpC1(make([]byte,16), lrand, preq, pres, lat, rat, la, ra)
	if !bytes.Equal(expMConfirm, confirm) {
		t.Fatalf("failed to generate correct mConfirm value:\n%s\n%s",
			hex.EncodeToString(expMConfirm), hex.EncodeToString(confirm))
	}

	confirm, _ = smpC1(make([]byte, 16), rrand, preq, pres, lat, rat, la, ra)
	if !bytes.Equal(expRConfirm, confirm) {
		t.Fatalf("failed to generate correct rConfirm value:\n%s\n%s",
			hex.EncodeToString(expRConfirm), hex.EncodeToString(confirm))
	}
}

