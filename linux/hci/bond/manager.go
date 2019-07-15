package bond

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/go-ble/ble/linux/hci"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type manager struct {
	lock sync.RWMutex
}

type bondInfo struct {
	Bonds []remoteKeyInfo `json:"bonds"`
}

type remoteKeyInfo struct {
	Address string `json:"address"`
	LongTermKey string `json:"longTermKey"`
	EncryptionDiversifier string `json:"encryptionDiversifier"`
	RandomValue string `json:"randomValue"`
	Legacy bool `json:"legacy"`
}

const (
	bondFilename = "bonds.json"
)

func NewBondManager() hci.BondManager {
	return &manager{}
}

//todo: is this function really needed?
func (m *manager) Exists(addr string) bool {
	if len(addr) != 12 {
		return false
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	bonds, err := loadBonds()
	if err != nil {
		fmt.Print(err)
		return false
	}

	for _, b := range bonds.Bonds {
		if b.Address == addr {
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

	bonds, err := loadBonds()
	if err != nil {
		return nil, err
	}

	var bi hci.BondInfo
	for _, bond := range bonds.Bonds {
		if bond.Address == addr {
			//todo: if any of this is invalid, it should be deleted
			ltk, err := hex.DecodeString(bond.LongTermKey)
			if err != nil {
				return nil, fmt.Errorf("failed to decode long term key: %s", err)
			}

			eDiv, err := hex.DecodeString(bond.EncryptionDiversifier)
			if err != nil {
				return nil, fmt.Errorf("invalid ediv in bond file")
			}

			randVal, err := hex.DecodeString(bond.RandomValue)
			if err != nil {
				return nil, fmt.Errorf("invalid random value in bond file")
			}

			bi = hci.NewBondInfo(ltk, binary.LittleEndian.Uint16(eDiv), binary.LittleEndian.Uint64(randVal), bond.Legacy)
		}
	}

	if bi == nil {
		return nil, fmt.Errorf("bond information not found for %s", addr)
	}

	return bi, nil
}

func (m *manager) Save(addr string, bond hci.BondInfo) error {
	if len(addr) != 12 {
		return fmt.Errorf("invalid address: %s", addr)
	}

	if bond == nil {
		return fmt.Errorf("empty bond information")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	bonds, err := loadBonds()
	if err != nil {
		return err
	}

	rki := createRemoteKeyInfo(bond)
	rki.Address = addr

	bonds.Bonds = append(bonds.Bonds, rki)

	return storeBonds(bonds)
}

func createRemoteKeyInfo(bond hci.BondInfo) remoteKeyInfo {
	rki := remoteKeyInfo{}

	rki.LongTermKey = hex.EncodeToString(bond.LongTermKey())

	eDiv := make([]byte, 2)
	binary.LittleEndian.PutUint16(eDiv, bond.EDiv())

	randVal := make([]byte, 8)
	binary.LittleEndian.PutUint64(randVal, bond.Random())

	rki.EncryptionDiversifier = hex.EncodeToString(eDiv)
	rki.RandomValue = hex.EncodeToString(randVal)
	rki.Legacy = bond.Legacy()

	return rki
}

func loadBonds() (*bondInfo, error) {
	//open local file
	bondFile := filepath.Join(os.Getenv("SNAP_DATA"), bondFilename)
	_, err := os.Stat(bondFile)
	var f *os.File
	if os.IsNotExist(err) {
		f, err = os.Create(bondFile)
		if err != nil {
			return nil, fmt.Errorf("unable to create bond file: %s",err)
		}
		_ = f.Close()
	}

	fileData, err := ioutil.ReadFile(bondFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read bond file information: %s", err)
	}

	var bonds bondInfo
	if len(fileData) > 0 {
		err = json.Unmarshal(fileData, &bonds)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal current bond info: %s", err)
		}
	}

	if len(bonds.Bonds) == 0 {
		bonds.Bonds = make([]remoteKeyInfo, 0, 1)
	}

	return &bonds, nil
}

func storeBonds(bonds *bondInfo) error {
	bondFile := filepath.Join(os.Getenv("SNAP_DATA"), bondFilename)
	out, err := json.Marshal(bonds)
	if err != nil {
		return fmt.Errorf("failed to marshal bonds to json: %s", err)
	}

	err = ioutil.WriteFile(bondFile, out, 0644)
	if err != nil {
		return fmt.Errorf("failed to update bond information: %s", err)
	}

	return nil
}

