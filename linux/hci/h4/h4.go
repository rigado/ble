// +build linux

package h4

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/chmorgan/go-serial2/serial"
	"github.com/pkg/errors"
)

const (
	rxQueueSize = 64
	txQueueSize = 64
)

type h4 struct {
	sp  io.ReadWriteCloser
	rmu sync.Mutex
	wmu sync.Mutex

	rxQueue chan []byte
	txQueue chan []byte

	done chan int
	cmu  sync.Mutex
}

func New(opts serial.OpenOptions) (io.ReadWriteCloser, error) {
	// force these
	opts.MinimumReadSize = 0
	opts.InterCharacterTimeout = 100

	log.Println("opening...")
	sp, err := serial.Open(opts)
	if err != nil {
		return nil, err
	}

	// dump data
	// todo this is mega slow and stupid, but I doubt we can change this on the fly
	log.Println("flushing...")
	b := make([]byte, 2048)
	//reset to wake up?
	sp.Write([]byte{1, 3, 12, 0})
	_, err = sp.Read(b)
	if err != nil {
		sp.Close()
		return nil, err
	}

	log.Println("opened", opts, err)

	h := &h4{
		sp:      sp,
		done:    make(chan int),
		rxQueue: make(chan []byte, rxQueueSize),
		txQueue: make(chan []byte, txQueueSize),
	}

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
		return 0, fmt.Errorf("timeout")
	}

	log.Println("read", p[:n], n, err)

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
	n, err := h.sp.Write(p)
	log.Println("write", p, n, err)

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
		err := h.sp.Close()
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
		// log.Printf("isOpen: %v\n", h.sp != nil)
		return h.sp != nil
	}
}

func (h *h4) rxLoop() {
	msg := make([]byte, 0, 256)
	tmp := make([]byte, 256)
	fto := time.Now().Add(time.Millisecond * 100)
	for {
		select {
		case <-h.done:
			log.Printf("rxLoop killed")
			return
		default:
			if h.sp == nil {
				log.Printf("rxLoop nil sp")
				return
			}
		}

		// read
		n, err := h.sp.Read(tmp)
		// log.Printf("sp read %v %v", n, err)
		if err == nil && n != 0 {
			log.Printf("rx %v", tmp[:n])
			msg = append(msg, tmp[:n]...)
		}

		//flush
		if time.Now().After(fto) {
			if len(msg) != 0 {
				log.Printf("sp flush %v", msg)
				h.rxQueue <- msg
			} else {
				log.Printf("sp flush: nil")
			}

			fto = time.Now().Add(time.Millisecond * 200)
			msg = make([]byte, 0, 256)
		}
	}
}
