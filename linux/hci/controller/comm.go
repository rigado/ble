package controller

import (
	"fmt"
	"github.com/rigado/ble/linux/hci"
	"io"
	"strings"
	"time"
)

// Send ...
func (h *HCI) Send(c hci.Command, r hci.CommandRP) error {
	b, err := h.send(c)
	if err != nil {
		return err
	}
	if len(b) > 0 && b[0] != 0x00 {
		return hci.ErrCommand(b[0])
	}
	if r != nil {
		return r.Unmarshal(b)
	}
	return nil
}

func (h *HCI) send(c hci.Command) ([]byte, error) {
	if h.err != nil {
		return nil, h.err
	}

	p := &pkt{c, make(chan []byte)}

	// get buffer w/timeout
	var b []byte
	select {
	case <-h.done:
		return nil, fmt.Errorf("hci closed")
	case b = <-h.chCmdBufs:
		//ok
	case <-time.After(chCmdBufTimeout):
		err := fmt.Errorf("chCmdBufs get timeout")
		h.dispatchError(err)
		return nil, err
	}

	b[0] = hci.PktTypeCommand // HCI header
	b[1] = byte(c.OpCode())
	b[2] = byte(c.OpCode() >> 8)
	b[3] = byte(c.Len())
	if err := c.Marshal(b[4:]); err != nil {
		h.close(fmt.Errorf("hci: failed to marshal cmd"))
	}

	h.muSent.Lock()
	_, ok := h.sent[c.OpCode()]
	if ok {
		h.muSent.Unlock()
		return nil, fmt.Errorf("command with opcode %v pending", c.OpCode())
	}

	h.sent[c.OpCode()] = p
	h.muSent.Unlock()

	if !h.isOpen() {
		return nil, fmt.Errorf("hci closed")
	} else if n, err := h.skt.Write(b[:4+c.Len()]); err != nil {
		h.close(fmt.Errorf("hci: failed to send cmd"))
	} else if n != 4+c.Len() {
		h.close(fmt.Errorf("hci: failed to send whole cmd pkt to hci socket"))
	}

	var ret []byte
	var err error

	// emergency timeout to prevent calls from locking up if the HCI
	// interface doesn't respond.  Responses here should normally be fast
	// a timeout indicates a major problem with HCI.
	select {
	case <-time.After(10 * time.Second):
		err = fmt.Errorf("hci: no response to command, hci connection failed")
		ret = nil
	case <-h.done:
		err = h.err
		ret = nil
	case b := <-p.done:
		err = nil
		ret = b
	}

	// clear sent table when done, we sometimes get command complete or
	// command status messages with no matching send, which can attempt to
	// access stale packets in sent and fail or lock up.
	h.muSent.Lock()
	delete(h.sent, c.OpCode())
	h.muSent.Unlock()

	return ret, err
}

func (h *HCI) sktProcessLoop() {

	defer h.cleanup()
	defer h.dispatchError(h.err)

	for {
		var p []byte
		var ok bool

		select {
		case <-h.done:
			fmt.Println("close requested")
			h.err = io.EOF
			return

		case p, ok = <-h.sktRxChan:
			if !ok {
				fmt.Println("socket rx closed")
				h.err = io.EOF
				return
			}
			// will process the bytes below
		}

		if err := h.handlePkt(p); err != nil {
			// Some bluetooth devices may append vendor specific packets at the last,
			// in this case, simply ignore them.
			if strings.HasPrefix(err.Error(), "unsupported vendor packet:") {
				//todo: change this back to a logger
				fmt.Printf("skt: %v\n", err)
			} else {
				h.err = fmt.Errorf("skt handle error: %v", err)
				return
			}
		}
	}
}

func (h *HCI) sktReadLoop() {
	defer func() {
		fmt.Println("sktRxLoop done")
		close(h.sktRxChan)
	}()

	b := make([]byte, 4096)

	for {
		n, err := h.skt.Read(b)

		switch {
		case n == 0 && err == nil:
			// read timeout
			select {
			case <-h.done:
				//exit!
				return
			default:
				continue
			}

		//callers depend on detecting io.EOF, don't wrap it.
		case err == io.EOF:
			h.err = err
			return

		case err != nil:
			h.err = fmt.Errorf("skt read error: %v", err)
			return

		default:
			// ok
			p := make([]byte, n)
			copy(p, b)
			h.sktRxChan <- p
		}
	}
}

func (h *HCI) close(err error) error {
	h.err = err
	return h.skt.Close()
}
