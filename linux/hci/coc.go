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
	rxChannelSendTimeout = time.Second * 5 //arbitrary
	txCreditDelay        = time.Millisecond * 500
	cocRxChannelSize     = 8
	minDynamicCID        = 0x40
	maxDynamicCID        = 0xffff
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
	nextSrcCID   uint16 // 0x0040 to 0xFFFF
	remoteCidLut map[uint16]*cocInfo
	*Conn
	sync.RWMutex
	ble.Logger
}

func NewCoc(c *Conn, l ble.Logger) *coc {
	return &coc{
		Conn:         c,
		remoteCidLut: make(map[uint16]*cocInfo),
		Logger:       l,
		nextSrcCID:   minDynamicCID,
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

func (c *coc) Open(psm uint16) (ble.LECreditBasedConnection, error) {
	localCID, err := c.NextSourceCID()
	if err != nil {
		return nil, err
	}

	out := &LECreditBasedConnectionRequest{
		SourceCID: localCID,
		LEPSM:     psm,

		// TODO: all of these we will just ignore for now...
		MTU:            64,
		MPS:            64,
		InitialCredits: 2,
	}

	in := &LECreditBasedConnectionResponse{}
	if err := c.Signal(out, in); err != nil {
		return nil, err
	}

	if in.Result != 0 {
		return nil, fmt.Errorf("cocOpen/rsp result code 0x%x", in.Result)
	}

	conn, err := c.addChannel(localCID, in.DestinationCID, in.InitialCreditsCID, in.MTU, in.MPS)
	if err != nil {
		return nil, err
	}

	c.Infof("cocOpen localCID %v, remoteCID %v, credits %v OK", localCID, in.DestinationCID, in.InitialCreditsCID)

	return conn, nil
}

func (c *coc) addChannel(localCid, remoteCid, remoteCredits, remoteMtu, remoteMps uint16) (ble.LECreditBasedConnection, error) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.remoteCidLut[remoteCid]; ok {
		return nil, fmt.Errorf("cid %v already open", remoteCid)
	}

	i := &cocInfo{
		remoteCID: remoteCid,
		localCID:  localCid,
		mtu:       remoteMtu,
		mps:       remoteMps,
		credits:   remoteCredits,
	}
	c.remoteCidLut[remoteCid] = i

	return c.newCocWrapper(i), nil
}

func (c *coc) CloseChannel(cid uint16) error {
	c.Lock()
	defer c.Unlock()

	i, ok := c.remoteCidLut[cid]
	if !ok {
		return fmt.Errorf("cid %v not found", cid)
	}

	if i.rxChan != nil {
		close(i.rxChan)
	}

	delete(c.remoteCidLut, cid)
	return nil
}

func (c *coc) lookupLocalCID(localCid uint16) (*cocInfo, error) {
	for _, v := range c.remoteCidLut {
		if v.localCID == localCid {
			out := *v
			return &out, nil
		}
	}
	return nil, fmt.Errorf("%v not found", localCid)
}

func (c *coc) Subscribe(localCid uint16) (<-chan []byte, error) {
	c.Lock()
	defer c.Unlock()

	i, err := c.lookupLocalCID(localCid)
	if err != nil {
		return nil, err
	}

	if i.rxChan != nil {
		return nil, fmt.Errorf("cid %v has an existing subscriber", localCid)
	}

	i.rxChan = make(chan []byte, cocRxChannelSize)
	c.remoteCidLut[i.remoteCID] = i

	return i.rxChan, nil
}

func (c *coc) Unsubscribe(localCid uint16) error {
	c.Lock()
	defer c.Unlock()

	i, err := c.lookupLocalCID(localCid)
	if err != nil {
		return err
	}

	if i.rxChan != nil {
		close(i.rxChan)
		i.rxChan = nil
	}
	c.remoteCidLut[i.remoteCID] = i

	return nil
}

func (c *coc) Info(remoteCid uint16) (*cocInfo, error) {
	c.RLock()
	defer c.RUnlock()

	if v, ok := c.remoteCidLut[remoteCid]; ok {
		out := *v
		return &out, nil
	}

	return nil, fmt.Errorf("cid %v not found", remoteCid)
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
	v, ok := c.remoteCidLut[cid]
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

	// todo: maybe this should be at the top?
	defer c.returnCredit(cid, 1)

	if i.rxTransaction == nil {
		n := int(binary.LittleEndian.Uint16(data))
		c.Debugf("creating new rx transaction for %v bytes", n)
		i.rxTransaction = &cocRxTransaction{
			start:  time.Now(),
			buf:    new(bytes.Buffer),
			rxSize: n,
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
	if b.Len() >= i.rxTransaction.rxSize {
		// consider the txn done
		defer func() { i.rxTransaction = nil }()

		switch {
		case b.Len() > i.rxTransaction.rxSize:
			c.Warnf("rx overflow continuing anyways, have %v, want %v bytes", b.Len(), i.rxTransaction.rxSize)
			fallthrough
		case i.rxChan != nil:
			select {
			case i.rxChan <- b.Bytes():
				c.Debugf("sent %v bytes to subscriber", b.Len())
				// ok
			case <-time.After(rxChannelSendTimeout):
				return fmt.Errorf("subscriber channel send timeout")
			}

		default:
			c.Warnf("cid %v has no subscriber, discarding completed rx [%x]", cid, b.Bytes())
		} // switch

	} // if

	return nil
}

func (c *coc) returnCredit(localcid, credits uint16) error {
	// give the credit back to the remote
	sig := &LEFlowControlCredit{
		CID:     localcid,
		Credits: 1,
	}
	return c.Conn.Signal(sig, nil)
}

func (c *coc) send(cid uint16, data []byte) error {
	i, err := c.Info(cid)
	if err != nil {
		return err
	}

	// slice into mps-2 chunks
	if i.mps <= 2 {
		return fmt.Errorf("invalid mps %v for remote cid %v", i.mps, cid)
	}

	c.Debugf("attempting to send %v bytes on cid %v", len(data), cid)
	c.Debugf("connInfo %+v", i)

	// Vol 3, Pt A, 3.4.2 L2CAP SDU Length field (2 octets)
	// The first K-frame of the SDU shall contain the L2CAP SDU Length field that
	// shall specify the total number of octets in the SDU. The value shall not be
	// greater than the peer device's MTU for the channel. All subsequent K-frames
	// that are part of the same SDU shall not contain the L2CAP SDU Length field.

	br := bytes.NewReader(data)
	sent := 0
	for br.Len() > 0 {
		bb := make([]byte, i.mps-2)
		n, err := br.Read(bb)
		if err != nil {
			return err
		}

		// try and get a credit
		ok := false
		for i := 0; i < 10; i++ {
			err := c.DecrementCredits(cid, 1)
			c.Debugf("decrementCredit: attempt: %v, cid %v, credits %v, err: %v", i, cid, 1, err)
			if err == nil {
				ok = true
				break
			}
			// wait and try again
			time.Sleep(txCreditDelay)
		}

		if !ok {
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

		// sdu length (how many payload bytes)
		if err := binary.Write(buf, binary.LittleEndian, uint16(len(data))); err != nil {
			return err
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
		c.Debugf("sent [%x], progress %v/%v bytes", sent, len(data))
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

func (w *cocWrapper) Info() ble.LECreditBasedConnectionInfo {
	return ble.LECreditBasedConnectionInfo{
		RemoteCID: w.remoteCID,
		LocalCID:  w.localCID,
		MTU:       w.mtu,
		MPS:       w.mps,
	}
}
