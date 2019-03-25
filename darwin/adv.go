package darwin

import (
	"github.com/go-ble/ble"
	"github.com/raff/goble/xpc"
)

//DSC: todo detect/return errors with (type,bool)

type adv struct {
	args xpc.Dict
	ad   xpc.Dict
}

func (a *adv) LocalName() (string, bool) {
	return a.ad.GetString("kCBAdvDataLocalName", a.args.GetString("kCBMsgArgName", "")), true
}

func (a *adv) ManufacturerData() ([]byte, bool) {
	return a.ad.GetBytes("kCBAdvDataManufacturerData", nil), true
}

func (a *adv) ServiceData() ([]ble.ServiceData, bool) {
	xSDs, ok := a.ad["kCBAdvDataServiceData"]
	if !ok {
		return nil, true
	}

	xSD := xSDs.(xpc.Array)
	var sd []ble.ServiceData
	for i := 0; i < len(xSD); i += 2 {
		sd = append(
			sd, ble.ServiceData{
				UUID: ble.UUID(xSD[i].([]byte)),
				Data: xSD[i+1].([]byte),
			})
	}
	return sd, true
}

func (a *adv) Services() ([]ble.UUID, bool) {
	xUUIDs, ok := a.ad["kCBAdvDataServiceUUIDs"]
	if !ok {
		return nil, nil, true
	}
	var uuids []ble.UUID
	for _, xUUID := range xUUIDs.(xpc.Array) {
		uuids = append(uuids, ble.UUID(ble.Reverse(xUUID.([]byte))))
	}
	return nil, uuids, true
}

func (a *adv) OverflowService() ([]ble.UUID, bool) {
	return nil, nil, true // TODO
}

func (a *adv) TxPowerLevel() (int, bool) {
	return a.ad.GetInt("kCBAdvDataTxPowerLevel", 0), true
}

func (a *adv) SolicitedService() ([]ble.UUID, bool) {
	return nil, nil, true // TODO
}

func (a *adv) Connectable() (bool, bool) {
	return false, a.ad.GetInt("kCBAdvDataIsConnectable", 0) > 0, true
}

func (a *adv) RSSI() (int, bool) {
	return a.args.GetInt("kCBMsgArgRssi", 0), true
}

func (a *adv) Addr() (ble.Addr, bool) {
	return a.args.MustGetUUID("kCBMsgArgDeviceUUID"), true
}
