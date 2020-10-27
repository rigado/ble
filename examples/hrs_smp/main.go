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
)

var (
	device       = flag.Int("device", 1, "hci index")
	name         = flag.String("name", "Nordic_HRM", "name of remote peripheral")
	addr         = flag.String("addr", "", "address of remote peripheral (MAC on Linux, UUID on OS X)")
	sub          = flag.Duration("sub", 0, "subscribe to notification and indication for a specified period")
	sd           = flag.Duration("sd", 20*time.Second, "scanning duration, 0 for indefinitely")
	pair         = flag.Bool("pair", false, "attempt to pair on connection")
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
		return strings.ToUpper(a.LocalName()) == strings.ToUpper(*name)
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
	start := time.Now()

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

	ad := ble.AuthData{}
	if passkey != nil {
		ad.Passkey = *passkey
	}

	if *pair {
		//pairing can be manually triggered by issuing the pair command
		//however, the typical process is
		/* 1. connect
		   2. attempt to read or write to a characteristic which requires security
		   3. peripheral responds with insufficient authentication
		   4. central triggers bonding
		*/
		log.Println("pairing with", cln.Addr().String())

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
			m := make(chan ble.EncryptionChangedInfo)
			if err := cln.StartEncryption(m); err != nil {
				log.Println("failed to start encryption:", err)
			}
		}
	}

	//perform server and characteristic discovery
	hrCharUUID, _ := ble.Parse("2a37")
	hrServiceUUID, _ := ble.Parse("180d")
	services, err := cln.DiscoverServices([]ble.UUID{hrServiceUUID})
	if err != nil {
		log.Println("failed to discover heart rate service:", err)
		_ = cln.CancelConnection()
		<-done
		os.Exit(1)
	}

	if len(services) == 0 {
		log.Println("failed to discover heart rate service")
		_ = cln.CancelConnection()
		<-done
		os.Exit(1)
	}

	var svc *ble.Service
	for _, s := range services {
		if hrServiceUUID.Equal(s.UUID) {
			svc = s
			break
		}
	}

	if svc == nil {
		log.Println("failed to discover heart rate service")
		_ = cln.CancelConnection()
		<-done
		os.Exit(1)
	}

	chars, err := cln.DiscoverCharacteristics([]ble.UUID{hrCharUUID}, svc)
	if err != nil {
		log.Println("failed to discover heart rate characteristic:", err)
		_ = cln.CancelConnection()
		<-done
		os.Exit(1)
	}

	if len(chars) == 0 {
		log.Println("failed to discover heart rate characteristic")
		_ = cln.CancelConnection()
		<-done
		os.Exit(1)
	}

	var hrc *ble.Characteristic
	for _, c := range chars {
		if hrCharUUID.Equal(c.UUID) {
			hrc = c
			break
		}
	}

	if hrc == nil {
		log.Println("failed to discover heart rate characteristic")
		_ = cln.CancelConnection()
		<-done
		os.Exit(1)
	}

	_, _ = cln.DiscoverDescriptors(nil, hrc)

	//try to enable notifications
	//attempt pairing on authentication failure
	first := true
	for tries := 0; tries < 3; tries++ {
		fmt.Println("attempt to subscribe")
		err = cln.Subscribe(hrc, false, func(id uint, req []byte) {
			if first {
				first = false
				end := time.Now()
				log.Printf("time to first notification: %d ms\n", int(end.Sub(start).Milliseconds()))
			}
			log.Printf("hr notification: %s\n", hex.EncodeToString(req))
		})

		if err != nil {
			if strings.Contains(err.Error(), "insufficient authentication") {
				err = cln.Pair(ad, time.Minute)
				if err != nil {
					log.Printf("failed to pair: %s\n", err)
					_ = cln.CancelConnection()
					<-done
					os.Exit(1)
				} else {
					log.Println("successfully paired")
				}
			} else {
				log.Printf("failed to enable notifications for heart rate: %s\n", err)
				_ = cln.CancelConnection()
				<-done
				os.Exit(1)
			}
		}
	}

	<-time.After(1 * time.Second)

	// Since this is performing a bond, we need to disable notifications at the end
	// Otherwise, upon reconnection, the notifications will be sent automatically by the device
	err = cln.Unsubscribe(hrc, false)
	if err != nil {
		log.Println("failed to unsubscribe to heart rate notifications:", err)
	}
	// Disconnect the connection. (On OS X, this might take a while.)
	fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", cln.Addr())
	err = cln.CancelConnection()
	if err != nil {
		log.Println("failed to cancel peripheral connection:", err)
	}

	<-done
}
