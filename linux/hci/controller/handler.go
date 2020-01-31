package controller

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rigado/ble/linux/hci"
	"github.com/rigado/ble/linux/hci/evt"
)

func (h *HCI) handlePkt(b []byte) error {
	// Strip the 1-byte HCI header and pass down the rest of the packet.
	t, b := b[0], b[1:]
	switch t {
	case hci.PktTypeACLData:
		return h.handleACL(b)
	case hci.PktTypeEvent:
		return h.handleEvt(b)

		//unhandled stuff
	case hci.PktTypeCommand:
		return fmt.Errorf("unmanaged cmd: % X", b)
	case hci.PktTypeSCOData:
		return fmt.Errorf("unsupported sco packet: % X", b)
	case hci.PktTypeVendor:
		return fmt.Errorf("unsupported vendor packet: % X", b)
	default:
		return fmt.Errorf("invalid packet: 0x%02X % X", t, b)
	}
}

func (h *HCI) handleACL(b []byte) error {
	handle := getConnHandle(b)
	if handle == 0xffff { //invalid connection handle
		return fmt.Errorf("invalid connection handle")
	}

	h.muConns.Lock()
	defer h.muConns.Unlock()

	if c, ok := h.conns[handle]; ok {
		c.PutPacket(b)
	} else {
		//todo: properly fix this
		_ = hci.Logger.Warn("invalid connection handle on ACL packet", "handle", handle)
	}

	return nil
}

func getConnHandle(aclPacket []byte) uint16 {
	if len(aclPacket) >= 2 {
		return uint16(aclPacket[0]) | (uint16(aclPacket[1]&0x0f) << 8)
	}

	return 0xffff
}

func (h *HCI) handleEvt(b []byte) error {
	code, plen := int(b[0]), int(b[1])
	if plen != len(b[2:]) {
		return fmt.Errorf("invalid event packet: % X", b)
	}

	if code == evt.CommandCompleteCode || code == evt.CommandStatusCode {
		if f := h.evth[code]; f != nil {
			return f(b[2:])
		}
	}

	if f := h.evth[code]; f != nil {
		h.err = f(b[2:])
		return nil
	}
	if code == 0xff { // Ignore vendor events
		return nil
	}
	return fmt.Errorf("unsupported event packet: % X", b)
}

func (h *HCI) handleLEMeta(b []byte) error {
	subcode := int(b[0])
	if f := h.subh[subcode]; f != nil {
		return f(b)
	}
	return fmt.Errorf("unsupported LE event: % X", b)
}

