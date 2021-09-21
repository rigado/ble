package parser

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/rigado/ble"
)

type testPdu struct {
	b []byte
}

func (t *testPdu) addBad(recTyp byte, badRecLen byte, recBytes []byte) {
	t.b = append(t.b, badRecLen, recTyp)
	t.b = append(t.b, recBytes...)
}

func (t *testPdu) add(recTyp byte, recBytes []byte) {
	lb := byte(len(recBytes) + 1)
	t.b = append(t.b, lb, recTyp)
	t.b = append(t.b, recBytes...)
}

func (t *testPdu) bytes() []byte {
	return t.b
}

func testArrayBad(typ byte, t *testing.T) error {
	dec, ok := pduDecodeMap[typ]
	if !ok || dec.arrayElementSz == 0 {
		t.Fatalf("unsupported type %v", typ)
	}

	//len % arraySz != 0
	var err error
	var b []byte

	p := testPdu{}
	b1 := []byte{}
	b2 := []byte{}
	for i := 0; i < dec.arrayElementSz; i++ {
		bi := byte(i)
		b1 = append(b1, bi)
		b2 = append(b2, 255-bi)
	}

	b = append(b1, b2...)
	b = append(b, 0xbb) //appending extra byte here!
	p.add(typ, b)

	_, err = Parse(p.bytes())
	if err == nil {
		return fmt.Errorf("len%%size != 0, no decode error")
	}

	// len < elementSz
	p = testPdu{}
	b = []byte{}
	for i := 0; i < (dec.arrayElementSz - 1); i++ { //-1 for error
		bi := byte(i)
		b = append(b, bi)
	}
	p.add(typ, b)

	t.Logf("type %v, bytes %x", typ, b)

	_, err = Parse(p.bytes())
	if err == nil {
		return fmt.Errorf("len<arrayElementSize, no decode error")
	}

	// len < minSz
	p = testPdu{}
	b = []byte{}
	for i := 0; i < (dec.minSz - 1); i++ { //-1 for error
		bi := byte(i)
		b = append(b1, bi)
	}
	p.add(typ, b)

	_, err = Parse(p.bytes())
	if err == nil {
		return fmt.Errorf("len<minSz, no decode error")
	}

	//corrupt encoding (bad length)
	p = testPdu{}
	b1 = []byte{}
	b2 = []byte{}

	for i := 0; i < dec.arrayElementSz; i++ {
		bi := byte(i)
		b1 = append(b1, bi)
		b2 = append(b2, 128+bi)
	}

	//+32
	b = append(b1, b2...)
	badLength := byte(len(b) + 32)
	p.addBad(typ, badLength, b)

	_, err = Parse(p.bytes())
	if err == nil {
		return fmt.Errorf("corrupt length +32, no decode error")
	}

	//255
	p = testPdu{}
	p.addBad(typ, 255, b)
	_, err = Parse(p.bytes())
	if err == nil {
		return fmt.Errorf("corrupt length 255, no decode error")
	}

	return nil
}

func testArrayGood(typ byte, t *testing.T) error {
	dec, ok := pduDecodeMap[typ]
	if !ok || dec.arrayElementSz == 0 {
		t.Fatalf("unsupported type %v", typ)
	}

	p := testPdu{}
	b1 := []byte{}
	b2 := []byte{}
	b3 := []byte{}

	for i := 0; i < dec.arrayElementSz; i++ {
		bi := byte(i)
		b1 = append(b1, bi)
		b2 = append(b2, 128+bi)
		b3 = append(b3, 255-bi)
	}

	b := append(b1, b2...)
	b = append(b, b3...)
	p.add(typ, b)

	m, err := Parse(p.bytes())
	if err != nil {
		return fmt.Errorf("decode error %v", err)
	}

	t.Logf("%+v", m)

	v, ok := m[dec.key]
	if !ok {
		return fmt.Errorf("missing key %v", dec.key)
	}

	//check type
	vv, ok := v.([]ble.UUID)
	if !ok {
		return fmt.Errorf("wrong type %v", reflect.TypeOf(v))
	}

	//check the count
	if len(vv) != 3 {
		return fmt.Errorf("uuid count mismatch, exp 3, have %v", len(vv))
	}

	//check contents
	ok = reflect.DeepEqual(vv[0], ble.UUID(b1))
	if !ok {
		return fmt.Errorf("mismatch @ 0")
	}
	ok = reflect.DeepEqual(vv[1], ble.UUID(b2))
	if !ok {
		return fmt.Errorf("mismatch @ 1")
	}
	ok = reflect.DeepEqual(vv[2], ble.UUID(b3))
	if !ok {
		return fmt.Errorf("mismatch @ 2")
	}

	return nil
}

