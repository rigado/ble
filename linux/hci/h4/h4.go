package h4

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/donaldschen/go-serial2/serial"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	rxQueueSize    = 64
	txQueueSize    = 64
	defaultTimeout = time.Second * 1
)

type h4 struct {
	rwc io.ReadWriteCloser
	rmu sync.Mutex
	wmu sync.Mutex

	frame *frame

	rxQueue chan []byte
	txQueue chan []byte

	done chan int
	cmu  sync.Mutex
}

func DefaultSerialOptions() serial.OpenOptions {
	return serial.OpenOptions{
		PortName:              "/dev/ttyACM0",
		BaudRate:              1000000,
		DataBits:              8,
		ParityMode:            serial.PARITY_NONE,
		StopBits:              1,
		RTSCTSFlowControl:     true,
		MinimumReadSize:       0,
		InterCharacterTimeout: 100,
	}
}

func NewSerial(opts serial.OpenOptions) (io.ReadWriteCloser, error) {
	// force these
	opts.MinimumReadSize = 0
	opts.InterCharacterTimeout = 100

	logrus.Infoln("opening...")
	rwc, err := serial.Open(opts)
	if err != nil {
		return nil, err
	}

	// dump data
	// todo this is mega slow and stupid, but I doubt we can change this on the fly
	logrus.Infoln("flushing...")
	b := make([]byte, 2048)
	rwc.Write([]byte{1, 3, 12, 0}) //dummy reset
	<-time.After(time.Millisecond * 250)
	_, err = rwc.Read(b)
	if err != nil {
		rwc.Close()
		return nil, err
	}

	logrus.Infof("opened %v, err: %v", opts, err)

	h := &h4{
		rwc:     rwc,
		done:    make(chan int),
		rxQueue: make(chan []byte, rxQueueSize),
		txQueue: make(chan []byte, txQueueSize),
	}
	h.frame = newFrame(h.rxQueue)

	go h.rxLoop()

	return h, nil
}

func NewSocket(addr string, connTimeout time.Duration) (io.ReadWriteCloser, error) {
	logrus.Infof("Dialing %v ...", addr)
	c, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}

	// use a shorter timeout when flushing so we dont block for too long in init
	fast := time.Millisecond * 500
	rwc := &connWithTimeout{c, fast}
	logrus.Infoln("flushing...")
	b := make([]byte, 2048)
	rwc.Write([]byte{1, 3, 12, 0}) //dummy reset
	for {
		n, err := rwc.Read(b)
		if n == 0 || err != nil {
			break
		}
	}

	// set the real timeout
	rwc.timeout = connTimeout
	logrus.Debugf("connect %v, err: %v", c.RemoteAddr().String(), err)

	h := &h4{
		rwc:     rwc,
		done:    make(chan int),
		rxQueue: make(chan []byte, rxQueueSize),
		txQueue: make(chan []byte, txQueueSize),
	}
	h.frame = newFrame(h.rxQueue)

	go h.rxLoop()

	return h, nil
}

func (h *h4) Read(p []byte) (int, error) {
	if !h.isOpen() {
		return 0, io.EOF
	}

	h.rmu.Lock()
	defer h.rmu.Unlock()

	var n int
	var err error
	select {
	case t := <-h.rxQueue:
		//ok
		if len(p) < len(t) {
			return 0, fmt.Errorf("buffer too small")
		}
		n = copy(p, t)

	case <-time.After(time.Second):
		return 0, nil
	}

	// check if we are still open since the read could take a while
	if !h.isOpen() {
		return 0, io.EOF
	}
	return n, errors.Wrap(err, "can't read h4")
}

func (h *h4) Write(p []byte) (int, error) {
	if !h.isOpen() {
		return 0, io.EOF
	}

	h.wmu.Lock()
	defer h.wmu.Unlock()
	n, err := h.rwc.Write(p)
	// log.Printf("write [% 0x], %v, %v", p, n, err)

	return n, errors.Wrap(err, "can't write h4")
}

func (h *h4) Close() error {
	h.cmu.Lock()
	defer h.cmu.Unlock()

	select {
	case <-h.done:
		logrus.Infoln("h4 already closed!")
		return nil

	default:
		close(h.done)
		logrus.Infoln("closing h4")
		h.rmu.Lock()
		err := h.rwc.Close()
		h.rmu.Unlock()

		return errors.Wrap(err, "can't close h4")
	}
}

func (h *h4) isOpen() bool {
	select {
	case <-h.done:
		logrus.Infoln("isOpen: <-h.done, false")
		return false
	default:
		return h.rwc != nil
	}
}

func (h *h4) rxLoop() {
	tmp := make([]byte, 512)
	for {
		select {
		case <-h.done:
			logrus.Infoln("rxLoop killed")
			return
		default:
			if h.rwc == nil {
				logrus.Infoln("rxLoop nil rwc")
				return
			}
		}

		// read
		n, err := h.rwc.Read(tmp)
		switch {
		case err == io.EOF:
			logrus.Error(err)
			h.Close()
			break
		case err == nil && n > 0:
			//process
			h.frame.Assemble(tmp[:n])
		default:
			//nothing to do
			continue
		}
	}
}
