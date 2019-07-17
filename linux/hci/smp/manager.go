package smp

import (
	"fmt"
	"github.com/go-ble/ble/linux/hci"
	"log"
)

type manager struct {
	config hci.SmpConfig
	pairing *pairingContext
	t *transport
	bondManager hci.BondManager
	encrypt func(info hci.BondInfo) error
}

//todo: need to have on instance per connection which requires a mutex in the bond manager
//todo: remove bond manager from input parameters?
func NewSmpManager(config hci.SmpConfig, bm hci.BondManager) *manager {
	p := &pairingContext{request: config}
	m := &manager{config:config, pairing:p, bondManager: bm}
	t := NewSmpTransport(p, bm, m, nil)
	m.t = t
	return m
}

func (m *manager) SetConfig(config hci.SmpConfig) {
	m.config = config
}

func (m *manager) SetWritePDUFunc(w func([]byte) (int, error)) {
	m.t.writePDU = w
}

func (m *manager) SetEncryptFunc(e func(info hci.BondInfo) error) {
	m.encrypt = e
}

func (m *manager) InitContext(localAddr, remoteAddr []byte,
	localAddrType, remoteAddrType uint8) {
	if m.pairing == nil {
		m.pairing = &pairingContext{}
	}

	m.pairing.localAddr = swapBuf(localAddr)
	m.pairing.localAddrType = localAddrType
	m.pairing.remoteAddr = swapBuf(remoteAddr)
	m.pairing.remoteAddrType = remoteAddrType

	m.t.pairing = m.pairing
}

func (m *manager) Handle(in []byte) error {
	p := pdu(in)
	payload := p.payload()
	code := payload[0]
	data := payload[1:]
	v, ok := dispatcher[code]
	if !ok {
		fmt.Println("smp:", "unhandled smp code %v", code)
		return m.t.send([]byte{pairingFailed, 0x05})
	}

	if v.handler != nil {
		_, err := v.handler(m.t, data)
		if err != nil {
			log.Println(err)
			return err
		}

		return nil
		// return c.sendSMP(r)
	}

	fmt.Println("no smp handler...")
	// FIXME: work around to the lack of SMP implementation - always return non-supported.
	// C.5.1 Pairing Not Supported by Slave
	return m.t.send([]byte{pairingFailed, 0x05})
}

func (m *manager) Bond() error {

	keys, err := GenerateKeys()
	if err != nil {
		fmt.Println("error generating secure keys:", err)
	}

	//todo: can this be made less bad??
	m.pairing.scECDHKeys = keys
	m.t.pairing = m.pairing

	return m.t.sendPairingRequest()
}

//todo: implement if needed
func (m *manager) BondInfoFor(addr string) hci.BondInfo {
	bi, err := m.bondManager.Find(addr)
	if err != nil {
		fmt.Print(err)
		return nil
	}

	return bi
}

func (m *manager) LegacyPairingInfo() (bool, []byte) {
	if m.pairing.legacy {
		return true, m.pairing.shortTermKey
	}

	return false, nil
}

func (m *manager) EnableEncryption(addr string) error {
	return m.encrypt(m.pairing.bond)
}

func (m *manager) Encrypt() error {
	return m.encrypt(m.pairing.bond)
}
