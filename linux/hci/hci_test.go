package hci

import (
	"reflect"
	"testing"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux/hci/evt"
)

var r interface{}

func BenchmarkAdv2Map(b *testing.B) {
	var rr interface{}
	//ibeacon

	for i := 0; i < b.N; i++ {
		e := evt.LEAdvertisingReport{2, 1, 3, 1, 144, 17, 101, 210, 60, 246, 30, 2, 1, 2, 26, 255, 76, 0, 2, 21, 255, 254, 45, 18, 30, 75, 15, 164, 153, 78, 4, 99, 49, 239, 205, 171, 52, 18, 120, 86, 195, 205}
		a, _ := newAdvertisement(e, 0)
		rr, _ = a.ToMap()
	}
	r = rr
}

func TestAdvDecode(t *testing.T) {
	/*
		2, (subevt code)
		1, (report count)
		0, (evt type: conn scannable)
		0, (addr type: public)
		45, 58, 130, 157, 134, 122, (mac)
		29, (data len)
			2, 1, 6, (flag field 2 bytes, flag=6)
			2, 5, 9, (uuid32 field 2 bytes, 9...?????)
			(broken...)
			67, 97, 115, 99,
			97, 100, 101, 45,
			67, 48, 51, 49,
			48, 54, 49, 56,
			51, 52, 45, 48,
			48, 49, 57,
			49, (this is byte 30... bad???)
		197 (rssi)
	*/
	bad := evt.LEAdvertisingReport{2, 1, 0, 0, 45, 58, 130, 157, 134, 122, 29, 2, 1, 6, 2, 5, 9, 67, 97, 115, 99, 97, 100, 101, 45, 67, 48, 51, 49, 48, 54, 49, 56, 51, 52, 45, 48, 48, 49, 57, 49, 197}
	a, err := newAdvertisement(bad, 0)
	t.Log(a, err)
	if err == nil {
		t.Fatal("no error on malformed payload")
	}

	//good ibeacon
	good := evt.LEAdvertisingReport{2, 1, 3, 1, 144, 17, 101, 210, 60, 246, 30, 2, 1, 2, 26, 255, 76, 0, 2, 21, 255, 254, 45, 18, 30, 75, 15, 164, 153, 78, 4, 99, 49, 239, 205, 171, 52, 18, 120, 86, 195, 205}
	a, err = newAdvertisement(good, 0)
	t.Log(a, err)
	if err != nil {
		t.Fatal(err)
	}

	//good uuid16, uuid128
	good = evt.LEAdvertisingReport{2, 1, 3, 1, 1, 2, 3, 4, 5, 6, 24,
		5, 2,
		1, 2, 11, 22,
		17, 6,
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 1, 2, 3,
		4, 5, 6, 7,
		16}
	a, err = newAdvertisement(good, 0)
	t.Log(a, err)
	if err != nil {
		t.Fatal(err)
	}
	m, err := a.ToMap()
	t.Log(m, err)

	v, ok := m["mac"].(string)
	if !ok {
		t.Fatal("missing mac")
	}
	if !reflect.DeepEqual(v, ble.UUID(good[4:10]).String()) {
		t.Fatal("mac mismatch")
	}

	s, ok := m["services"].([]ble.UUID)
	if !ok {
		t.Fatal("no services present")
	}
	if len(s) != 3 {
		t.Fatal("incorrect service count")
	}

	if !reflect.DeepEqual(s[0], ble.UUID(good[13:15])) {
		t.Fatal("service uuid mismatch @ 0")
	}

	if !reflect.DeepEqual(s[1], ble.UUID(good[15:17])) {
		t.Fatal("service uuid mismatch @ 1")
	}

	if !reflect.DeepEqual(s[2], ble.UUID(good[19:35])) {
		t.Fatal("service uuid mismatch @ 2\n", ble.UUID(good[19:35]), s[2])
	}

	//good mfg data (ruuvi mode 3)
	good = evt.LEAdvertisingReport{2, 1, 3, 1, 1, 2, 3, 4, 5, 6, 21, 0x02, 0x01, 0x06, 0x11, 0xFF, 0x99, 0x04, 0x03, 0x4B, 0x16, 0x19, 0xC7, 0x3B, 0xFF, 0xFF, 0x00, 0x0C, 0x03, 0xE0, 0x0B, 0x89, 255}
	a, err = newAdvertisement(good, 0)
	t.Log(a, err)
	if err != nil {
		t.Fatal(err)
	}
	m, err = a.ToMap()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(m, err)

	v, ok = m["mac"].(string)
	if !ok {
		t.Fatal("missing mac")
	}

	ok = reflect.DeepEqual(v, "060504030201")
	if !ok {
		t.Fatal("mac mismatch")
	}

	vv, ok := m["eventType"].(byte)
	if !ok {
		t.Fatal("missing eventType")
	}

	ok = reflect.DeepEqual(vv, byte(3))
	if !ok {
		t.Fatal("eventType mismatch")
	}

	md, ok := m["mfg"].([]byte)
	if !ok {
		t.Fatal("missing mfg data")
	}

	if !reflect.DeepEqual(md, []byte(good[16:32])) {
		t.Fatal("mfgData mismatch")
	}
}
