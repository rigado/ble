package bond

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/rigado/ble/linux/hci"
	"io/ioutil"
	"os"
	"sync"
)

type manager struct {
	filePath string
	lock     sync.RWMutex
	bonds    map[string]bondData
}

type bondData struct {
	LongTermKey           string `json:"longTermKey"`
	EncryptionDiversifier string `json:"encryptionDiversifier"`
	RandomValue           string `json:"randomValue"`
	Legacy                bool   `json:"legacy"`
}

const (
	defaultBondFilename = "bonds.json"
)

func NewBondManager(bondFilePath string) hci.BondManager {
	if len(bondFilePath) == 0 {
		bondFilePath = defaultBondFilename
	}
	return &manager{
		filePath: bondFilePath,
	}
}

//todo: is this function really needed?
func (m *manager) Exists(addr string) bool {
	if len(addr) != 12 {
		return false
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	bonds, err := m.loadBonds()
	if err != nil {
		fmt.Print(err)
		return false
	}

	for k := range bonds {
		if k == addr {
			return true
		}
	}

	return false
}

func (m *manager) Find(addr string) (hci.BondInfo, error) {
	if len(addr) != 12 {
		return nil, fmt.Errorf("invalid address")
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	bonds, err := m.loadBonds()
	if err != nil {
		return nil, err
	}

	bd, ok := bonds[addr]
	if !ok {
		return nil, fmt.Errorf("bond information not found for %s", addr)
	}

	//validate bondData information; if any of it is invalid, delete the bondData
	bi, bondErr := createBondInfo(bd)
	if bondErr != nil {
		delete(bonds, addr)
		err := m.storeBonds(bonds)
		if err != nil {
			fmt.Printf("bondData manager err: %s\n", err)
		}
		return nil, fmt.Errorf("found invalid bondData information: %s\n", bondErr)
	}

	return bi, nil
}

func (m *manager) Save(addr string, bond hci.BondInfo) error {
	if len(addr) != 12 {
		return fmt.Errorf("invalid address: %s", addr)
	}

	if bond == nil {
		return fmt.Errorf("empty bondData information")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	bonds, err := m.loadBonds()
	if err != nil {
		return err
	}

	bd := createBondData(bond)

	//check to see if this address already exists
	if _, ok := bonds[addr]; ok {
		fmt.Printf("replacing existing bondData for %s\n", addr)
	}

	bonds[addr] = bd

	return m.storeBonds(bonds)
}

func (m *manager) Delete(addr string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	bonds, err := m.loadBonds()
	if err != nil {
		return err
	}

	if _, ok := bonds[addr]; ok {
		delete(bonds, addr)
	} else {
		return fmt.Errorf("bond for mac %v not found", addr)
	}

	err = m.storeBonds(bonds)
	if err != nil {
		return err
	}

	return nil
}

//this is mutex protected at the public function level
func (m *manager) loadBonds() (map[string]bondData, error) {
	//open local file
	_, err := os.Stat(m.filePath)
	var f *os.File
	if os.IsNotExist(err) {
		f, err = os.Create(m.filePath)
		if err != nil {
			return nil, fmt.Errorf("unable to create bondData file: %s", err)
		}
		_ = f.Close()
	}

	fileData, err := ioutil.ReadFile(m.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read bondData file information: %s", err)
	}

	var bonds map[string]bondData
	if len(fileData) > 0 {
		err = json.Unmarshal(fileData, &bonds)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal current bondData info: %s", err)
		}
	}

	if len(bonds) == 0 {
		bonds = make(map[string]bondData)
	}

	return bonds, nil
}

//this is mutex protected at the public function level
func (m *manager) storeBonds(bonds map[string]bondData) error {
	out, err := json.Marshal(bonds)
	if err != nil {
		return fmt.Errorf("failed to marshal bonds to json: %s", err)
	}

	err = ioutil.WriteFile(m.filePath, out, 0644)
	if err != nil {
		return fmt.Errorf("failed to update bondData information: %s", err)
	}

	return nil
}

//bondData is a local structure
func createBondData(bi hci.BondInfo) bondData {
	b := bondData{}

	b.LongTermKey = hex.EncodeToString(bi.LongTermKey())

	eDiv := make([]byte, 2)
	binary.LittleEndian.PutUint16(eDiv, bi.EDiv())

	randVal := make([]byte, 8)
	binary.LittleEndian.PutUint64(randVal, bi.Random())

	b.EncryptionDiversifier = hex.EncodeToString(eDiv)
	b.RandomValue = hex.EncodeToString(randVal)
	b.Legacy = bi.Legacy()

	return b
}

//BondInfo is defined in the HCI packaged an used internally to enable
//encryption after a connect has been established with a device
func createBondInfo(b bondData) (hci.BondInfo, error) {
	ltk, err := hex.DecodeString(b.LongTermKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode long term key: %s", err)
	}
	if len(ltk) == 0 {
		return nil, fmt.Errorf("invalid long term key length")
	}

	eDiv, err := hex.DecodeString(b.EncryptionDiversifier)
	if err != nil {
		return nil, fmt.Errorf("invalid ediv in bondData file")
	}

	randVal, err := hex.DecodeString(b.RandomValue)
	if err != nil {
		return nil, fmt.Errorf("invalid random value in bondData file")
	}

	bi := hci.NewBondInfo(ltk, binary.LittleEndian.Uint16(eDiv), binary.LittleEndian.Uint64(randVal), b.Legacy)
	return bi, nil
}
