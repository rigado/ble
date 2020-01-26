package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rigado/ble"
	"github.com/rigado/ble/linux"
	"github.com/pkg/errors"
)

var (
	device = flag.Int("device", 1, "hci index")
	name   = flag.String("name", "", "name of remote peripheral")
	addr   = flag.String("addr", "", "address of remote peripheral (MAC on Linux, UUID on OS X)")
	sub    = flag.Duration("sub", 0, "subscribe to notification and indication for a specified period")
	sd     = flag.Duration("sd", 20*time.Second, "scanning duration, 0 for indefinitely")
	bond = flag.Bool("bond", false, "attempt to bond on connection")
	forceEncrypt = flag.Bool("fe", false, "force encryption to be started if bond information is found")
	test = flag.String("test", "", "the test to be run")
)

func main() {
	flag.Parse()

	log.Printf("device: hci%v", *device)

	if len(*name) > 0 {
		log.Printf("name: %v", *name)
	}

	if len(*addr) > 0 {
		log.Printf("addr: %v", *addr)
	}

	if len(*test) != 0 {
		runTest(*test)
		return
	}

	fmt.Println("no test specified! use --test")
	fmt.Println("available tests are `scan`, `connect`, and `notif`")
}

func explore(cln ble.Client, p *ble.Profile) error {
	for _, s := range p.Services {
		fmt.Printf("    Service: %s %s, Handle (0x%02X)\n", s.UUID, ble.Name(s.UUID), s.Handle)

		for _, c := range s.Characteristics {
			fmt.Printf("      Characteristic: %s %s, Property: 0x%02X (%s), Handle(0x%02X), VHandle(0x%02X)\n",
				c.UUID, ble.Name(c.UUID), c.Property, propString(c.Property), c.Handle, c.ValueHandle)
			if (c.Property & ble.CharRead) != 0 {
				b, err := cln.ReadCharacteristic(c)
				if err != nil {
					fmt.Printf("Failed to read characteristic: %s\n", err)
					continue
				}
				fmt.Printf("        Value         %x | %q\n", b, b)
			}

			for _, d := range c.Descriptors {
				fmt.Printf("        Descriptor: %s %s, Handle(0x%02x)\n", d.UUID, ble.Name(d.UUID), d.Handle)
				b, err := cln.ReadDescriptor(d)
				if err != nil {
					fmt.Printf("Failed to read descriptor: %s\n", err)
					continue
				}
				fmt.Printf("        Value         %x | %q\n", b, b)
			}

			if *sub != 0 {
				// Don't bother to subscribe the Service Changed characteristics.
				if c.UUID.Equal(ble.ServiceChangedUUID) {
					continue
				}

				// Don't touch the Apple-specific Service/Characteristic.
				// Service: D0611E78BBB44591A5F8487910AE4366
				// Characteristic: 8667556C9A374C9184ED54EE27D90049, Property: 0x18 (WN),
				//   Descriptor: 2902, Client Characteristic Configuration
				//   Value         0000 | "\x00\x00"
				if c.UUID.Equal(ble.MustParse("8667556C9A374C9184ED54EE27D90049")) {
					continue
				}

				if (c.Property & ble.CharNotify) != 0 {
					fmt.Printf("\n-- Subscribe to notification for %s --\n", *sub)
					h := func(req []byte) { fmt.Printf("Notified: %q [ % X ]\n", string(req), req) }
					if err := cln.Subscribe(c, false, h); err != nil {
						log.Fatalf("subscribe failed: %s", err)
					}
					time.Sleep(*sub)
					if err := cln.Unsubscribe(c, false); err != nil {
						log.Fatalf("unsubscribe failed: %s", err)
					}
					fmt.Printf("-- Unsubscribe to notification --\n")
				}
				if (c.Property & ble.CharIndicate) != 0 {
					fmt.Printf("\n-- Subscribe to indication of %s --\n", *sub)
					h := func(req []byte) { fmt.Printf("Indicated: %q [ % X ]\n", string(req), req) }
					if err := cln.Subscribe(c, true, h); err != nil {
						log.Fatalf("subscribe failed: %s", err)
					}
					time.Sleep(*sub)
					if err := cln.Unsubscribe(c, true); err != nil {
						log.Fatalf("unsubscribe failed: %s", err)
					}
					fmt.Printf("-- Unsubscribe to indication --\n")
				}
			}
		}
		fmt.Printf("\n")
	}
	return nil
}

func propString(p ble.Property) string {
	var s string
	for k, v := range map[ble.Property]string{
		ble.CharBroadcast:   "B",
		ble.CharRead:        "R",
		ble.CharWriteNR:     "w",
		ble.CharWrite:       "W",
		ble.CharNotify:      "N",
		ble.CharIndicate:    "I",
		ble.CharSignedWrite: "S",
		ble.CharExtended:    "E",
	} {
		if p&k != 0 {
			s += v
		}
	}
	return s
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

func runTest(test string) {
	switch test {
	case "scan":
		runScanTest()
	case "connect":
		runConnectTest()
	}
}

func runScanTest() {
	optid := ble.OptDeviceID(*device)
	d, err := linux.NewDeviceWithNameAndHandler("", nil, optid)//, optSecurity)
	if err != nil {
		log.Fatalf("can't new device : %s", err)
	}
	ble.SetDefaultDevice(d)

	fmt.Printf("Scanning for %s...\n", *sd)
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), *sd))

	adv := func(a ble.Advertisement) {}

	go func() {
		_ = ble.Scan(ctx, true, adv, nil)
	}()

	time.Sleep(5 * time.Second)

	fmt.Println("closing hci after 5 seconds")
	err = d.HCI.Close()
	if err != nil {
		fmt.Println(err)
	}
}

func runConnectTest() {
	optid := ble.OptDeviceID(*device)
	d, err := linux.NewDeviceWithNameAndHandler("", nil, optid)//, optSecurity)
	if err != nil {
		log.Fatalf("can't new device : %s", err)
	}
	ble.SetDefaultDevice(d)

	fmt.Printf("Scanning for %s...\n", *sd)
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), *sd))

	filter := func(a ble.Advertisement) bool {
		return strings.ToUpper(a.Addr().String()) == strings.ToUpper(*addr)
	}

	_, err = ble.Connect(ctx, filter)
	if err != nil {
		log.Fatalf("can't connect : %s", err)
	}

	time.Sleep(5 * time.Second)

	fmt.Println("closing hci after 5 seconds")
	err = d.HCI.Close()
	if err != nil {
		fmt.Println(err)
	}
}

