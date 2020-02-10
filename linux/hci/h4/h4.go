package h4

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/jacobsa/go-serial/serial"
	"github.com/pkg/errors"
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

	log.Println("opening...")
	rwc, err := serial.Open(opts)
	if err != nil {
		return nil, err
	}

	// dump data
	// todo this is mega slow and stupid, but I doubt we can change this on the fly
	log.Println("flushing...")
	b := make([]byte, 2048)
	rwc.Write([]byte{1, 3, 12, 0}) //dummy reset
	<-time.After(time.Millisecond * 250)
	_, err = rwc.Read(b)
	if err != nil {
		rwc.Close()
		return nil, err
	}

	log.Println("opened", opts, err)

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
	log.Printf("Dialing %v ...", addr)
	c, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}

	rwc := &connWithTimeout{c, connTimeout}
	log.Println("flushing...")
	b := make([]byte, 2048)
	rwc.Write([]byte{1, 3, 12, 0}) //dummy reset
	for {
		n, err := rwc.Read(b)
		// log.Println(n, err)
		if n == 0 || err != nil {
			break
		}
	}

	log.Println("connect", c.RemoteAddr().String(), err)

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
		return 0, nil //fmt.Errorf("timeout")
	}

	// log.Printf("read [% 0x], %v, %v", p[:n], n, err)

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
		fmt.Println("h4 already closed!")
		return nil

	default:
		close(h.done)
		fmt.Println("closing h4")
		h.rmu.Lock()
		err := h.rwc.Close()
		h.rmu.Unlock()

		return errors.Wrap(err, "can't close h4")
	}
}

func (h *h4) isOpen() bool {
	select {
	case <-h.done:
		log.Printf("isOpen: <-h.done, false\n")
		return false
	default:
		// log.Printf("isOpen: %v\n", h.rwc != nil)
		return h.rwc != nil
	}
}

func (h *h4) rxLoop() {
	tmp := make([]byte, 512)
	for {
		select {
		case <-h.done:
			log.Printf("rxLoop killed")
			return
		default:
			if h.rwc == nil {
				log.Printf("rxLoop nil rwc")
				return
			}
		}

		// read
		n, err := h.rwc.Read(tmp)
		if err != nil || n == 0 {
			continue
		}

		// put
		h.frame.Assemble(tmp[:n])
	}
}
