package ble

// AdvHandler handles advertisement.
type AdvHandler func(a Advertisement)

// AdvFilter returns true if the advertisement matches specified condition.
type AdvFilter func(a Advertisement) bool

// Advertisement ...
type Advertisement interface {
	LocalName() (string, error)
	ManufacturerData() ([]byte, error)
	ServiceData() ([]ServiceData, error)
	Services() ([]UUID, error)
	OverflowService() ([]UUID, error)
	TxPowerLevel() (int, error)
	Connectable() (bool, error)
	SolicitedService() ([]UUID, error)
	RSSI() (int, error)
	Addr() (Addr, error)

	ToMap() (map[string]interface{}, error)
}

var AdvertisementMapKeys = struct {
	MAC         string
	RSSI        string
	Name        string
	MFG         string
	Services    string
	ServiceData string
	Connectable string
	Solicited   string
	EventType   string
}{
	MAC:         "mac",
	RSSI:        "rssi",
	Name:        "name",
	MFG:         "mfg",
	Services:    "services",
	ServiceData: "serviceData",
	Connectable: "connectable",
	Solicited:   "solicited",
	EventType:   "eventType",
}

// ServiceData ...
type ServiceData struct {
	UUID UUID
	Data []byte
}
