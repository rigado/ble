package hci

import (
	"crypto/aes"
	"encoding/binary"
	"fmt"

	"github.com/enceve/crypto/cmac"
)

func smpF4(u, v, x []byte, z uint8) ([]byte, error) {
	if len(u) != 32 || len(v) != 32 || len(x) != 16 {
		return nil, fmt.Errorf("length error")
	}

	m := []byte{z}
	m = append(m, v...)
	m = append(m, u...)

	return aesCMAC(x, m)
}

func smpF5(w, n1, n2, a1, a2 []byte) ([]byte, []byte, error) {
	switch {
	case len(w) != 32:
		return nil, nil, fmt.Errorf("length error w")
	case len(n1) != 16:
		return nil, nil, fmt.Errorf("length error n1")
	case len(n2) != 16:
		return nil, nil, fmt.Errorf("length error n2")
	case len(a1) != 7:
		return nil, nil, fmt.Errorf("length error a1")
	case len(a2) != 7:
		return nil, nil, fmt.Errorf("length error a2")
	}

	btle := []byte{0x65, 0x6c, 0x74, 0x62}
	salt := []byte{0xbe, 0x83, 0x60, 0x5a, 0xdb, 0x0b, 0x37, 0x60,
		0x38, 0xa5, 0xf5, 0xaa, 0x91, 0x83, 0x88, 0x6c}
	length := []byte{0x00, 0x01}

	t, err := aesCMAC(salt, w)
	if err != nil {
		fmt.Println("failed to generate f5 key:", err)
		return nil, nil, err
	}

	m := length
	m = append(m, a2...)
	m = append(m, a1...)
	m = append(m, n2...)
	m = append(m, n1...)
	m = append(m, btle...)
	m = append(m, 0x00)

	macKey, err := aesCMAC(t, m)
	if err != nil {
		fmt.Println("failed to generate macKey:", err)
		return nil, nil, err
	}

	//ltk generation bit
	m[52] = 0x01

	ltk, err := aesCMAC(t, m)
	if err != nil {
		fmt.Print("failed to generate ltk:", err)
		return nil, nil, err
	}

	return macKey, ltk, nil
}

func smpF6(w, n1, n2, r, ioCap, a1, a2 []byte) ([]byte, error) {
	if len(w) != 16 || len(n1) != 16 || len(n2) != 16 || len(r) != 16 || len(ioCap) != 3 || len(a1) != 7 || len(a2) != 7 {
		return nil, fmt.Errorf("length error")
	}

	// f6(W, N1, N2, R, IOcap, A1, A2) = AES-CMAC W (N1 || N2 || R || IOcap || A1 || A2)
	m := append(a2, a1...)
	m = append(m, ioCap...)
	m = append(m, r...)
	m = append(m, n2...)
	m = append(m, n1...)

	return aesCMAC(w, m)
}

func smpG2(u, v, x, y []byte) (uint32, error) {
	if len(u) != 32 || len(v) != 32 || len(x) != 16 || len(y) != 16 {
		return 0, fmt.Errorf("length error")
	}

	// g2 (U, V, X, Y) = AES-CMAC X (U || V || Y) mod 2^32
	m := append(y, v...)
	m = append(m, u...)

	h, err := aesCMAC(x, m)
	if err != nil {
		return 0, err
	}

	out := binary.LittleEndian.Uint32(h[:4])
	return uint32(out % 1000000), nil
}

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

func swapBuf(in []byte) []byte {
	a := make([]byte, 0, len(in))
	a = append(a, in...)
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}
