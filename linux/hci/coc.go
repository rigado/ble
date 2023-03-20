package hci

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/rigado/ble"
)

const (
	cocRxSendTimeout = time.Second * 5 //arbitrary
	cocRxChannelSize = 8
	minDynamicCID    = 0x40
	maxDynamicCID    = 0xffff
)

type cocInfo struct {
	localCID, remoteCID uint16
	credits, mtu, mps   uint16
	rxChan              chan []byte
	rxTransaction       *cocRxTransaction
}

type cocRxTransaction struct {
	rxSize int
	start  time.Time
	buf    *bytes.Buffer
}

type coc struct {
	nextSrcCID uint16 // 0x0040 to 0xFFFF
	m          map[uint16]*cocInfo
	*Conn
	sync.RWMutex
	ble.Logger
}

func NewCoc(c *Conn, l ble.Logger) *coc {
	return &coc{
		Conn:       c,
		m:          make(map[uint16]*cocInfo),
		Logger:     l,
		nextSrcCID: minDynamicCID,
	}
}

func (c *coc) NextSourceCID() (uint16, error) {
	c.Lock()
	defer c.Unlock()

	switch {
	case c.nextSrcCID > maxDynamicCID:
		fallthrough
	case c.nextSrcCID < minDynamicCID:
		return 0, fmt.Errorf("invalid cid value %v, must be %v-%v", c.nextSrcCID, minDynamicCID, maxDynamicCID)
	}

	out := c.nextSrcCID
	c.nextSrcCID++
	return out, nil

}

func (c *coc) OpenChannel(localCid, remoteCid, remoteCredits, remoteMtu, remoteMps uint16) (ble.LECreditBasedConnection, error) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.m[remoteCid]; ok {
		return nil, fmt.Errorf("cid %v already open", remoteCid)
	}

	i := &cocInfo{
		remoteCID: remoteCid,
		localCID:  localCid,
		mtu:       remoteMtu,
		mps:       remoteMps,
		credits:   remoteCredits,
	}
	c.m[remoteCid] = i

	return c.newCocWrapper(i), nil
}

func (c *coc) CloseChannel(cid uint16) error {
	c.Lock()
	defer c.Unlock()

	i, ok := c.m[cid]
	if !ok {
		return fmt.Errorf("cid %v not found", cid)
	}

	if i.rxChan != nil {
		close(i.rxChan)
	}

	delete(c.m, cid)
	return nil
}

func (c *coc) lookupLocalCID(cid uint16) (*cocInfo, error) {
	for _, v := range c.m {
		if v.localCID == cid {
			out := *v
			return &out, nil
		}
	}
	return nil, fmt.Errorf("%v not found", cid)
}

func (c *coc) Subscribe(cid uint16) (<-chan []byte, error) {
	c.Lock()
	defer c.Unlock()

	i, err := c.lookupLocalCID(cid)
	if err != nil {
		return nil, err
	}

	if i.rxChan != nil {
		return nil, fmt.Errorf("cid %v has an existing subscriber", cid)
	}

	i.rxChan = make(chan []byte, cocRxChannelSize)

	return i.rxChan, nil
}

func (c *coc) Unsubscribe(cid uint16) error {
	c.Lock()
	defer c.Unlock()

	i, err := c.lookupLocalCID(cid)
	if err != nil {
		return err
	}

	if i.rxChan != nil {
		close(i.rxChan)
	}

	return nil
}

func (c *coc) Info(cid uint16) (*cocInfo, error) {
	c.RLock()
	defer c.RUnlock()

	if v, ok := c.m[cid]; ok {
		out := *v
		return &out, nil
	}

	return nil, fmt.Errorf("cid %v not found", cid)
}

func (c *coc) IncrementCredits(cid, credits uint16) error {
	c.Lock()
	defer c.Unlock()
	return c.unsafeModifyCredits(cid, credits, true)
}

func (c *coc) DecrementCredits(cid, credits uint16) error {
	c.Lock()
	defer c.Unlock()

	return c.unsafeModifyCredits(cid, credits, false)
}

func (c *coc) unsafeModifyCredits(cid, credits uint16, add bool) error {
	v, ok := c.m[cid]
	if !ok {
		return fmt.Errorf("cid %v not found", cid)
	}

	if add {
		// overflow check
		if nv := uint32(v.credits) + uint32(credits); nv > 0xffff {
			return fmt.Errorf("cid %v would overflow if adding %v credits", cid, credits)
		}
		v.credits += credits
	} else {
		// underflow check
		if credits > v.credits {
			return fmt.Errorf("cid %v would overflow if subtracting %v credits", cid, credits)
		}
		v.credits -= credits
	}

	return nil
}

