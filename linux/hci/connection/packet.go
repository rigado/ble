package connection

import "encoding/binary"

// pkt implements HCI ACL Data Packet [Vol 2, Part E, 5.4.2]
// Packet boundary flags , bit[5:6] of handle field's MSB
// Broadcast flags. bit[7:8] of handle field's MSB
// Not used in LE-U. Leave it as 0x00 (Point-to-Point).
// Broadcasting in LE uses ADVB logical transport.
type Packet []byte

func (a Packet) handle() uint16 { return uint16(a[0]) | (uint16(a[1]&0x0f) << 8) }
func (a Packet) Pbf() int       { return (int(a[1]) >> 4) & 0x3 }
func (a Packet) bcf() int       { return (int(a[1]) >> 6) & 0x3 }
func (a Packet) dlen() int      { return int(a[2]) | (int(a[3]) << 8) }
func (a Packet) data() []byte   { return a[4:] }

type Pdu []byte

func (p Pdu) dlen() int       { return int(binary.LittleEndian.Uint16(p[0:2])) }
func (p Pdu) cid() uint16     { return binary.LittleEndian.Uint16(p[2:4]) }
func (p Pdu) payload() []byte { return p[4:] }

type leFrameHdr Pdu

func (f leFrameHdr) slen() int       { return int(binary.LittleEndian.Uint16(f[4:6])) }
func (f leFrameHdr) payload() []byte { return f[6:] }
