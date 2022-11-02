package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/rigado/ble/linux"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/rigado/ble"
)

var (
	device = flag.String("device", "default", "implementation of ble")
	du     = flag.Duration("du", 5*time.Second, "advertising duration, 0 for indefinitely")
	name   = flag.String("name", "Cascade", "name of the peripheral device")
)

func main() {
	flag.Parse()

	opt := ble.OptTransportHCISocket(0)

	d, err := linux.NewDeviceWithNameAndHandler("", nil, opt)
	if err != nil {
		log.Fatalf("can't new device : %s", err)
	}
	ble.SetDefaultDevice(d)

	// Advertise for specified durantion, or until interrupted by user.
	fmt.Printf("Advertising for %s...\n", *du)
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), *du))
	chkErr(ble.AdvertiseNameAndServices(ctx, *name, ble.BatteryUUID, ble.DeviceInfoUUID))
}

func chkErr(err error) {
	switch errors.Cause(err) {
	case nil:
	case context.DeadlineExceeded:
		fmt.Printf("done\n")
	case context.Canceled:
		fmt.Printf("canceled\n")
	default:
		log.Fatalf(err.Error())
	}
}
