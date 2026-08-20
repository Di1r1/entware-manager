// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package main

import (
	_ "entware-manager/internal/buildinfo"
	"entware-manager/internal/cgiutil"
	"os"

	_ "entware-manager/internal/localtime"
	"entware-manager/internal/monitor"
)

func main() {
	ep := os.Getenv("ENDPOINT")
	if ep == "" {
		cgiutil.WriteError("ENDPOINT not set")
		return
	}

	switch ep {
	case "monitor_status":
		monitor.HandleStatus()
	case "monitor_action":
		monitor.HandleAction()
	case "monitor_config":
		monitor.HandleConfig()
	case "monitor_log":
		monitor.HandleLog()
	case "temperature":
		monitor.HandleTemperature()
	case "wifi_temp":
		monitor.HandleWifiTemp()
	case "temp_history":
		monitor.HandleTempHistory()
	case "wifi_temp_history":
		monitor.HandleWifiTempHistory()
	case "kill_pid":
		monitor.HandleKillPID()
	default:
		cgiutil.WriteError("unknown endpoint: " + ep)
	}
}
