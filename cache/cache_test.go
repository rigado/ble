package cache

import (
	"github.com/rigado/ble"
	"os"
	"reflect"
	"testing"
)

func TestGattCache_Store(t *testing.T) {
	defer os.Remove("./test.cache")
	p := ble.Profile{}

	svc := ble.NewService(ble.MustParse("180d"))
	svc.NewCharacteristic(ble.MustParse("2f37"))
	p.Services = append(p.Services, svc)

	c := New("./test.cache")
	err := c.Store(ble.NewAddr("12:34:56:78:90:ab:cd"), p, false)
	if err != nil {
		t.Fatalf("expected nil error but got %s instead", err)
	}

	loaded, err := c.Load(ble.NewAddr("12:34:56:78:90:ab:cd"))
	if err != nil {
		t.Fatalf("expected to find mac in cache but did not: %s", err)
	}

	if !reflect.DeepEqual(p, loaded) {
		t.Fatalf("stored and loaded caches are not equal")
	}
}
