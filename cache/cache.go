package cache

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/rigado/ble"
	"io/ioutil"
	"os"
	"sync"
)

type gattCache struct {
	filename string
	lock sync.RWMutex
}

func New(filename string) ble.GattCache {
	gc := gattCache{
		filename: filename,
	}

	return &gc
}

func (gc *gattCache) Store(mac ble.Addr, profile ble.Profile, replace bool) error {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	cache, err := gc.loadExisting()
	if err != nil {
		return err
	}

	_, ok := cache[mac.String()]
	if ok && !replace {
		return fmt.Errorf("cache already contains gatt db for %s", mac.String())
	}

	cache[mac.String()] = profile

	err = gc.storeCache(cache)
	if err != nil {
		return err
	}

	return nil
}

func (gc *gattCache) Load(mac ble.Addr) (ble.Profile, error) {
	gc.lock.RLock()
	defer gc.lock.RUnlock()

	cache, err := gc.loadExisting()
	if err != nil {
		return ble.Profile{}, err
	}

	p, ok := cache[mac.String()]
	if !ok {
		return ble.Profile{}, fmt.Errorf("gatt db for %s not found in cache", mac.String())
	}

	return p, nil
}

func (gc *gattCache) Clear() error {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	err := os.Remove(gc.filename)
	if err != nil {
		return err
	}

	return nil
}

func (gc *gattCache) loadExisting() (map[string]ble.Profile, error) {
	_, err := os.Stat(gc.filename)
	if os.IsNotExist(err) {
		return map[string]ble.Profile{}, nil
	}

	in, err := ioutil.ReadFile(gc.filename)
	if err != nil {
		return nil, err
	}

	var cache map[string]ble.Profile
	err = jsoniter.Unmarshal(in, &cache)
	if err != nil {
		return nil, err
	}

	return cache, nil
}

func (gc *gattCache) storeCache(cache map[string]ble.Profile) error {
	out, err := jsoniter.Marshal(cache)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(gc.filename, out, 0644)
}
