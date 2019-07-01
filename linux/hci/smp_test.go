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
