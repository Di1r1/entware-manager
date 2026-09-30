package network

import (
	"encoding/json"
	"entware-manager/internal/cgiutil"
	"io"
	"os"

	"entware-manager/internal/auth"
)

const defaultConfig = `{
  "autostart": false,
  "watchdog": {
    "enabled": true,
    "interval": 30,
    "watch_interfaces": ["eth0"],
    "watch_internet": true,
    "ping_host": "8.8.8.8",
    "ping_timeout": 5,
    "notify_on": ["interface_down", "no_internet", "ip_changed"]
  }
}`

// flatWatchdogKeys — ключи, которые старые версии писали плоско, а демоны
// читают из .watchdog (network_watchdog.sh:33,56,99). При GET переносятся
// внутрь; вложенное имеет приоритет (демон его и читает).
var flatWatchdogKeys = []string{
	"interval", "watch_interfaces", "watch_internet",
	"ping_host", "ping_timeout", "notify_on",
}

func HandleConfig() {
	switch os.Getenv("REQUEST_METHOD") {
	case "GET":
		handleConfigGet()
	case "POST":
		if auth.IsCrossSiteOrigin() {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
			return
		}
		handleConfigPost()
	default:
		cgiutil.NotAllowed()
	}
}

func handleConfigGet() {
	data, err := os.ReadFile(ConfigFile)
	if err != nil || !json.Valid(data) {
		data = []byte(defaultConfig)
	} else {
		var cfg map[string]interface{}
		if json.Unmarshal(data, &cfg) == nil {
			dirty := false
			if _, ok := cfg["autostart"]; !ok {
				cfg["autostart"] = false
				dirty = true
			}
			if migrateFlatWatchdog(cfg) {
				dirty = true
				// Best effort: не записалось — отдадим мигрированное
				// в ответе, файл сойдётся на следующем GET.
				if migrated, err := json.MarshalIndent(cfg, "", "  "); err == nil {
					_ = cgiutil.WriteFileAtomic(ConfigFile, migrated, 0644)
				}
			}
			if dirty {
				if patched, err := json.MarshalIndent(cfg, "", "  "); err == nil {
					data = patched
				}
			}
		}
	}
	os.Stdout.WriteString("Content-type: application/json; charset=utf-8\n\n")
	os.Stdout.Write(data)
}

// migrateFlatWatchdog переносит плоские ключи демона в .watchdog.
// Возвращает true, если структура менялась. Неизвестные поля, sidebar,
// paths, version, .autostart и существующее содержимое .watchdog (включая
// NOC-ключи watch_wifi/wifi_weak_rssi) не трогает.
func migrateFlatWatchdog(cfg map[string]interface{}) bool {
	changed := false
	var wd map[string]interface{}
	if raw, ok := cfg["watchdog"]; ok {
		if m, ok := raw.(map[string]interface{}); ok {
			wd = m
		}
	}
	if wd == nil {
		wd = map[string]interface{}{}
	}
	for _, k := range flatWatchdogKeys {
		v, ok := cfg[k]
		if !ok {
			continue
		}
		if _, exists := wd[k]; !exists {
			wd[k] = v
		}
		delete(cfg, k)
		changed = true
	}
	if changed {
		cfg["watchdog"] = wd
	}
	return changed
}

func handleConfigPost() {
	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Failed to read request"})
		return
	}

	data := string(body)
	if !json.Valid([]byte(data)) {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Invalid JSON"})
		return
	}

	if err := cgiutil.WriteFileAtomic(ConfigFile, []byte(data), 0644); err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Failed to write config"})
		return
	}

	cgiutil.WriteJSON(map[string]string{"status": "ok"})
}