func (c *coc) recieve(cid uint16, data []byte) error {
	c.Lock()
	defer c.Unlock()

	in := data
	i, err := c.lookupLocalCID(cid)
	if err != nil {
		return err
	}

	if i.rxTransaction == nil {
		n := int(binary.LittleEndian.Uint16(data))
		c.Infof("creating new rx transaction for %v bytes", n)
		i.rxTransaction = &cocRxTransaction{
			start: time.Now(),
			buf:   bytes.NewBuffer(make([]byte, n)),
		}

		in = data[2:] // advance the index past the u16 length
	}

	// write the bytes to the buffer
	b := i.rxTransaction.buf
	if b == nil {
		return fmt.Errorf("nil buffer")
	}

	if _, err = b.Write(in); err != nil {
		return err
	}

	c.Debugf("rx %v bytes on cid %v [%x]", len(in), cid, in)

	// are we done?
	if b.Len() == b.Cap() {
		if i.rxChan != nil {
			c.Debugf("dispatching completed rx of %v bytes to subscriber", b.Len())
			select {
			case i.rxChan <- b.Bytes():
				// ok
			case <-time.After(cocRxSendTimeout):
				c.Errorf("rx result channel timeout")
			}

		} else {
			c.Warnf("cid %v has no subscriber, discarding completed rx [%x]", cid, b.Len(), b.Bytes())
		}

		// delete it
		i.rxTransaction = nil
	}

	// give the credit back to the remote
	sig := &LEFlowControlCredit{
		CID:     cid,
		Credits: 1,
	}
	if err := c.Conn.Signal(sig, nil); err != nil {
		return err
	}

	return nil
}

func (c *coc) send(cid uint16, data []byte) error {
	i, err := c.Info(cid)
	if err != nil {
		return err
	}

	// MTU is the max SDU size the rx side can take
	if int(i.mtu) < len(data) {
		return fmt.Errorf("cid %v MTU exceeded, have %v bytes, want <= %v ", cid, len(data), i.mtu)
	}

	c.Debugf("attempting to send %v bytes on cid %v", len(data), cid)
	c.Debugf("connInfo %+v", i)

	// Vol 3, Pt A, 3.4.2 L2CAP SDU Length field (2 octets)
	// The first K-frame of the SDU shall contain the L2CAP SDU Length field that
	// shall specify the total number of octets in the SDU. The value shall not be
	// greater than the peer device's MTU for the channel. All subsequent K-frames
	// that are part of the same SDU shall not contain the L2CAP SDU Length field.

	first := true
	br := bytes.NewReader(data)
	sent := 0
	for br.Len() > 0 {
		readSz := i.mps
		if first {
			readSz = i.mps - 2
		}
		bb := make([]byte, readSz)

		n, err := br.Read(bb)
		if err != nil {
			return err
		}

		// try and get a credit
		gotCredit := false
		for i := 0; i < 10; i++ {
			err := c.DecrementCredits(cid, 1)
			c.Debugf("decrementCredit: attempt: %v, cid %v, credits %v, err: %v", i, cid, 1, err)
			if err == nil {
				gotCredit = true
				break
			}
			// wait and try again
			time.Sleep(time.Millisecond * 500)
		}
		if !gotCredit {
			return fmt.Errorf("unable to get credit for cid %v", cid)
		}

		buf := bytes.NewBuffer(make([]byte, 0))
		// l2cap pdu len
		if err := binary.Write(buf, binary.LittleEndian, uint16(2+n)); err != nil {
			return err
		}

		// l2cap channel id
		if err := binary.Write(buf, binary.LittleEndian, cid); err != nil {
			return err
		}

		// sdu length (total input payload sz)
		if first {
			if err := binary.Write(buf, binary.LittleEndian, uint16(len(data))); err != nil {
				return err
			}
			first = false
		}

		// information payload
		if err := binary.Write(buf, binary.LittleEndian, bb[:n]); err != nil {
			return err
		}
		txbb := buf.Bytes()
		// send it
		if _, err := c.writePDU(txbb); err != nil {
			return err
		}

		sent += n
		c.Infof("Sent %v/%v bytes", sent, len(data))
	}

	return nil
}

func (c *coc) newCocWrapper(i *cocInfo) ble.LECreditBasedConnection {
	return &cocWrapper{c, i}
}

type cocWrapper struct {
	*coc
	*cocInfo
}

func (w *cocWrapper) Send(bb []byte) error {
	return w.coc.send(w.remoteCID, bb)
}

func (w *cocWrapper) Subscribe() (<-chan []byte, error) {
	return w.coc.Subscribe(w.localCID)
}

func (w *cocWrapper) Unsubscribe() error {
	return w.coc.Unsubscribe(w.localCID)
}

func (w *cocWrapper) Close() error {
	return w.coc.CloseChannel(w.remoteCID)
}
