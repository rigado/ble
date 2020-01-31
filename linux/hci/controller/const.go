package controller

import "time"



const (
	chCmdBufChanSize    = 16 // TODO: decide correct size (comment migrated)
	chCmdBufElementSize = 64
	chCmdBufTimeout     = time.Second * 5
)
