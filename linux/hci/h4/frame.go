package h4

import (
	"fmt"
	"time"
)

const (
	headerOffsetH4Event    = 0
	headerOffsetEventType  = 1
	headerOffsetDataLength = 2
	headerLength           = 3
)

type frame struct {
	b       []byte
	timeout time.Time
	out     chan []byte
}

func newFrame(c chan []byte) *frame {
	fr := &frame{
		b:   make([]byte, 0, 256),
		out: c,
	}

	return fr
}

func (f *frame) Assemble(b []byte) {
	switch {
	case len(b) == 0:
		// nothing to look at
		return

	case !f.timeout.IsZero() && time.Now().After(f.timeout):
		//timed out
		fallthrough
	case f.b == nil:
		//lazy init
		f.reset()

	default:
		// ok
	}

	if len(f.b) == 0 {
		err := f.waitStart(b)
		if err != nil {
			return
		}
	} else {
		bb := make([]byte, len(b))
		copy(bb, b)
		f.b = append(f.b, bb...)
	}

	// fmt.Printf("in  %0x\n", b)
	// fmt.Printf("buf %0x\n", b)

	rf, err := f.rawFrame()
	if err != nil {
		return
	}
	out := make([]byte, len(rf))
	copy(out, rf)
	// fmt.Printf("out: %0x\n", out)
	f.out <- out

	// shift
	if len(f.b) > len(rf) {
		rem := make([]byte, len(f.b[len(rf):]))
		copy(rem, f.b[len(rf):])
		f.reset()
		f.Assemble(rem)
	} else {
		f.reset()
	}
}

func (f *frame) reset() {
	f.b = make([]byte, 0, 256)
	f.timeout = time.Time{}
}

func (f *frame) waitStart(b []byte) error {
	// find the start byte
	var i int
	var v byte
	var ok bool
	for i, v = range b {
		if v == BT_H4_EVT_PKT {
			ok = true
			f.timeout = time.Now().Add(time.Millisecond * 500)
			break
		}
	}

	if !ok {
		return fmt.Errorf("couldnt find start byte")
	}

	bb := make([]byte, len(b[i:]))
	copy(bb, b[i:])
	f.b = append(f.b, bb...)
	return nil
}

func (f *frame) event() (byte, error) {
	if len(f.b) <= headerOffsetEventType {
		return 0, fmt.Errorf("not enough bytes")
	}

	return f.b[headerOffsetEventType], nil
}

func (f *frame) dataLength() (byte, error) {
	if len(f.b) <= headerOffsetDataLength {
		return 0, fmt.Errorf("not enough bytes")
	}

	return f.b[headerOffsetDataLength], nil
}

func (f *frame) data() ([]byte, error) {
	dl, err := f.dataLength()
	if err != nil {
		return nil, err
	}

	tl := int(dl) + headerLength
	if len(f.b) < tl {
		return nil, fmt.Errorf("not enough bytes")
	}
	return f.b[headerLength:tl], nil
}

func (f *frame) rawFrame() ([]byte, error) {
	dl, err := f.dataLength()
	if err != nil {
		return nil, err
	}

	dli := int(dl)
	tl := dli + headerLength
	if len(f.b) < tl {
		return nil, fmt.Errorf("not enough bytes")
	}
	return f.b[:tl], nil
}
