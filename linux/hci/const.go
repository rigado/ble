package hci

import "time"

// HCI Packet types
const (
	PktTypeCommand uint8 = 0x01
	PktTypeACLData uint8 = 0x02
	PktTypeSCOData uint8 = 0x03
	PktTypeEvent   uint8 = 0x04
	PktTypeVendor  uint8 = 0xFF
)

// Packet boundary flags of HCI ACL Data Packet [Vol 2, Part E, 5.4.2].
const (
	PbfHostToControllerStart = 0x00 // Start of a non-automatically-flushable from host to controller.
	PbfContinuing            = 0x01 // Continuing fragment.
	pbfControllerToHostStart = 0x02 // Start of a non-automatically-flushable from controller to host.
	pbfCompleteL2CAPPDU      = 0x03 // A automatically flushable complete PDU. (Not used in LE-U).
)



const (
	chCmdBufChanSize    = 16 // TODO: decide correct size (comment migrated)
	chCmdBufElementSize = 64
	chCmdBufTimeout     = time.Second * 5
)

const (
	RoleMaster = 0x00
	RoleSlave  = 0x01
)
