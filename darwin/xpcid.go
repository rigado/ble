package darwin

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

const (
	evtDescriptorWritten        = 80
	evtSlaveConnectionComplete  = 81
	evtMasterConnectionComplete = 82
)

var darwinOSVersion int

func getDarwinReleaseVersion() int {
	version, err := exec.Command("uname", "-r").Output()
	if err != nil {
		fmt.Println(err)
		return 0
	}
	v, _ := strconv.Atoi(strings.Split(version, ".")[0])
	return v
}

// xpc command IDs are OS X version specific, so we will use a map
// to be able to handle arbitrary versions
const (
	cmdInit = iota
	cmdAdvertiseStart
	cmdAdvertiseStop
	cmdScanningStart
	cmdScanningStop
	cmdServicesAdd
	cmdServicesRemove
	cmdSendData
	cmdSubscribed
	cmdConnect
	cmdDisconnect
	cmdReadRSSI
	cmdDiscoverServices
	cmdDiscoverIncludedServices
	cmdDiscoverCharacteristics
	cmdReadCharacteristic
	cmdWriteCharacteristic
	cmdSubscribeCharacteristic
	cmdDiscoverDescriptors
	cmdReadDescriptor
	cmdWriteDescriptor
	evtStateChanged
	evtAdvertisingStarted
	evtAdvertisingStopped
	evtServiceAdded
	evtReadRequest
	evtWriteRequest
	evtSubscribe
	evtUnsubscribe
	evtConfirmation
	evtPeripheralDiscovered
	evtPeripheralConnected
	evtPeripheralDisconnected
	evtATTMTU
	evtRSSIRead
	evtServiceDiscovered
	evtIncludedServicesDiscovered
	evtCharacteristicsDiscovered
	evtCharacteristicRead
	evtCharacteristicWritten
	evtNotificationValueSet
	evtDescriptorsDiscovered
	evtDescriptorRead
	evtDescriptorWritten
	evtSlaveConnectionComplete
	evtMasterConnectionComplete
)

// XpcIDs is the map of the commands for the current version of OS X
var xpcID map[int]int

func initXpcIDs() {
	darwinOSVersion = getDarwinReleaseVersion()

	xpcID := make(map[int]int)

	xpcID[cmdInit] = 1
	xpcID[cmdAdvertiseStart] = 8
	xpcID[cmdAdvertiseStop] = 9
	xpcID[cmdServicesAdd] = 10
	xpcID[cmdServicesRemove] = 12

	if darwinOSVersion < 17 {
		// yosemite
		xpcID[cmdSendData] = 13
		xpcID[cmdSubscribed] = 15
		xpcID[cmdScanningStart] = 29
		xpcID[cmdScanningStop] = 30
		xpcID[cmdConnect] = 31
		xpcID[cmdDisconnect] = 32
		xpcID[cmdReadRSSI] = 44
		xpcID[cmdDiscoverServices] = 45
		xpcID[cmdDiscoverIncludedServices] = 60
		xpcID[cmdDiscoverCharacteristics] = 62
		xpcID[cmdReadCharacteristic] = 65
		xpcID[cmdWriteCharacteristic] = 66
		xpcID[cmdSubscribeCharacteristic] = 68
		xpcID[cmdDiscoverDescriptors] = 70
		xpcID[cmdReadDescriptor] = 77
		xpcID[cmdWriteDescriptor] = 78

		xpcID[evtStateChanged] = 4
		xpcID[evtAdvertisingStarted] = 16
		xpcID[evtAdvertisingStopped] = 17
		xpcID[evtServiceAdded] = 18
		xpcID[evtReadRequest] = 19
		xpcID[evtWriteRequest] = 20
		xpcID[evtSubscribe] = 21
		xpcID[evtUnsubscribe] = 22
		xpcID[evtConfirmation] = 23
		xpcID[evtPeripheralDiscovered] = 37
		xpcID[evtPeripheralConnected] = 38
		xpcID[evtPeripheralDisconnected] = 40
		xpcID[evtATTMTU] = 53
		xpcID[evtRSSIRead] = 55
		xpcID[evtServiceDiscovered] = 56
		xpcID[evtIncludedServicesDiscovered] = 63
		xpcID[evtCharacteristicsDiscovered] = 64
		xpcID[evtCharacteristicRead] = 71
		xpcID[evtCharacteristicWritten] = 72
		xpcID[evtNotificationValueSet] = 74
		xpcID[evtDescriptorsDiscovered] = 76
		xpcID[evtDescriptorRead] = 79
		xpcID[evtDescriptorWritten] = 80
		xpcID[evtSlaveConnectionComplete] = 81
		xpcID[evtMasterConnectionComplete] = 21
	} else {
		// high sierra
		xpcID[cmdSendData] = 13 // TODO: find out the correct value for this
		xpcID[cmdScanningStart] = 44
		xpcID[cmdScanningStop] = 45
		xpcID[cmdConnect] = 46
		xpcID[cmdDisconnect] = 47
		xpcID[cmdReadRSSI] = 61
		xpcID[cmdDiscoverServices] = 62
		xpcID[cmdDiscoverIncludedServices] = 74
		xpcID[cmdDiscoverCharacteristics] = 75
		xpcID[cmdReadCharacteristic] = 78
		xpcID[cmdWriteCharacteristic] = 79
		xpcID[cmdSubscribeCharacteristic] = 81
		xpcID[cmdDiscoverDescriptors] = 82
		xpcID[cmdReadDescriptor] = 88
		xpcID[cmdWriteDescriptor] = 89

		xpcID[evtPeripheralDiscovered] = 48
		xpcID[evtPeripheralConnected] = 49
		xpcID[evtPeripheralDisconnected] = 50
		xpcID[evtRSSIRead] = 71
		xpcID[evtServiceDiscovered] = 72
		xpcID[evtCharacteristicsDiscovered] = 77
		xpcID[evtCharacteristicRead] = 83
		xpcID[evtCharacteristicWritten] = 84
		xpcID[evtNotificationValueSet] = 86
		xpcID[evtDescriptorsDiscovered] = 87
		xpcID[evtDescriptorRead] = 90
		xpcID[evtDescriptorWritten] = 91

		xpcID[evtIncludedServicesDiscovered] = 87
	}
}
