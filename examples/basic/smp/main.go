package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rigado/ble"
	"github.com/rigado/ble/linux"
	bonds "github.com/rigado/ble/linux/hci/bond"
	"github.com/pkg/errors"
)

var (
	device       = flag.Int("device", 1, "hci index")
	name         = flag.String("name", "Nordic_HRM", "name of remote peripheral")
	addr         = flag.String("addr", "", "address of remote peripheral (MAC on Linux, UUID on OS X)")
	sub          = flag.Duration("sub", 0, "subscribe to notification and indication for a specified period")
	sd           = flag.Duration("sd", 20*time.Second, "scanning duration, 0 for indefinitely")
	pair         = flag.Bool("pair", true, "attempt to pair on connection")
	forceEncrypt = flag.Bool("fe", false, "force encryption to be started if pair information is found")
	passkey      = flag.Int("passkey", 0, "if passkey is required, use this passkey for pairing")
)

func main() {
	flag.Parse()
	log.Printf("device: hci%v", *device)
	log.Printf("name: %v", *name)
	log.Printf("addr: %v", *addr)

	optid := ble.OptDeviceID(*device)

	//To create a pair with a device, the pair manager needs a file
	//to store and load pair information
	bondFilePath := filepath.Join("bonds.json")
	bm := bonds.NewBondManager(bondFilePath)

	//Enable security by putting the pair manager in the enable security option
	optSecurity := ble.OptEnableSecurity(bm)
	d, err := linux.NewDeviceWithNameAndHandler("", nil, optid, optSecurity)
	if err != nil {
		log.Fatalf("can't new device : %s", err)
	}
	ble.SetDefaultDevice(d)


	// Default to search device with name of Gopher (or specified by user).
	filter := func(a ble.Advertisement) bool {
		return strings.Contains(strings.ToUpper(a.LocalName()), strings.ToUpper(*name))
	}

	// If addr is specified, search for addr instead.
	if len(*addr) != 0 {
		filter = func(a ble.Advertisement) bool {
			return strings.ToUpper(a.Addr().String()) == strings.ToUpper(*addr)
		}
	}

	// Scan for specified durantion, or until interrupted by user.
	fmt.Printf("Scanning for %s...\n", *sd)
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), *sd))
	cln, err := ble.Connect(ctx, filter)
	if err != nil {
		log.Fatalf("can't connect : %s", err)
	}

	// Make sure we had the chance to print out the message.
	done := make(chan struct{})
	// Normally, the connection is disconnected by us after our exploration.
	// However, it can be asynchronously disconnected by the remote peripheral.
	// So we wait(detect) the disconnection in the go routine.
	go func() {
		<-cln.Disconnected()
		fmt.Printf("[ %s ] is disconnected \n", cln.Addr())
		close(done)
	}()

	log.Println("connected!")
	//<-time.After(2 * time.Second)

	if *pair {
		//pairing can be manually triggered by issuing the pair command
		//however, the typical process is
		/* 1. connect
		   2. attempt to read or write to a characteristic which requires security
		   3. peripheral responds with insufficient authentication
		   4. central triggers bonding
		*/
		log.Println("pairing with", cln.Addr().String())
		ad := ble.AuthData{}
		if passkey != nil {
			ad.Passkey = *passkey
		}
		err = cln.Pair(ad, time.Minute)
		if err != nil {
			log.Println(err)
			_ = cln.CancelConnection()
			<-done
			os.Exit(1)
		} else {
			log.Println("pairing successful!")
		}
	}

	if *forceEncrypt {
		aStr := strings.Replace(cln.Addr().String(), ":", "", -1)
		aBytes, _ := hex.DecodeString(aStr)
		for i := len(aBytes)/2 - 1; i >= 0; i-- {
			opp := len(aBytes) - 1 - i
			aBytes[i], aBytes[opp] = aBytes[opp], aBytes[i]
		}

		log.Println("starting encryption for", hex.EncodeToString(aBytes))
		if exists := bm.Exists(hex.EncodeToString(aBytes)); exists == true {
			log.Println("found pair info; starting encryption")
			if err := cln.StartEncryption(); err != nil {
				log.Println("failed to start encryption:", err)
			}
		}
	}

	fmt.Printf("Discovering profile...\n")
	p, err := cln.DiscoverProfile(true)
	if err != nil {
		log.Fatalf("can't discover profile: %s", err)
	}

	explore(cln, p)
	<-time.After(5 * time.Second)

	// Disconnect the connection. (On OS X, this might take a while.)
	fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", cln.Addr())
	cln.CancelConnection()

	<-done
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
