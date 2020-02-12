package connection

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rigado/ble/linux/hci"
	"io"
	"time"

	"github.com/rigado/ble"
	"github.com/rigado/ble/linux/hci/cmd"
	"github.com/rigado/ble/linux/hci/evt"
)

// Conn ...
type Conn struct {
	ctrl hci.Controller

	ctx context.Context
	cancel context.CancelFunc

	param evt.LEConnectionComplete

	// While MTU is the maximum size of payload data that the upper layer (ATT)
	// can accept, the MPS is the maximum PDU payload size this L2CAP implementation
	// supports. When segmentation is not used, the MPS should be made to the same
	// values of MTUs [Vol 3, Part A, 1.4].
	//
	// For LE-U logical transport, the L2CAP implementations should support
	// a minimum of 23 bytes, which are also the default values before the
	// upper layer (ATT) optionally reconfigures them [Vol 3, Part A, 3.2.8].
	rxMTU int
	txMTU int
	rxMPS int

	// Signaling MTUs are The maximum size of command information that the
	// L2CAP layer entity is capable of accepting.
	// A L2CAP implementations supporting LE-U should support at least 23 bytes.
	// Currently, we support 512 bytes, which should be more than sufficient.
	// The sigTxMTU is discovered via when we sent a signaling pkt that is
	// larger thean the remote device can handle, and get a response of "Command
	// Reject" indicating "Signaling MTU exceeded" along with the actual
	// signaling MTU [Vol 3, Part A, 4.1].
	sigRxMTU int
	sigTxMTU int

	sigSent chan []byte
	// smpSent chan []byte

	chInPkt chan Packet
	chInPDU chan Pdu

	chDone chan struct{}
	// Host to Controller Data Flow Control pkt-based Data flow control for LE-U [Vol 2, Part E, 4.1.1]
	// chSentBufs tracks the HCI buffer occupied by this connection.
	txBuffer hci.BufferPool

	// sigID is used to match responses with signaling requests.
	// The requesting device sets this field and the responding device uses the
	// same value in its response. Within each signalling channel a different
	// Identifier shall be used for each successive command. [Vol 3, Part A, 4]
	sigID uint8

	// leFrame is set to be true when the LE Credit based flow control is used.
	leFrame bool

	smp hci.SmpManager
}

type Encrypter interface {
	Encrypt() error
}

func New(ctrl hci.Controller, param evt.LEConnectionComplete) *Conn {

	c := &Conn{
		ctrl:  ctrl,
		param: param,

		rxMTU: ble.DefaultMTU,
		txMTU: ble.DefaultMTU,

		rxMPS: ble.DefaultMTU,

		sigRxMTU: ble.MaxMTU,
		sigTxMTU: ble.DefaultMTU,

		chInPkt: make(chan Packet, 16),
		chInPDU: make(chan Pdu, 16),

		txBuffer: ctrl.RequestBufferPool(),

		chDone: make(chan struct{}),
	}

	c.ctx, c.cancel = context.WithCancel(ctrl.Context())

	smp, err := c.ctrl.RequestSmpManager(hci.DefaultSmpConfig)
	if err == nil {
		c.smp = smp
		c.initPairingContext()
		c.smp.SetWritePDUFunc(c.writePDU)
		c.smp.SetEncryptFunc(c.encrypt)
	}

	return c
}

func (c *Conn) Run() {
	for {
		var pkt Packet
		var ok bool

		select {
		//this channel is closed either when the connection is disconnect
		//or if the HCI instance is closed
		case pkt, ok = <-c.chInPkt:
			if !ok {
				hci.Logger.Debug("c.chInPkt is closed; exiting")
				return
			}
		//todo: is this really necessary??
		case <-time.After(time.Minute * 10):
			hci.Logger.Info("recombine timed out")
			c.ctrl.DispatchError(fmt.Errorf("connection.Run idle timeout"))
			return
		case <-c.ctx.Done():
			hci.Logger.Info("connection context cancelled", c.param.ConnectionHandle())
			//c.ctrl.DispatchError(fmt.Errorf("connection cancelled: %s", c.ctx.Err()))
			return
		}

		if err := c.recombine(pkt); err != nil {
			if err != io.EOF {
				err = errors.Wrap(err, "recombine")
				c.ctrl.DispatchError(err)

				//attempt to cleanup
				//todo: this is the job of hci
				//if err := c.hci.cleanupConnectionHandle(c.param.ConnectionHandle()); err != nil {
				//	fmt.Printf("recombine cleanup: %v\n", err)
				//}
			} else {
				fmt.Println("recombine non io.EOF error:", err)
			}
			close(c.chInPDU)
			return
		}
	}
}

// Context returns the context that is used by this Conn.
func (c *Conn) Context() context.Context {
	return c.ctx
}

// SetContext sets the context that is used by this Conn.
func (c *Conn) SetContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *Conn) initPairingContext() {
	smp := c.smp

	la := c.LocalAddr().Bytes()
	lat := uint8(0x00)
	if (la[0] & 0xc0) == 0xc0 {
		lat = 0x01
	}
	ra := c.RemoteAddr().Bytes()
	rat := c.param.PeerAddressType()

	smp.InitContext(la, ra, lat, rat)
}

