package smp

import (
	"crypto/aes"
	"encoding/binary"

	"github.com/aead/cmac"
	"github.com/rigado/ble/sliceops"
)

func aesCMAC(key, msg []byte) ([]byte, error) {
	tmp := sliceops.SwapBuf(key)
	mCipher, err := aes.NewCipher(tmp)
	if err != nil {
		return nil, err
	}

	msgMsb := sliceops.SwapBuf(msg)

	mMac, err := cmac.New(mCipher)
	if err != nil {
		return nil, err
	}

	mMac.Write(msgMsb)

	return sliceops.SwapBuf(mMac.Sum(nil)), nil
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

func isLegacy(authReq byte) bool {
	if authReq&0x08 == 0x08 {
		return false
	}

	return true
}

func getLegacyParingTK(key int) []byte {
	tk := make([]byte, 16)
	keyBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(keyBytes, uint32(key))
	tk[12] = keyBytes[0]
	tk[13] = keyBytes[1]
	tk[14] = keyBytes[2]
	tk[15] = keyBytes[3]

	tk = sliceops.SwapBuf(tk)

	return tk
}
