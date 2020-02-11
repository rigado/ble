package controller

import (
	"context"
	"fmt"
	"github.com/rigado/ble/linux/hci"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rigado/ble"
	"github.com/rigado/ble/linux/hci/cmd"
	"github.com/rigado/ble/linux/hci/evt"
	"github.com/rigado/ble/linux/hci/socket"
)

type handlerFn func(b []byte) error

type pkt struct {
	cmd  hci.Command
	done chan []byte
}

// NewHCI returns a hci device.
func NewHCI(smp hci.SmpManagerFactory, cf hci.ConnectionFactory, opts ...ble.Option) (*HCI, error) {
	h := &HCI{
		id:        -1,
		smp:       smp,
		cf:        cf,
		chCmdPkt:  make(chan *pkt),
		chCmdBufs: make(chan []byte, chCmdBufChanSize),
		sent:      make(map[int]*pkt),
		muSent:    sync.Mutex{},

		evth: map[int]handlerFn{},
		subh: map[int]handlerFn{},

		muConns:      sync.Mutex{},
		conns:        make(map[uint16]hci.Connection),
		chMasterConn: make(chan hci.Connection),
		chSlaveConn:  make(chan hci.Connection),

		muClose:   sync.Mutex{},
		done:      make(chan bool),
		sktRxChan: make(chan []byte, 16), //todo pick a real number

	}

	h.ctx, h.cancel = context.WithCancel(context.Background())

	h.params.init()
	if err := h.Option(opts...); err != nil {
		return nil, errors.Wrap(err, "can't set options")
	}

	return h, nil
}

// HCI ...
type HCI struct {
	sync.Mutex

	params params

	smp        hci.SmpManagerFactory
	smpEnabled bool

	cf         hci.ConnectionFactory

	skt io.ReadWriteCloser
	id  int

	// Host to Controller command flow control [Vol 2, Part E, 4.4]
	chCmdPkt  chan *pkt
	chCmdBufs chan []byte
	muSent    sync.Mutex
	sent      map[int]*pkt

	// evtHub
	evth map[int]handlerFn
	subh map[int]handlerFn

	// aclHandler
	bufSize int
	bufCnt  int

	// Device information or status.
	addr    net.HardwareAddr
	txPwrLv int

	// adHist and adLast track the history of past scannable advertising packets.
	// Controller delivers AD(Advertising Data) and SR(Scan Response) separately
	// through HCI. Upon receiving an AD, no matter it's scannable or not, we
	// pass a Advertisement (AD only) to advHandler immediately.
	// Upon receiving a SR, we search the AD history for the AD from the same
	// device, and pass the Advertisiement (AD+SR) to advHandler.
	// The adHist and adLast are allocated in the Scan().
	advHandlerSync bool
	advHandler     ble.AdvHandler
	adHist         []*hci.Advertisement
	adLast         int

	// Host to Controller Data Flow Control Packet-based Data flow control for LE-U [Vol 2, Part E, 4.1.1]
	// Minimum 27 bytes. 4 bytes of L2CAP Header, and 23 bytes Payload from upper layer (ATT)
	pool *hci.Pool

	// L2CAP connections
	muConns      sync.Mutex
	conns        map[uint16]hci.Connection
	chMasterConn chan hci.Connection // Dial returns master connections.
	chSlaveConn  chan hci.Connection // Peripheral accept slave connections.

	dialerTmo   time.Duration
	listenerTmo time.Duration

	//error handler
	errorHandler func(error)
	err          error

	muClose sync.Mutex
	done    chan bool

	sktRxChan chan []byte

	ctx context.Context
	cancel context.CancelFunc
}

// Init ...
func (h *HCI) Init() error {
	h.evth[0x3E] = h.handleLEMeta
	h.evth[evt.CommandCompleteCode] = h.handleCommandComplete
	h.evth[evt.CommandStatusCode] = h.handleCommandStatus
	h.evth[evt.DisconnectionCompleteCode] = h.handleDisconnectionComplete
	h.evth[evt.NumberOfCompletedPacketsCode] = h.handleNumberOfCompletedPackets
	h.evth[evt.EncryptionChangeCode] = h.handleEncryptionChange

	h.subh[evt.LEAdvertisingReportSubCode] = h.handleLEAdvertisingReport
	h.subh[evt.LEConnectionCompleteSubCode] = h.handleLEConnectionComplete
	h.subh[evt.LEConnectionUpdateCompleteSubCode] = h.handleLEConnectionUpdateComplete
	h.subh[evt.LELongTermKeyRequestSubCode] = h.handleLELongTermKeyRequest
	h.subh[evt.EncryptionChangeCode] = h.handleEncryptionChange
	// evt.ReadRemoteVersionInformationCompleteCode: todo),
	// evt.HardwareErrorCode:                        todo),
	// evt.DataBufferOverflowCode:                   todo),
	// evt.EncryptionKeyRefreshCompleteCode:         todo),
	// evt.AuthenticatedPayloadTimeoutExpiredCode:   todo),
	// evt.LEReadRemoteUsedFeaturesCompleteSubCode:   todo),
	// evt.LERemoteConnectionParameterRequestSubCode: todo),

	var err error
	h.skt, err = socket.NewSocket(h.id)
	if err != nil {
		return err
	}

	h.setAllowedCommands(1)

	go h.sktReadLoop()
	go h.sktProcessLoop()
	if err := h.init(); err != nil {
		return err
	}

	// Pre-allocate buffers with additional head room for lower layer headers.
	// HCI header (1 Byte) + ACL Data Header (4 bytes) + L2CAP PDU (or fragment)
	h.pool = hci.NewPool(1+4+h.bufSize, h.bufCnt-1)

	h.Send(&h.params.advParams, nil)
	h.Send(&h.params.scanParams, nil)
	return nil
}