func (c *Conn) encrypt(bi hci.BondInfo) error {
	legacy, stk := c.smp.LegacyPairingInfo()
	//if a short term key is present, use it as the long term key
	if legacy && len(stk) > 0 {
		fmt.Println("encrypting with short term key")
		return c.stkEncrypt(stk)
	}

	if bi == nil {
		return fmt.Errorf("no bond information")
	}

	ltk := bi.LongTermKey()
	if ltk == nil {
		return fmt.Errorf("no ltk present")
	}

	m := cmd.LEStartEncryption{}
	m.ConnectionHandle = c.param.ConnectionHandle()

	eDiv := bi.EDiv()
	randVal := bi.Random()

	if bi.Legacy() {
		//expect LTK, EDiv, and Rand to be present
		if len(ltk) != 16 {
			return fmt.Errorf("invalid length for ltk")
		}

		if eDiv == 0 || randVal == 0 {
			return fmt.Errorf("ediv and random must not be 0 for legacy pairing")
		}
	}

	for i, v := range ltk {
		m.LongTermKey[i] = v
	}

	m.EncryptedDiversifier = eDiv
	m.RandomNumber = randVal

	return c.ctrl.Send(&m, nil)
}

func (c *Conn) stkEncrypt(key []byte) error {
	m := cmd.LEStartEncryption{}
	m.ConnectionHandle = c.param.ConnectionHandle()
	for i, v := range key {
		m.LongTermKey[i] = v
	}

	m.EncryptedDiversifier = 0
	m.RandomNumber = 0

	return c.ctrl.Send(&m, nil)
}

// writePDU breaks down a L2CAP PDU into fragments if it's larger than the HCI buffer size. [Vol 3, Part A, 7.2.1]
func (c *Conn) writePDU(pdu []byte) (int, error) {
	sent := 0
	flags := uint16(hci.PbfHostToControllerStart << 4) // ACL boundary flags

	// All L2CAP fragments associated with an L2CAP PDU shall be processed for
	// transmission by the Controller before any other L2CAP PDU for the same
	// logical transport shall be processed.
	c.txBuffer.Lock()
	defer c.txBuffer.Unlock()

	// Fail immediately if the connection is already closed
	// Check this with the pool locked to avoid race conditions
	// with handleDisconnectionComplete
	select {
	case <-c.chDone:
		return 0, io.ErrClosedPipe
	default:
	}

	for len(pdu) > 0 {
		// Get a buffer from our pre-allocated and flow-controlled pool.
		pkt := c.txBuffer.Get() // ACL pkt
		fragmentLen := len(pdu)
		if fragmentLen > pkt.Cap()-1-4 {
			fragmentLen = pkt.Cap() - 1 - 4
		}

		// prepare the packet
		err := buildPacket(pdu, pkt, c.param.ConnectionHandle(), flags, fragmentLen)
		if err != nil {
			return 0, err
		}

		// Flush the pkt to HCI
		// eps: I wonder if this should be an error situation or not
		select {
		case <-c.chDone:
			return 0, io.ErrClosedPipe
		default:
		}

		if _, err := c.ctrl.SocketWrite(pkt.Bytes()); err != nil {
			return sent, errors.Wrap(err, "connection.writePdu")
		}
		sent += fragmentLen

		flags = hci.PbfContinuing << 4 // Set "continuing" in the boundary flags for the rest of fragments, if any.
		pdu = pdu[fragmentLen:] // update slice with unsetn data
	}

	return sent, nil
}

func buildPacket(pdu Pdu, pkt *bytes.Buffer, handle uint16, flags uint16, fragmentLen int) error {
	// HCI Header: pkt Type
	if err := binary.Write(pkt, binary.LittleEndian, hci.PktTypeACLData); err != nil {
		return errors.Wrap(err, "buildPacket")
	}
	// ACL Header: handle and flags
	if err := binary.Write(pkt, binary.LittleEndian, handle|(flags<<8)); err != nil {
		return errors.Wrap(err, "buildPacket")
	}
	// ACL Header: data len
	if err := binary.Write(pkt, binary.LittleEndian, uint16(fragmentLen)); err != nil {
		return errors.Wrap(err, "buildPacket")
	}
	// Append payload
	if err := binary.Write(pkt, binary.LittleEndian, pdu[:fragmentLen]); err != nil {
		return errors.Wrap(err, "buildPacket")
	}

	return nil
}

// Recombines fragments into a L2CAP PDU. [Vol 3, Part A, 7.2.2]
func (c *Conn) recombine(pkt Packet) error {
	p := Pdu(pkt.data())

	// Currently, check for LE-U only. For channels that we don't recognizes,
	// re-combine them anyway, and discard them later when we dispatch the PDU
	// according to CID.
	if p.cid() == cidLEAtt && p.dlen() > c.rxMPS {
		return fmt.Errorf("fragment size (%d) larger than rxMPS (%d)", p.dlen(), c.rxMPS)
	}

	// If this pkt is not a complete PDU, and we'll be receiving more
	// fragments, re-allocate the whole PDU (including Header).
	if len(p.payload()) < p.dlen() {
		p = make([]byte, 0, 4+p.dlen())
		p = append(p, Pdu(pkt.data())...)
	}

	for len(p) < 4+p.dlen() {
		var more Packet
		var ok bool
		if more, ok = <-c.chInPkt; !ok || (more.Pbf()&hci.PbfContinuing) == 0 {
			return io.ErrUnexpectedEOF
		}

		p = append(p, Pdu(more.data())...)
	}

	// TODO: support dynamic or assigned channels for LE-Frames.
	switch p.cid() {
	case cidLEAtt:
		c.chInPDU <- p
	case cidLESignal:
		_ = c.handleSignal(p)
	case CidSMP:
		_ = c.smp.Handle(p)
	default:
		_ = hci.Logger.Error("recombine()", "unrecognized CID", fmt.Sprintf("%04X, [%X]", p.cid(), p))
	}
	return nil
}