func TestParserArrays(t *testing.T) {
	types := []byte{
		types.uuid16inc,
		types.uuid16comp,
		types.uuid32inc,
		types.uuid32comp,
		types.uuid128inc,
		types.uuid128comp,
		types.sol16,
		types.sol32,
		types.sol128,
	}

	for _, v := range types {
		err := testArrayGood(v, t)
		t.Logf("adv type %v, testArrayGood err %v", v, err)
		if err != nil {
			t.Fatal(err)
		}

		err = testArrayBad(v, t)
		t.Logf("adv type %v, testArrayBad err %v", v, err)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func testNonArrayGood(typ byte, t *testing.T) error {
	dec, ok := pduDecodeMap[typ]
	if !ok || dec.arrayElementSz != 0 {
		t.Fatalf("unsupported type %v", typ)
	}

	p := testPdu{}
	b := []byte{}
	for i := 0; i < dec.minSz; i++ {
		bi := byte(i)
		b = append(b, bi)
	}

	p.add(typ, b)
	m, err := Parse(p.bytes())
	if err != nil {
		return fmt.Errorf("decode error %v", err)
	}

	t.Logf("%+v", m)
	v, ok := m[dec.key]
	if !ok {
		return fmt.Errorf("missing key %v", dec.key)
	}

	vv, ok := v.(interface{})
	if !ok {
		return fmt.Errorf("wrong type %v", reflect.TypeOf(v))
	}

	ok = reflect.DeepEqual(vv, b)
	if !ok {
		return fmt.Errorf("mismatch")
	}

	return nil
}

func testNonArrayBad(typ byte, t *testing.T) error {
	dec, ok := pduDecodeMap[typ]
	if !ok || dec.arrayElementSz != 0 {
		t.Fatalf("unsupported type %v", typ)
	}

	var err error
	var p testPdu
	var b []byte

	// len < minSz (skip if resulting in len == 0)
	if dec.minSz > 1 {
		p = testPdu{}
		b = []byte{}
		for i := 0; i < (dec.minSz - 1); i++ { //-1 for error
			bi := byte(i)
			b = append(b, bi)
		}
		p.add(typ, b)

		_, err = Parse(p.bytes())
		if err == nil {
			return fmt.Errorf("len<minSz, no decode error")
		}
	}

	//corrupt encoding (bad length)
	p = testPdu{}
	b = []byte{}

	for i := 0; i < dec.arrayElementSz; i++ {
		bi := byte(i)
		b = append(b, bi)
	}

	//+32
	badLength := byte(len(b) + 32)
	p.addBad(typ, badLength, b)

	_, err = Parse(p.bytes())
	if err == nil {
		return fmt.Errorf("corrupt length +32, no decode error")
	}

	//255
	p = testPdu{}
	p.addBad(typ, 255, b)
	_, err = Parse(p.bytes())
	if err == nil {
		return fmt.Errorf("corrupt length 255, no decode error")
	}

	return nil
}

func TestParserNonArrays(t *testing.T) {
	types := []byte{
		types.flags,
		types.nameshort,
		types.namecomp,
		types.txpwr,
		types.mfgdata,
		// types.svc16,
		// types.svc32,
		// types.svc128,
	}

	for _, v := range types {
		err := testNonArrayGood(v, t)
		t.Logf("adv type %v, testArrayGood err %v", v, err)
		if err != nil {
			t.Fatal(err)
		}

		err = testNonArrayBad(v, t)
		t.Logf("adv type %v, testArrayBad err %v", v, err)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestParserCombined(t *testing.T) {
	//uuid16 + uuid128 + flags
	u16 := []byte{1, 2, 3, 4}
	u128 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	flags := []byte{0xcd}

	p := testPdu{}
	p.add(types.flags, flags)
	p.add(types.uuid16comp, u16)
	p.add(types.uuid128comp, u128)

	m, err := Parse(p.bytes())
	if err != nil {
		t.Fatalf("combined adv decode err %v", err)
	}
	t.Log(m, err)
	ms, msok := m[keys.services].([]ble.UUID)
	mf, mfok := m[keys.flags].([]byte)

	if !msok || !mfok {
		t.Fatalf("decoded map is missing key")
	}

	//flag?
	if !reflect.DeepEqual(mf, flags) {
		t.Fatalf("flags mismatch")
	}

	//count?
	if len(ms) != 3 {
		t.Fatalf("incorrect service count exp %v, got %v", 3, len(ms))
	}

	//uuid16
	ok0 := reflect.DeepEqual(ms[0], ble.UUID(u16[0:2]))
	if !ok0 {
		t.Fatalf("uuid16 mismatch at idx 0")
	}
	ok1 := reflect.DeepEqual(ms[1], ble.UUID(u16[2:]))
	if !ok1 {
		t.Fatalf("uuid16 mismatch at idx 1")
	}

	//uuid128
	ok3 := reflect.DeepEqual(ms[2], ble.UUID(u128))
	if !ok3 {
		t.Fatalf("uuid128 mismatch at idx 0")
	}

}

/*
func TestIBeacon(t *testing.T) {
	u128 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	p, _ := adv.NewPacket(adv.Flags(123), adv.IBeacon(u128, 12345, 45678, 56))
	b := p.Bytes()
	m, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}

	if len(m) != 2 {
		t.Fatalf("map has %v keys, exp %v", len(m), 2)
	}

	//check flags
	ff, ok := m[keys.flags].([]byte)
	if !ok {
		t.Fatalf("flags missing")
	}

	fexp := b[2:3] //flags at idx 0 (len), 1 (type 0x01), 3 (data 1 byte)
	fok := reflect.DeepEqual(ff, fexp)
	if !fok {
		t.Fatalf("mismatch:\nexp %v %v\ngot %v %v", fexp, reflect.TypeOf(fexp), ff, reflect.TypeOf(ff))
	}

	//check mfg
	md, ok := m[keys.mfgdata].([]byte)
	if !ok {
		t.Fatalf("mfgdata missing")
	}

	mdexp := b[5:] //flags at idx 0-2, 3 (len), 4 (type 0xff), 5 -... (data)
	mdok := reflect.DeepEqual(md, mdexp)
	if !mdok {
		t.Fatalf("mismatch:\nexp %v %v\ngot %v %v", mdexp, reflect.TypeOf(mdexp), md, reflect.TypeOf(md))
	}
}
*/

func testServiceData(typ byte, dl int, t *testing.T) error {
	if dl < 0 {
		return fmt.Errorf("invalid data length")
	}

	switch typ {
	case types.svc16:
	case types.svc32:
	case types.svc128:

	default:
		return fmt.Errorf("non-svcData type %v", typ)
	}

	dec, _ := pduDecodeMap[typ]

	p := testPdu{}
	uuid := make([]byte, dec.minSz)
	data := make([]byte, dl)
	for i := range uuid {
		uuid[i] = byte(i)
	}
	for i := range data {
		data[i] = byte(255 - i)
	}

	p.add(typ, append(uuid, data...))

	m, err := Parse(p.bytes())
	if err != nil {
		return fmt.Errorf("decode error %v", err)
	}

	if len(m) != 1 {
		return fmt.Errorf("map has %v keys, exp %v", len(m), 1)
	}

	t.Logf("%+v", m)

	//check service data map exists?
	sdm, ok := m[dec.key].(map[string]interface{})
	if !ok {
		return fmt.Errorf("svc data map invalid %v", dec.key)
	}

	// sd exists for uuid?
	ai, ok := sdm[ble.UUID(uuid).String()].([]interface{})
	if !ok {
		return fmt.Errorf("svc data for %v not found", uuid)
	}

	// sd has bytes?
	sd, ok := ai[0].([]byte)
	if !ok {
		return fmt.Errorf("no ble svc data found, got %v", reflect.TypeOf(ai[0]))
	}

	// sd data correct?
	sdok := reflect.DeepEqual(sd, data)
	if !sdok {
		return fmt.Errorf("svc data mismatch:\nexp %v %v\ngot %v %v", data, reflect.TypeOf(data), sd, reflect.TypeOf(sd))
	}

	return nil
}

func TestServiceData(t *testing.T) {
	types := []byte{types.svc16, types.svc32, types.svc128}
	for _, v := range types {
		err := testServiceData(v, 16, t)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func Test_FieldCombo(t *testing.T) {
	u128 := []byte{0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3}
	md := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	p := testPdu{}
	p.add(types.flags, []byte{0x12})
	p.add(types.uuid16comp, []byte{1, 2, 3, 4})
	p.add(types.uuid128comp, u128)
	p.add(types.mfgdata, md)
	p.add(types.svc16, []byte{1, 2, 3, 4, 5, 6, 7})

	m, err := Parse(p.bytes())
	if err != nil {
		t.Fatal(err)
	}

	var v, exp interface{}
	t.Log(m)

	// check flags
	exp = []byte{0x12}
	v = m[keys.flags]
	if !reflect.DeepEqual(v, exp) {
		t.Fatalf("have %v (%T), want %v (%T)", v, v, exp, exp)
	}

	// check services
	exp = []ble.UUID{
		ble.UUID16(0x0201),
		ble.UUID16(0x0403),
		ble.UUID(u128),
	}

	v = m[keys.services]
	if !reflect.DeepEqual(v, exp) {
		t.Fatalf("have %v (%T), want %v (%T)", v, v, exp, exp)
	}

	// check mfg
	exp = md
	v = m[keys.mfgdata]
	if !reflect.DeepEqual(v, exp) {
		t.Fatalf("have %v (%T), want %v (%T)", v, v, exp, exp)
	}

	// check svc data
	exp = map[string]interface{}{
		"0201": []interface{}{[]byte{3, 4, 5, 6, 7}},
	}
	v = m[keys.serviceData]
	if !reflect.DeepEqual(v, exp) {
		t.Fatalf("have %v (%T), want %v (%T)", v, v, exp, exp)
	}
}

func Test_ParseErrors(t *testing.T) {
	// missing a byte on uuid128
	u128bad := []byte{0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3}
	md := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	p := testPdu{}
	p.add(types.uuid128comp, u128bad)
	p.add(types.mfgdata, md)

	_, err := Parse(p.bytes())
	if err == nil {
		t.Fatal("expect error on bad input length")
	}

	// empty uuid128 field, check mfg ok
	p = testPdu{}
	p.add(types.uuid128comp, []byte{})
	p.add(types.mfgdata, md)

	m, err := Parse(p.bytes())
	if err != nil {
		t.Fatal(err)
	}

	var v, exp interface{}
	t.Log(m)

	// check mfg
	exp = md
	v = m[keys.mfgdata]
	if !reflect.DeepEqual(v, exp) {
		t.Fatalf("have %v (%T), want %v (%T)", v, v, exp, exp)
	}

	if v, ok := m[keys.services]; ok {
		t.Fatalf("service field present on empty input, got %v", v)
	}

	// empty name on end
	p = testPdu{}
	p.add(types.mfgdata, md)
	p.add(types.namecomp, []byte{})

	m, err = Parse(p.bytes())
	if err != nil {
		t.Fatal(err)
	}

	// check mfg
	exp = md
	v = m[keys.mfgdata]
	if !reflect.DeepEqual(v, exp) {
		t.Fatalf("have %v (%T), want %v (%T)", v, v, exp, exp)
	}

	if v, ok := m[keys.localName]; ok {
		t.Fatalf("name field present on empty input, got %v", v)
	}

}
