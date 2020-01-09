// +build !linux

package h4

import (
	"fmt"
	"io"
)

// NewSocket is a dummy function for non-Linux platform.
func NewH4() (io.ReadWriteCloser, error) {
	return nil, fmt.Errorf("only available on linux")
}
