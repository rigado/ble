package ble

import log "github.com/mgutz/logxi/v1"

var Logger = log.New("ble")

func SetLogLevelDebug() {
	Logger.SetLevel(log.LevelDebug)
}
