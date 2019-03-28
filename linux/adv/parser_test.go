package adv

import (
	"fmt"
	"reflect"
	"testing"
)

type testPdu struct {
	b []byte
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
		t.Fatalf("unsupported type")
	}

	//len == 0
	p := testPdu{}
	b := []byte{}
	p.add(typ, b)

	_, err := decode(p.bytes())
	if err == nil {
		return fmt.Errorf("len==0, no decode error")
	}

	//len % arraySz != 0
	p = testPdu{}
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

	_, err = decode(p.bytes())
	if err == nil {
		return fmt.Errorf("len%%size != 0, no decode error")
	}

	// len < elementSz
	p = testPdu{}
	b = []byte{}
	for i := 0; i < (dec.arrayElementSz - 1); i++ { //-1 for error
		bi := byte(i)
		b1 = append(b1, bi)
	}
	p.add(typ, b)

	_, err = decode(p.bytes())
	if err == nil {
		return fmt.Errorf("len<arrayElementSize, no decode error")
	}

	// len < minSz
	p = testPdu{}
	b = []byte{}
	for i := 0; i < (dec.minSz - 1); i++ { //-1 for error
		bi := byte(i)
		b1 = append(b1, bi)
	}
	p.add(typ, b)

	_, err = decode(p.bytes())
	if err == nil {
		return fmt.Errorf("len<minSz, no decode error")
	}

	return nil
}

func testArrayGood(typ byte, t *testing.T) error {
	dec, ok := pduDecodeMap[typ]
	if !ok || dec.arrayElementSz == 0 {
		t.Fatalf("unsupported type")
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
	p.add(typ, b)

	m, err := decode(p.bytes())
	if err != nil {
		return fmt.Errorf("decode error %v", err)
	}

	t.Logf("%+v", m)

	v, ok := m[dec.key]
	if !ok {
		return fmt.Errorf("missing key %v", dec.key)
	}

	vv, ok := v.([]interface{})
	if !ok {
		return fmt.Errorf("wrong type %v", reflect.TypeOf(v))
	}

	ok = reflect.DeepEqual(vv[0], b1)
	if !ok {
		return fmt.Errorf("mismatch @ 0")
	}
	ok = reflect.DeepEqual(vv[1], b2)
	if !ok {
		return fmt.Errorf("mismatch @ 1")
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
