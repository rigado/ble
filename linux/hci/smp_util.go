package hci

import (
	"crypto/aes"
	"github.com/enceve/crypto/cmac"
)

func aesCMAC(key, msg []byte) ([]byte, error) {
	tmp := swapBuf(key)
	mCipher, err := aes.NewCipher(tmp)
	if err != nil {
		return nil, err
	}

	msgMsb := swapBuf(msg)

	mMac, err := cmac.New(mCipher)
	if err != nil {
		return nil, err
	}

	mMac.Write(msgMsb)

	return swapBuf(mMac.Sum(nil)), nil
}

func xorSlice(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

func aes128(key, msg []byte) []byte {
	mCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}

	out := make([]byte, 16)
	mCipher.Encrypt(out, msg)
	return out
}

func swapBuf(in []byte) []byte {
	a := make([]byte, 0, len(in))
	a = append(a, in...)
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}
