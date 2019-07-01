package mgmt

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// Socket implements a HCI User Channel as ReadWriteCloser.
type Socket struct {
	fd     int
	buf    []byte
	closed chan struct{}
	rmu    sync.Mutex
	wmu    sync.Mutex
}

type Response struct {
	ID     uint16
	Index  uint16
	Length uint16
	Data   []byte
}

func cmd(id, index uint16, length uint16) []byte {
	b := []byte{
		byte(id & 0xff), byte((id & 0xff00) >> 8),
		byte((index & 0xff)), byte((index & 0xff00) >> 8),
		byte(length & 0xff), byte((length & 0xff00) >> 8),
	}

	return b
}

// NewSocket returns a HCI User Channel of specified device id.
// If id is -1, the first available HCI device is returned.
func NewSocket(id int) (*Socket, error) {
	var err error
	// Create RAW HCI Socket.
	fd, err := unix.Socket(unix.AF_BLUETOOTH, (unix.SOCK_RAW), unix.BTPROTO_HCI)
	if err != nil {
		return nil, errors.Wrap(err, "can't create socket")
	}

	hciSock := unix.SockaddrHCI{
		Dev:     uint16(0xffff), //must be hci dev none 0xffff?
		Channel: unix.HCI_CHANNEL_CONTROL,
	}

	if err := unix.Bind(fd, &hciSock); err != nil {
		return nil, err
	}

	// poll for 20ms to see if any data becomes available, then clear it
	pfds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	unix.Poll(pfds, 20)
	if pfds[0].Revents&unix.POLLIN > 0 {
		b := make([]byte, 512)
		unix.Read(fd, b)
	}

	s := &Socket{fd: fd, buf: make([]byte, 4096), closed: make(chan struct{})}
	log.Println("get indexes")
	s.WriteCmd(3, 0xffff, nil)
	rsp, err := s.ReadRsp()
	log.Println(rsp, err)

	<-time.After(time.Second)
	log.Println("power on idx 1")
	s.WriteCmd(5, uint16(1), []byte{1})
	rsp, err = s.ReadRsp()
	log.Println(rsp, err)

	<-time.After(time.Second)

	log.Println("get indexes")
	s.WriteCmd(3, 0xffff, nil)
	rsp, err = s.ReadRsp()
	log.Println(rsp, err)

	return s, nil
}

func (s *Socket) WriteCmd(id, ctrl uint16, b []byte) error {
	//todo: mutex protect?
	c := append(cmd(id, ctrl, uint16(len(b))), b...)
	log.Println("mgmt <", hex.EncodeToString(c))
	n, err := unix.Write(s.fd, c)
	if err != nil {
		return err
	}

	if n == 0 {
		return fmt.Errorf("wrote 0 bytes")
	}

	return nil
}

func (s *Socket) ReadRsp() (Response, error) {
	n, err := unix.Read(s.fd, s.buf)
	if err != nil {
		return Response{}, err
	}

	log.Println("mgmt >", hex.EncodeToString(s.buf[:n]))
	r := Response{
		binary.LittleEndian.Uint16(s.buf[0:2]),
		binary.LittleEndian.Uint16(s.buf[2:4]),
		binary.LittleEndian.Uint16(s.buf[4:6]),
		s.buf[6:n],
	}

	return r, nil
}