func (h *HCI) handleLEAdvertisingReport(b []byte) error {
	if h.advHandler == nil {
		return nil
	}

	var a *hci.Advertisement
	var err error

	e := evt.LEAdvertisingReport(b)

	nr, err := e.NumReportsWErr()
	if err != nil {
		ee := h.makeAdvError(errors.Wrap(err, "advRep numReports"), e, true)
		return ee
	}

	//DSC: zephyr currently returns 1 report per report wrapper
	if nr != 1 {
		ee := h.makeAdvError(fmt.Errorf("invalid rep count %v", nr), e, true)
		return ee
	}

	for i := 0; i < int(nr); i++ {
		var et byte
		et, err = e.EventTypeWErr(i)
		if err != nil {
			h.makeAdvError(errors.Wrap(err, "advRep eventType"), e, true)
			continue
		}

		switch et {
		case hci.EvtTypAdvInd: //0x00
			fallthrough
		case hci.EvtTypAdvScanInd: //0x02
			a, err = hci.NewAdvertisement(e, i)
			if err != nil {
				h.makeAdvError(errors.Wrap(err, fmt.Sprintf("newAdv (typ %v)", et)), e, true)
				continue
			}
			h.adHist[h.adLast] = a
			h.adLast++
			if h.adLast == len(h.adHist) {
				h.adLast = 0
			}

			//advInd, advScanInd

		case hci.EvtTypScanRsp: //0x04
			sr, err := hci.NewAdvertisement(e, i)
			if err != nil {
				h.makeAdvError(errors.Wrap(err, fmt.Sprintf("newAdv (typ %v)", et)), e, true)
				continue
			}

			for idx := h.adLast - 1; idx != h.adLast; idx-- {
				if idx == -1 {
					idx = len(h.adHist) - 1
					if idx == h.adLast {
						break
					}
				}
				if h.adHist[idx] == nil {
					break
				}

				//bad addr?
				addrh, err := h.adHist[idx].AddrWErr()
				if err != nil {
					h.makeAdvError(errors.Wrap(err, fmt.Sprintf("adHist addr (typ %v)", et)), e, true)
					break
				}

				//bad addr?
				addrsr, err := sr.AddrWErr()
				if err != nil {
					h.makeAdvError(errors.Wrap(err, fmt.Sprintf("srAddr (typ %v)", et)), e, true)
					break
				}

				//set the scan response here
				if addrh.String() == addrsr.String() {
					//this will leave everything alone if there is an error when we attach the scanresp
					err = h.adHist[idx].SetScanResponse(sr)
					if err != nil {
						h.makeAdvError(errors.Wrap(err, fmt.Sprintf("setScanResp (typ %v)", et)), e, true)
						break
					}
					a = h.adHist[idx]
					break
				}
			} //for

			// Got a SR without having received an associated AD before?
			if a == nil {
				ee := h.makeAdvError(errors.Wrap(err, fmt.Sprintf("scanRsp (typ %v) w/o associated advData, srAddr %v", et, sr.Addr())), e, true)
				return ee
			}
			// sr

		case hci.EvtTypAdvDirectInd: //0x01
			fallthrough
		case hci.EvtTypAdvNonconnInd: //0x03
			a, err = hci.NewAdvertisement(e, i)
			if err != nil {
				h.makeAdvError(errors.Wrap(err, fmt.Sprintf("newAdv (typ %v)", et)), e, true)
				continue
			}

		default:
			h.makeAdvError(fmt.Errorf("invalid eventType %v", et), e, true)
			continue
		} // switch

		if a == nil {
			h.makeAdvError(fmt.Errorf("nil advertisement (i %v, typ %v)", i, et), e, true)
			continue
		}

		//dispatch
		if h.advHandlerSync {
			h.advHandler(a)
		} else {
			go h.advHandler(a)
		}

	} //for

	return nil
}

func (h *HCI) handleCommandComplete(b []byte) error {
	e := evt.CommandComplete(b)
	h.setAllowedCommands(int(e.NumHCICommandPackets()))

	// NOP command, used for flow control purpose [Vol 2, Part E, 4.4]
	// no handling other than setAllowedCommands needed
	if e.CommandOpcode() == 0x0000 {
		return nil
	}
	h.muSent.Lock()
	p, found := h.sent[int(e.CommandOpcode())]
	h.muSent.Unlock()

	if !found {
		return fmt.Errorf("can't find the cmd for CommandCompleteEP: % X", e)
	}

	select {
	case <-h.done:
		return fmt.Errorf("hci closed")
	case p.done <- e.ReturnParameters():
		return nil
	}
}

func (h *HCI) handleCommandStatus(b []byte) error {
	e := evt.CommandStatus(b)

	if !e.Valid() {
		err := fmt.Errorf("invalid command status: %v", e)
		h.dispatchError(err)
		return err
	}

	h.setAllowedCommands(int(e.NumHCICommandPackets()))

	h.muSent.Lock()
	p, found := h.sent[int(e.CommandOpcode())]
	h.muSent.Unlock()
	if !found {
		return fmt.Errorf("can't find the cmd for CommandStatusEP: % X", e)
	}

	select {
	case <-h.done:
		return fmt.Errorf("hci closed")
	case p.done <- []byte{e.Status()}:
		return nil
	}
}

