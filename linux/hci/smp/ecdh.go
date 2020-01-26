package smp

import (
	"crypto"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/wsddn/go-ecdh"
)

type ECDHKeys struct {
	public  crypto.PublicKey
	private crypto.PrivateKey
}

func GenerateKeys() (*ECDHKeys, error) {
	var err error
	kp := ECDHKeys{}
	e := ecdh.NewEllipticECDH(elliptic.P256())

	kp.private, kp.public, err = e.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &kp, nil
}

func UnmarshalPublicKey(b []byte) (crypto.PublicKey, bool) {
	e := ecdh.NewEllipticECDH(elliptic.P256())
	xs := swapBuf(b[:32])
	ys := swapBuf(b[32:])

	//add header
	r := append([]byte{0x04}, xs...)
	r = append(r, ys...)

	pk, ok := e.Unmarshal(r)

	return pk, ok
}

func MarshalPublicKeyXY(k crypto.PublicKey) []byte {
	e := ecdh.NewEllipticECDH(elliptic.P256())

	ba := e.Marshal(k)
	ba = ba[1:] //remove header
	x := swapBuf(ba[:32])
	y := swapBuf(ba[32:])

	out := append(x, y...)

	return out
}

func MarshalPublicKeyX(k crypto.PublicKey) []byte {
	e := ecdh.NewEllipticECDH(elliptic.P256())

	ba := e.Marshal(k)
	ba = ba[1:] //remove header
	x := swapBuf(ba[:32])

	return x
}

func GenerateSecret(prv crypto.PrivateKey, pub crypto.PublicKey) ([]byte, error) {
	e := ecdh.NewEllipticECDH(elliptic.P256())
	b, err := e.GenerateSharedSecret(prv, pub)
	b = swapBuf(b)
	return b, err
}
