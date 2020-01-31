package connection

// L2CAP Channel Identifier namespace for LE-U logical link [Vol 3, Part A, 2.1].
const (
	cidLEAtt    uint16 = 0x04 // Attribute Protocol [Vol 3, Part F].
	cidLESignal uint16 = 0x05 // Low Energy L2CAP Signaling channel [Vol 3, Part A, 4].
	CidSMP      uint16 = 0x06 // SecurityManager Protocol [Vol 3, Part H].
)
