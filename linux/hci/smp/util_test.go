package smp

import (
	"bytes"
	"testing"
)

func TestAesCMAC(t *testing.T) {
	key := []byte("Stt8Zh+srft8Uv0q26R2FNo/QtQJ+RJL")
	msg := []byte("message")
	response := []byte{206, 52, 198, 186, 125, 62, 93, 46, 130, 150, 87, 239, 31, 97, 228, 37}

	r, err := aesCMAC(key, msg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(r, response) {
		t.Fatal("Response didn't match")
	}

}