func (h *HCI) cleanup() {
	//close the socket
	h.close(nil)

	//this effectively kills any dials in flight
	close(h.chMasterConn)
	h.chMasterConn = nil

	// kill all open connections w/o disconnect
	for ch := range h.conns {
		fmt.Println("hci cleanup, closing all connections")
		h.cleanupConnectionHandle(ch)
	}

	// clean out all sent commands (prob unneeded)
	h.muSent.Lock()
	for k := range h.sent {
		delete(h.sent, k)
	}
	h.muSent.Unlock()
}


// Close ...
func (h *HCI) Close() error {
	h.muClose.Lock()
	defer h.muClose.Unlock()

	select {
	case <-h.done:
		//already closed, nothing to do
	default:
		close(h.done)
	}

	return nil
}

// Error ...
func (h *HCI) Error() error {
	return h.err
}

// Option sets the options specified.
func (h *HCI) Option(opts ...ble.Option) error {
	var err error
	for _, opt := range opts {
		err = opt(h)
	}
	return err
}

func (h *HCI) isOpen() bool {
	select {
	case <-h.done:
		return false
	default:
		return true
	}
}

func (h *HCI) init() error {
	hci.Logger.Info("hci reset")
	h.Send(&cmd.Reset{}, nil)

	ReadBDADDRRP := cmd.ReadBDADDRRP{}
	h.Send(&cmd.ReadBDADDR{}, &ReadBDADDRRP)

	a := ReadBDADDRRP.BDADDR
	h.addr = net.HardwareAddr([]byte{a[5], a[4], a[3], a[2], a[1], a[0]})

	//ES note: Per Core Spec 5.0, Part E, 7.4.5
	//This command is _not_ to be supported by LE only controllers
	ReadBufferSizeRP := cmd.ReadBufferSizeRP{}
	h.Send(&cmd.ReadBufferSize{}, &ReadBufferSizeRP)

	// Assume the buffers are shared between ACL-U and LE-U.
	h.bufCnt = int(ReadBufferSizeRP.HCTotalNumACLDataPackets)
	h.bufSize = int(ReadBufferSizeRP.HCACLDataPacketLength)

	LEReadBufferSizeRP := cmd.LEReadBufferSizeRP{}
	h.Send(&cmd.LEReadBufferSize{}, &LEReadBufferSizeRP)

	if LEReadBufferSizeRP.HCTotalNumLEDataPackets != 0 {
		// Okay, LE-U do have their own buffers.
		h.bufCnt = int(LEReadBufferSizeRP.HCTotalNumLEDataPackets)
		h.bufSize = int(LEReadBufferSizeRP.HCLEDataPacketLength)
	}

	LEReadAdvertisingChannelTxPowerRP := cmd.LEReadAdvertisingChannelTxPowerRP{}
	h.Send(&cmd.LEReadAdvertisingChannelTxPower{}, &LEReadAdvertisingChannelTxPowerRP)

	h.txPwrLv = int(LEReadAdvertisingChannelTxPowerRP.TransmitPowerLevel)

	LESetEventMaskRP := cmd.LESetEventMaskRP{}
	h.Send(&cmd.LESetEventMask{LEEventMask: 0x000000000000001F}, &LESetEventMaskRP)

	SetEventMaskRP := cmd.SetEventMaskRP{}
	h.Send(&cmd.SetEventMask{EventMask: 0x3dbff807fffbffff}, &SetEventMaskRP)

	WriteLEHostSupportRP := cmd.WriteLEHostSupportRP{}
	h.Send(&cmd.WriteLEHostSupport{LESupportedHost: 1, SimultaneousLEHost: 0}, &WriteLEHostSupportRP)

	return h.err
}

func (h *HCI) makeAdvError(e error, b []byte, dispatch bool) error {
	err := fmt.Errorf("%v, bytes %v", e, b)
	if dispatch {
		h.dispatchError(err)
	}
	return err
}

func (h *HCI) setAllowedCommands(n int) {
	if n > chCmdBufChanSize {
		fmt.Printf("hci.setAllowedCommands: warning, defaulting %d -> %d\n", n, chCmdBufChanSize)
		n = chCmdBufChanSize
	}

	//put with timeout
	for len(h.chCmdBufs) < n {
		select {
		case <-h.done:
			//closed
			return
		case h.chCmdBufs <- make([]byte, chCmdBufElementSize):
			//ok
		case <-time.After(chCmdBufTimeout):
			h.dispatchError(fmt.Errorf("chCmdBufs put timeout"))
			//timeout
			break
		}
	}
}

func (h *HCI) dispatchError(e error) {
	switch {
	case h.errorHandler == nil:
		fmt.Println(e)
	case !h.isOpen():
		//don't dispatch
		fmt.Println("hci closing:", e)
	default:
		h.errorHandler(e)
	}
}
