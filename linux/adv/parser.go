package adv

import (
	"fmt"

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

type pduRecord struct {
	arrayElementSz int
	minSz          int
	key            string
}

var pduDecodeMap = map[byte]pduRecord{
	types.uuid16inc: pduRecord{
		2,
		2,
		"uuid16",
	},
	types.uuid16comp: pduRecord{
		2,
		2,
		"uuid16",
	},
	types.uuid32inc: pduRecord{
		4,
		4,
		"uuid32",
	},
	types.uuid32comp: pduRecord{
		4,
		4,
		"uuid32",
	},
	types.uuid128inc: pduRecord{
		16,
		16,
		"uuid128",
	},
	types.uuid128comp: pduRecord{
		16,
		16,
		"uuid128",
	},
	types.sol16: pduRecord{
		2,
		2,
		"sol16",
	},
	types.sol32: pduRecord{
		4,
		4,
		"sol32",
	},
	types.sol128: pduRecord{
		16,
		16,
		"sol128",
	},
	types.svc16: pduRecord{
		2,
		2,
		"svc16",
	},
	types.svc32: pduRecord{
		4,
		4,
		"svc32",
	},
	types.svc128: pduRecord{
		16,
		16,
		"svc128",
	},
	types.namecomp: pduRecord{
		0,
		1,
		"name",
	},
	types.nameshort: pduRecord{
		0,
		1,
		"name",
	},
	types.txpwr: pduRecord{
		0,
		1,
		"txpwr",
	},
	types.mfgdata: pduRecord{
		0,
		1,
		"mfg",
	},
	types.flags: pduRecord{
		0,
		1,
		"flags",
	},
}

func getArray(size int, bytes []byte) ([]interface{}, error) {
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
	arr := make([]interface{}, 0, count)

	for j := 0; j < len(bytes); j += size {
		o := bytes[j:(j + size)]
		arr = append(arr, o)
	}

	return arr, nil
}

func decode(pdu []byte) (map[string]interface{}, error) {
	if pdu == nil {
		return nil, fmt.Errorf("nil pdu")
	}

	m := make(map[string]interface{})
	for i := 0; (i + 1) < len(pdu); {
		//enough bytes for length & type?
		if (i + 1) >= len(pdu) {
			return nil, fmt.Errorf("not enough bytes for length+type")
		}

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

		dec, ok := pduDecodeMap[typ]
		if !ok {
			fmt.Printf("ignored unsupported adv type %v\n", typ)
			continue
		}

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
			m[dec.key] = arr
		} else {
			//we already checked for min length so just copy
			m[dec.key] = bytes
		}

		i += (length + 1)
	}

	if len(m) == 0 {
		fmt.Println("parsed adv to empty map")
	}

	return m, nil
}
