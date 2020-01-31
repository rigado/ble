package controller

import (
	"fmt"
	"github.com/rigado/ble"
	"github.com/rigado/ble/linux/hci"
	)

func (h *HCI) RequestBufferPool() hci.BufferPool {
	return hci.NewClient(h.pool)
}

func (h *HCI) RequestSmpManager(cfg hci.SmpConfig) (hci.SmpManager, error) {
	if !h.smpEnabled {
		return nil, fmt.Errorf("smp not available")
	}

	//todo: this sholud validate smp config
	m := h.smp.Create(cfg)
	return m, nil
}

func (h *HCI) DispatchError(e error) {
	h.dispatchError(e)
}

func (h *HCI) SocketWrite(b []byte) (int, error) {
	return h.skt.Write(b)
}

func (h *HCI) Addr() ble.Addr {
	return ble.NewAddr(h.addr.String())
}