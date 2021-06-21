package smp

import (
	"strings"
	"testing"
)

func TestOnSmpPairingPublicKey(t *testing.T) {
	p := pairingContext{}
	tran := &transport{}

	tran.pairing = &p

	var err error
	tran.pairing.scECDHKeys, err = GenerateKeys()
	if err != nil {
		t.Fatal("failed to generate key pair")
	}

	pubBytes := MarshalPublicKeyXY(tran.pairing.scECDHKeys.public)

	_, err = smpOnPairingPublicKey(tran, pubBytes)
	if !strings.Contains(err.Error(), "remote public key cannot") {
		t.Fatal("failed to detected remote public key matching local public key")
	}
}
