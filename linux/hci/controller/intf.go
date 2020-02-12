package controller

import (
	"context"
	"fmt"
	"github.com/rigado/ble"
	"github.com/rigado/ble/linux/hci"
	"github.com/rigado/ble/linux/hci/cmd"
	"time"
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

func (h *HCI) CloseConnection(handle uint16) error {
	h.muConns.Lock()
	defer h.muConns.Unlock()
	c, found := h.conns[handle]
	if !found {
		return fmt.Errorf("disconnecting an invalid handle %04X", handle)
	}

	go func(c hci.Connection) {
		select {
		case <-c.Disconnected():
		case <-time.After(10 * time.Second):
			fmt.Println("disconnect timeout!")
			err := h.cleanupConnectionHandle(handle)
			if err != nil {
				fmt.Println(err)
			}
		}
	}(c)

	c.CancelContext()

	return h.Send(&cmd.Disconnect{
		ConnectionHandle: handle,
		Reason:           0x13,
	}, nil)
}

func (h *HCI) Context() context.Context {
	return h.ctx
}