package adv

import (
	"fmt"

	"github.com/go-ble/ble"
	"github.com/pkg/errors"
)

// https://www.bluetooth.org/en-us/specification/assigned-numbers/generic-access-profile
var types = struct {
	flags       byte
	uuid16inc   byte
	uuid16comp  byte
	uuid32inc   byte
	uuid32comp  byte
	uuid128inc  byte
	uuid128comp byte
	sol16       byte
	sol32       byte
	sol128      byte
	svc16       byte
	svc32       byte
	svc128      byte
	nameshort   byte
	namecomp    byte
	txpwr       byte
	mfgdata     byte
}{
	flags:       0x01,
	uuid16inc:   0x02,
	uuid16comp:  0x03,
	uuid32inc:   0x04,
	uuid32comp:  0x05,
	uuid128inc:  0x06,
	uuid128comp: 0x07,
	sol16:       0x14,
	sol32:       0x1f,
	sol128:      0x15,
	svc16:       0x16,
	svc32:       0x20,
	svc128:      0x21,
	nameshort:   0x08,
	namecomp:    0x09,
	txpwr:       0x0a,
	mfgdata:     0xff,
}

var keys = struct {
	flags       string
	services    string
	solicited   string
	serviceData string
	localName   string
	txpwr       string
	mfgdata     string
}{
	flags:       ble.AdvertisementMapKeys.Flags,
	services:    ble.AdvertisementMapKeys.Services,
	solicited:   ble.AdvertisementMapKeys.Solicited,
	serviceData: ble.AdvertisementMapKeys.ServiceData,
	localName:   ble.AdvertisementMapKeys.Name,
	txpwr:       ble.AdvertisementMapKeys.TxPower,
	mfgdata:     ble.AdvertisementMapKeys.MFG,
}

type pduRecord struct {
	arrayElementSz int
	minSz          int
	svcDataUUIDSz  int
	key            string
}

var pduDecodeMap = map[byte]pduRecord{
	types.uuid16inc: pduRecord{
		2,
		2,
		0,
		keys.services,
	},
	types.uuid16comp: pduRecord{
		2,
		2,
		0,
		keys.services,
	},
	types.uuid32inc: pduRecord{
		4,
		4,
		0,
		keys.services,
	},
	types.uuid32comp: pduRecord{
		4,
		4,
		0,
		keys.services,
	},
	types.uuid128inc: pduRecord{
		16,
		16,
		0,
		keys.services,
	},
	types.uuid128comp: pduRecord{
		16,
		16,
		0,
		keys.services,
	},
	types.sol16: pduRecord{
		2,
		2,
		0,
		keys.solicited,
	},
	types.sol32: pduRecord{
		4,
		4,
		0,
		keys.solicited,
	},
	types.sol128: pduRecord{
		16,
		16,
		0,
		keys.solicited,
	},
	types.svc16: pduRecord{
		0,
		2,
		2,
		keys.serviceData,
	},
	types.svc32: pduRecord{
		0,
		4,
		4,
		keys.serviceData,
	},
	types.svc128: pduRecord{
		0,
		16,
		16,
		keys.serviceData,
	},
	types.namecomp: pduRecord{
		0,
		1,
		0,
		keys.localName,
	},
	types.nameshort: pduRecord{
		0,
		1,
		0,
		keys.localName,
	},
	types.txpwr: pduRecord{
		0,
		1,
		0,
		keys.txpwr,
	},
	types.mfgdata: pduRecord{
		0,
		1,
		0,
		keys.mfgdata,
	},
	types.flags: pduRecord{
		0,
		1,
		0,
		keys.flags,
	},
}

func getArray(size int, bytes []byte) ([]ble.UUID, error) {
	//valid size?
	if size <= 0 {
		return nil, fmt.Errorf("invalid size")
	}

	//bytes empty/nil?
	if bytes == nil || len(bytes) == 0 {
		return nil, fmt.Errorf("nil/empty bytes")
	}

	//any remainder?
	count := len(bytes) / size
	rem := len(bytes) % size
	if rem != 0 || count == 0 {
		return nil, fmt.Errorf("incorrect size")
	}

	//prealloc
	arr := make([]ble.UUID, 0, count)

	for j := 0; j < len(bytes); j += size {
		o := bytes[j:(j + size)]
		arr = append(arr, o)
	}

	return arr, nil
}

func decode(pdu []byte) (map[string]interface{}, error) {
	if pdu == nil {
		return nil, fmt.Errorf("invalid pdu: %v", pdu)
	}

	m := make(map[string]interface{})
	for i := 0; (i + 1) < len(pdu); {

		//length @ offset 0
		//type @ offset 1
		//data @ 1 - (length-1)
		length := int(pdu[i])
		typ := pdu[i+1]

		//length should be more than 1 since there is a type byte
		if length < 1 {
			return nil, fmt.Errorf("invalid record length %d", length)
		}

		//do we have all the bytes for the payload?
		if (i + length) >= len(pdu) {
			return nil, fmt.Errorf("buffer overflow: want %v, have %v", (i + length), len(pdu))
		}

		start := i + 2
		end := start + length - 1
		bytes := pdu[start:end]

		//fmt.Printf("type %v, len %v, cur %v, start %v, end %v, bytes %v\n", typ, length, i, start, end, bytes)

		dec, ok := pduDecodeMap[typ]
		if !ok {
			fmt.Printf("ignored unsupported adv type %v\n", typ)
		} else {
			//have min length?
			if dec.minSz > len(bytes) {
				return nil, fmt.Errorf("adv type %v: min length %v, have %v\n", typ, dec.minSz, len(bytes))
			}

			//expecting array?
			if dec.arrayElementSz > 0 {
				arr, err := getArray(dec.arrayElementSz, bytes)

				//is this fatal?
				if err != nil {
					return nil, errors.Wrap(err, fmt.Sprintf("adv type %v", typ))
				}

				v, ok := m[dec.key].([]ble.UUID)
				if !ok {
					//nx key
					m[dec.key] = arr
				} else {
					m[dec.key] = append(v, arr...)
				}

			} else if dec.svcDataUUIDSz > 0 {
				sd := ble.ServiceData{UUID: bytes[:dec.svcDataUUIDSz], Data: bytes[dec.svcDataUUIDSz:]}
				v, ok := m[dec.key].([]ble.ServiceData)
				if !ok {
					//nx key
					m[dec.key] = []ble.ServiceData{sd}
				} else {
					m[dec.key] = append(v, sd)
				}
			} else {
				//we already checked for min length so just copy
				m[dec.key] = bytes
			}

		}

		i += (length + 1)
	}

	return m, nil
}
