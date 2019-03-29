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
