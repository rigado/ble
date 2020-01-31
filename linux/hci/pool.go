package hci

import "bytes"

type BufferPool interface {
	Lock()
	Unlock()
	Get() *bytes.Buffer
	Put()
	PutAll()
}
