package hci

import (
	"testing"

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

	//good
	good := evt.LEAdvertisingReport{2, 1, 3, 1, 144, 17, 101, 210, 60, 246, 30, 2, 1, 2, 26, 255, 76, 0, 2, 21, 255, 254, 45, 18, 30, 75, 15, 164, 153, 78, 4, 99, 49, 239, 205, 171, 52, 18, 120, 86, 195, 205}
	a, err = newAdvertisement(good, 0)
	t.Log(a, err)
	if err != nil {
		t.Fatal(err)
	}
}
