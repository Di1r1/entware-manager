package network

import (
	"entware-manager/internal/cgiutil"
)

var (
	ConfigFile     = "/opt/web_entware/network_config.json"
	WatchdogScript = "/opt/web_entware/network_watchdog.sh"
	PidFile        = "/tmp/entware/pid/network_watchdog.pid"
	LogDir         = "/tmp/entware/logs"
)

// GetParam извлекает параметр из QUERY_STRING или тела POST.
func GetParam(key string) string {
	return cgiutil.GetParam(key)
}
