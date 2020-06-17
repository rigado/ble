// +build linux

package socket

import (
	"fmt"
	"io"
	"sync"
	"time"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

func ioR(t, nr, size uintptr) uintptr {
	return (2 << 30) | (t << 8) | nr | (size << 16)
}

func ioW(t, nr, size uintptr) uintptr {
	return (1 << 30) | (t << 8) | nr | (size << 16)
}

func ioctl(fd, op, arg uintptr) error {
	if _, _, ep := unix.Syscall(unix.SYS_IOCTL, fd, op, arg); ep != 0 {
		return ep
	}
	return nil
}

const (
	ioctlSize      = 4
	hciMaxDevices  = 16
	typHCI         = 72 // 'H'
	readTimeout    = 1000
	unixPollErrors = int16(unix.POLLHUP | unix.POLLNVAL | unix.POLLERR)
	unixPollDataIn = int16(unix.POLLIN)
)

var (
	hciUpDevice      = ioW(typHCI, 201, ioctlSize) // HCIDEVUP
	hciDownDevice    = ioW(typHCI, 202, ioctlSize) // HCIDEVDOWN
	hciResetDevice   = ioW(typHCI, 203, ioctlSize) // HCIDEVRESET
	hciGetDeviceList = ioR(typHCI, 210, ioctlSize) // HCIGETDEVLIST
	hciGetDeviceInfo = ioR(typHCI, 211, ioctlSize) // HCIGETDEVINFO
)

type devListRequest struct {
	devNum     uint16
	devRequest [hciMaxDevices]struct {
		id  uint16
		opt uint32
	}
}

// Socket implements a HCI User Channel as ReadWriteCloser.
type Socket struct {
	fd   int
	rmu  sync.Mutex
	wmu  sync.Mutex
	done chan int
	cmu  sync.Mutex
}

// NewSocket returns a HCI User Channel of specified device id.
// If id is -1, the first available HCI device is returned.
func NewSocket(id int) (*Socket, error) {
	var err error
	// Create RAW HCI Socket.
	fd, err := unix.Socket(unix.AF_BLUETOOTH, unix.SOCK_RAW, unix.BTPROTO_HCI)
	if err != nil {
		return nil, errors.Wrap(err, "can't create socket")
	}

	if id != -1 {
		to := time.Now().Add(time.Second * 60)
		var err error
		var s *Socket
		for time.Now().Before(to) {
			s, err = open(fd, id)
			if err == nil {
				return s, nil
			}
			unix.Close(fd)
			<-time.After(time.Second)
		}

		return nil, err
	}

	req := devListRequest{devNum: hciMaxDevices}
	if err = ioctl(uintptr(fd), hciGetDeviceList, uintptr(unsafe.Pointer(&req))); err != nil {
		unix.Close(fd)
		return nil, errors.Wrap(err, "can't get device list")
	}
	var msg string
	for id := 0; id < int(req.devNum); id++ {
		s, err := open(fd, id)
		if err == nil {
			return s, nil
		}
		msg = msg + fmt.Sprintf("(hci%d: %s)", id, err)
	}
	unix.Close(fd)
	return nil, errors.Errorf("no devices available: %s", msg)
}

func open(fd, id int) (*Socket, error) {

	// HCI User Channel requires exclusive access to the device.
	// The device has to be down at the time of binding.
	if err := ioctl(uintptr(fd), hciDownDevice, uintptr(id)); err != nil {
		return nil, errors.Wrap(err, "can't down device")
	}

	// Bind the RAW socket to HCI User Channel
	sa := unix.SockaddrHCI{Dev: uint16(id), Channel: unix.HCI_CHANNEL_USER}
	if err := unix.Bind(fd, &sa); err != nil {
		return nil, errors.Wrap(err, "can't bind socket to hci user channel")
	}

	// poll for 20ms to see if any data becomes available, then clear it
	pfds := []unix.PollFd{{Fd: int32(fd), Events: unixPollDataIn}}
	unix.Poll(pfds, 20)
	evts := pfds[0].Revents

	switch {
	case evts&unixPollErrors != 0:
		return nil, io.EOF

	case evts&unixPollDataIn != 0:
		b := make([]byte, 2048)
		unix.Read(fd, b)
	}

	return &Socket{fd: fd, done: make(chan int)}, nil
}

func (s *Socket) Read(p []byte) (int, error) {
	if !s.isOpen() {
		return 0, io.EOF
	}

	var err error
	n := 0
	s.rmu.Lock()
	defer s.rmu.Unlock()
	// dont need to add unixPollErrors, they are always returned
	pfds := []unix.PollFd{{Fd: int32(s.fd), Events: unixPollDataIn}}
	unix.Poll(pfds, readTimeout)
	evts := pfds[0].Revents

	switch {
	case evts&unixPollErrors != 0:
		fmt.Printf("hci socket error: poll events 0x%04x\n", evts)
		return 0, io.EOF

	case evts&unixPollDataIn != 0:
		// there is data!
		n, err = unix.Read(s.fd, p)

	default:
		// no data, read timeout
		return 0, nil
	}

	// check if we are still open since the read takes a while
	if !s.isOpen() {
		return 0, io.EOF
	}
	return n, errors.Wrap(err, "can't read hci socket")
}

func (s *Socket) Write(p []byte) (int, error) {
	if !s.isOpen() {
		return 0, io.EOF
	}

	s.wmu.Lock()
	defer s.wmu.Unlock()
	n, err := unix.Write(s.fd, p)
	return n, errors.Wrap(err, "can't write hci socket")
}

func (s *Socket) Close() error {
	s.cmu.Lock()
	defer s.cmu.Unlock()

	select {
	case <-s.done:
		return nil

	default:
		close(s.done)
		fmt.Println("closing hci socket!")
		s.rmu.Lock()
		err := unix.Close(s.fd)
		s.rmu.Unlock()

		return errors.Wrap(err, "can't close hci socket")
	}
}

func (s *Socket) isOpen() bool {
	select {
	case <-s.done:
		return false
	default:
		return true
	}
}