func (h *HCI) handleLEConnectionComplete(b []byte) error {
	e := evt.LEConnectionComplete(b)
	c := h.cf.Create(h, e)

	h.muConns.Lock()
	h.conns[e.ConnectionHandle()] = c
	h.muConns.Unlock()

	if e.Role() == hci.RoleMaster {
		if e.Status() == 0x00 {
			select {
			case h.chMasterConn <- c:
			default:
				go c.Close()
			}
			return nil
		}
		if hci.ErrCommand(e.Status()) == hci.ErrConnID {
			// The connection was canceled successfully.
			return nil
		}
		return nil
	}
	if e.Status() == 0x00 {
		h.chSlaveConn <- c
		// When a controller accepts a connection, it moves from advertising
		// state to idle/ready state. Host needs to explicitly ask the
		// controller to re-enable advertising. Note that the host was most
		// likely in advertising state. Otherwise it couldn't accept the
		// connection in the first place. The only exception is that user
		// asked the host to stop advertising during this tiny window.
		// The re-enabling might failed or ignored by the controller, if
		// it had reached the maximum number of concurrent connections.
		// So we also re-enable the advertising when a connection disconnected
		h.params.RLock()
		if h.params.advEnable.AdvertisingEnable == 1 {
			go h.Send(&h.params.advEnable, nil)
		}
		h.params.RUnlock()
	}
	return nil
}

func (h *HCI) handleLEConnectionUpdateComplete(b []byte) error {
	return nil
}

func (h *HCI) cleanupConnectionHandle(handle uint16) error {

	h.muConns.Lock()
	defer h.muConns.Unlock()
	c, found := h.conns[handle]
	if !found {
		return fmt.Errorf("disconnecting an invalid handle %04X", handle)
	}

	delete(h.conns, handle)
	fmt.Println("cleanupConnectionHandle: close c.chInPkt")
	c.CloseInputChannel()

	if !h.isOpen() && c.Role() == hci.RoleSlave {
		// Re-enable advertising, if it was advertising. Refer to the
		// handleLEConnectionComplete() for details.
		// This may failed with ErrCommandDisallowed, if the controller
		// was actually in advertising state. It does no harm though.
		h.params.RLock()
		if h.params.advEnable.AdvertisingEnable == 1 {
			go h.Send(&h.params.advEnable, nil)
		}
		h.params.RUnlock()
	} else {
		// remote peripheral disconnected
		c.SetClosed()
	}
	// When a connection disconnects, all the sent packets and weren't acked yet
	// will be recycled. [Vol2, Part E 4.3]
	//
	// must be done with the pool locked to avoid race conditions where
	// writePDU is in progress and does a Get from the pool after this completes,
	// leaking a buffer from the main pool.
	c.BufferPool().Lock()
	c.BufferPool().PutAll()
	c.BufferPool().Unlock()
	return nil
}

func (h *HCI) handleDisconnectionComplete(b []byte) error {
	e := evt.DisconnectionComplete(b)
	ch := e.ConnectionHandle()
	fmt.Printf("disconnect complete for %v, reason: %v; cleanup connection handle\n",
		e.ConnectionHandle(), e.Reason())
	return h.cleanupConnectionHandle(ch)
}

func (h *HCI) handleEncryptionChange(b []byte) error {
	return nil
}

func (h *HCI) handleNumberOfCompletedPackets(b []byte) error {
	e := evt.NumberOfCompletedPackets(b)
	h.muConns.Lock()
	defer h.muConns.Unlock()
	for i := 0; i < int(e.NumberOfHandles()); i++ {
		c, found := h.conns[e.ConnectionHandle(i)]
		if !found {
			continue
		}

		// Put the delivered buffers back to the pool.
		for j := 0; j < int(e.HCNumOfCompletedPackets(i)); j++ {
			c.BufferPool().Put()
		}
	}
	return nil
}

func (h *HCI) handleLELongTermKeyRequest(b []byte) error {
	////todo: probably need to support this
	//e := evt.LELongTermKeyRequest(b)
	//panic(nil)
	//return h.Send(&cmd.LELongTermKeyRequestNegativeReply{
	//	ConnectionHandle: e.ConnectionHandle(),
	//}, nil)
	return nil
}
