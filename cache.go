package ble

type GattCache interface {
	Store(Addr, Profile, bool) error
	Load(Addr) (Profile, error)
	Clear() error
}
