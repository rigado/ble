package parser

import (
	"fmt"

	"errors"
	"github.com/rigado/ble"
)

var EmptyOrNilPdu = errors.New("nil/empty pdu")

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
	types.uuid16inc: {
		2,
		2,
		0,
		keys.services,
	},
	types.uuid16comp: {
		2,
		2,
		0,
		keys.services,
	},
	types.uuid32inc: {
		4,
		4,
		0,
		keys.services,
	},
	types.uuid32comp: {
		4,
		4,
		0,
		keys.services,
	},
	types.uuid128inc: {
		16,
		16,
		0,
		keys.services,
	},
	types.uuid128comp: {
		16,
		16,
		0,
		keys.services,
	},
	types.sol16: {
		2,
		2,
		0,
		keys.solicited,
	},
	types.sol32: {
		4,
		4,
		0,
		keys.solicited,
	},
	types.sol128: {
		16,
		16,
		0,
		keys.solicited,
	},
	types.svc16: {
		0,
		2,
		2,
		keys.serviceData,
	},
	types.svc32: {
		0,
		4,
		4,
		keys.serviceData,
	},
	types.svc128: {
		0,
		16,
		16,
		keys.serviceData,
	},
	types.namecomp: {
		0,
		1,
		0,
		keys.localName,
	},
	types.nameshort: {
		0,
		1,
		0,
		keys.localName,
	},
	types.txpwr: {
		0,
		1,
		0,
		keys.txpwr,
	},
	types.mfgdata: {
		0,
		1,
		0,
		keys.mfgdata,
	},
	types.flags: {
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
	if len(bytes) == 0 {
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

func Parse(pdu []byte) (map[string]interface{}, error) {
	if len(pdu) == 0 {
		return nil, EmptyOrNilPdu
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
			return m, fmt.Errorf("invalid record length %v, idx %v", length, i)
		}

		//do we have all the bytes for the payload?
		if (i + length) >= len(pdu) {
			return m, fmt.Errorf("buffer overflow: want %v, have %v, idx %v", i+length, len(pdu), i)
		}

		start := i + 2
		end := start + length - 1
		bytes := make([]byte, len(pdu[start:end]))
		copy(bytes, pdu[start:end])
		dec, ok := pduDecodeMap[typ]
		if ok && len(bytes) != 0 {
			//have min length?
			if dec.minSz > len(bytes) {
				return m, fmt.Errorf("adv type %v: min length %v, have %v, idx %v", typ, dec.minSz, len(bytes), i)
			}

			//expecting array?
			if dec.arrayElementSz > 0 {
				arr, err := getArray(dec.arrayElementSz, bytes)

				//is this fatal?
				if err != nil {
					return m, fmt.Errorf("adv type %v, idx %v: %w", typ, i, err)
				}

				v, ok := m[dec.key].([]ble.UUID)
				if !ok {
					//nx key
					m[dec.key] = arr
				} else {
					m[dec.key] = append(v, arr...)
				}

			} else if dec.svcDataUUIDSz > 0 {
				su := ble.UUID(bytes[:dec.svcDataUUIDSz]).String()
				sd := bytes[dec.svcDataUUIDSz:]

				// service data map?
				msd, ok := m[dec.key].(map[string]interface{})
				if !ok {
					msd = make(map[string]interface{})
				}

				// add/append
				arr, ok := msd[su].([]interface{})
				if !ok {
					msd[su] = []interface{}{sd}
				} else {
					msd[su] = append(arr, sd)
				}

				//save result
				m[dec.key] = msd
			} else {
				//we already checked for min length so just copy
				writeOrAppendBytes(m, dec.key, bytes)
			}
		}

		i += length + 1
	}

	return m, nil
}

func writeOrAppendBytes(m map[string]interface{}, key string, data []byte) {
	if _, ok := m[key]; !ok {
		m[key] = data
	} else {
		if d, ok := m[key].([]byte); ok {
			if key == keys.mfgdata {
				//mfg data contains the company id again in the scan response
				//strip that out
				data = data[2:]
			}
			d = append(d, data...)
			m[key] = d
		} else {
			//just overwrite it if the data type is wrong
			m[key] = data
		}

	}
}
