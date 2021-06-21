package smp

import (
	"bytes"
	"strings"
	"testing"
)

func TestOnSmpPairingPublicKey(t *testing.T) {
	p := pairingContext{}
	tran := &transport{}

	tran.pairing = &p

	remote, err := GenerateKeys()
	if err != nil {
		t.Fatalf("failed to generate remote keys: %v\n", err)
	}
	rBytes := MarshalPublicKeyXY(remote.public)

	tran.pairing.scECDHKeys, err = GenerateKeys()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v\n", err)
	}

	_, err = smpOnPairingPublicKey(tran, rBytes)
	if err != nil {
		t.Fatalf("failed to process remote public key: %v\n", err)
	}

	testBytes := MarshalPublicKeyXY(tran.pairing.scRemotePubKey)
	if !bytes.Equal(rBytes, testBytes) {
		t.Fatalf("failed to correctly unmarshal remote public key")
	}
}

func TestOnSmpPairingPublicKeyMatchingRemoteKey(t *testing.T) {
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
