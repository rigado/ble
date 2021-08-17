package main

import (
	"context"
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
	"github.com/rigado/ble/linux/hci/h4"
)

var (
	name   = flag.String("name", "Thingy", "name of remote peripheral")
	addr   = flag.String("addr", "", "address of remote peripheral (MAC on Linux, UUID on OS X)")
	h4skt  = flag.String("h4s", "", "h4 socket server address")
	h4uart = flag.String("h4u", "", "h4 uart")
	hciSkt = flag.Int("device", -1, "hci index")
	sub    = flag.Duration("sub", 0, "subscribe to notification and indication for a specified period")
	sd     = flag.Duration("sd", 20*time.Second, "scanning duration, 0 for indefinitely")
	bond   = flag.Bool("bond", false, "attempt to bond on connection")
	dump   = flag.Bool("dump", false, "dump stack?")
)

func main() {
	flag.Parse()
	log.Printf("hciSkt: hci%v", *hciSkt)
	log.Printf("h4skt: %v", *h4skt)
	log.Printf("h4uart: %v", *h4uart)
	log.Printf("name: %v", *name)
	log.Printf("addr: %v", *addr)

	var opt ble.Option
	switch {
	case *hciSkt >= 0:
		opt = ble.OptTransportHCISocket(*hciSkt)
	case len(*h4skt) > 0:
		opt = ble.OptTransportH4Socket(*h4skt, 2*time.Second)
	case len(*h4uart) > 0:
		opt = ble.OptTransportH4Uart(*h4uart, int(h4.DefaultSerialOptions().BaudRate))
	default:
		log.Fatalf("no valid device to init")
	}

	//To create a pair with a device, the pair manager needs a file
	//to store and load pair information
	bondFilePath := filepath.Join("bonds.json")
	bm := bonds.NewBondManager(bondFilePath)

	//Enable security by putting the pair manager in the enable security option
	optSecurity := ble.OptEnableSecurity(bm)

	d, err := linux.NewDeviceWithNameAndHandler("", nil, opt, optSecurity)
	if err != nil {
		log.Fatalf("can't new device : %s", err)
	}
	ble.SetDefaultDevice(d)

	// Default to search device with name of Gopher (or specified by user).
	filter := func(a ble.Advertisement) bool {
		fmt.Println(a.Addr(), a.LocalName())
		return strings.EqualFold(a.LocalName(), *name)
	}

	// If addr is specified, search for addr instead.
	if len(*addr) != 0 {
		filter = func(a ble.Advertisement) bool {
			fmt.Println(a.Addr())
			return strings.EqualFold(a.Addr().String(), *addr)
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
	<-time.After(2000 * time.Millisecond)

	if *bond {
		//pairing can be manually triggered by issuing the pair command
		//however, the typical process is
		/* 1. connect
		   2. attempt to read or write to a characteristic which requires security
		   3. peripheral responds with insufficient authentication
		   4. central triggers bonding
		*/
		log.Println("pairing with", cln.Addr().String())

		err = cln.Pair(ble.AuthData{}, time.Second*30)
		if err != nil {
			log.Println(err)
			_ = cln.CancelConnection()
			<-done
			os.Exit(1)
		} else {
			log.Println("pairing successful!")
		}
	}

	rxMtu := ble.MaxMTU
	txMtu, err := cln.ExchangeMTU(rxMtu)
	if err != nil {
		fmt.Printf("%v - MTU exchange error: %v\n", *addr, err)
		// stay connected
	} else {
		fmt.Printf("%v - MTU exchange success: rx %v, tx %v\n", *addr, rxMtu, txMtu)
	}

	fmt.Printf("Discovering profile...\n")
	p, err := cln.DiscoverProfile(true)
	if err != nil {
		log.Fatalf("can't discover profile: %s", err)
	}

	log.Println("exploring")
	// Start the exploration.
	explore(cln, p)

	// Disconnect the connection. (On OS X, this might take a while.)
	fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", cln.Addr())
	cln.CancelConnection()

	<-done

	time.Sleep(125 * time.Millisecond)
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
					h := func(id uint, req []byte) { fmt.Printf("Notified: id %v, %q [ % X ]\n", id, string(req), req) }
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
					h := func(id uint, req []byte) { fmt.Printf("Indicated: id %v, %q [ % X ]\n", id, string(req), req) }
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
