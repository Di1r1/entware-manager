package monitor

import (
	"encoding/json"
	"entware-manager/internal/cgiutil"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"

	"entware-manager/internal/auth"
)

const monitorConfigFile = "/opt/web_entware/monitor_config.json"
const monitorPIDFile = "/tmp/entware/pid/watchdog.pid"

func HandleConfig() {
	if cgiutil.IsGET() {
		data := readConfigFromFile()
		if data == nil {
			data = defaultConfig()
		} else {
			// Auto-migrate: add max_processes/autostart if missing
			if _, ok := data["max_processes"]; !ok {
				data["max_processes"] = 200
				saveConfigToFile(data)
			}
			if _, ok := data["autostart"]; !ok {
				data["autostart"] = false
				saveConfigToFile(data)
			}
		}
		cgiutil.WriteJSON(data)
		return
	}

	if cgiutil.IsPOST() {
		if auth.IsCrossSiteOrigin() {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
			return
		}
		body, err := io.ReadAll(os.Stdin)
		if err != nil || len(body) == 0 {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Empty request"})
			return
		}

		var cfg map[string]interface{}
		if err := json.Unmarshal(body, &cfg); err != nil {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Invalid JSON"})
			return
		}

		saveConfigToFile(cfg)

		// Send HUP to watchdog daemon
		if pidData, err := os.ReadFile(monitorPIDFile); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(pidData))); err == nil {
				if proc, err := os.FindProcess(pid); err == nil {
					proc.Signal(syscall.SIGHUP)
				}
			}
		}

		logAction("INFO", "Сохранены настройки защиты")
		cgiutil.WriteJSON(map[string]string{"status": "ok"})
		return
	}

	cgiutil.NotAllowed()
}

func readConfigFromFile() map[string]interface{} {
	data, err := os.ReadFile(monitorConfigFile)
	if err != nil {
		return nil
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return cfg
}

func saveConfigToFile(cfg map[string]interface{}) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(monitorConfigFile, data, 0644)
}

func defaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"enabled":  false,
		"interval": 10,
		"individual": map[string]interface{}{
			"enabled":        true,
			"threshold_cpu":  80,
			"threshold_time": 300,
		},
		"ignore":        []string{"lighttpd", "cron", "ttyd", "watchdog"},
		"ignore_ps":     true,
		"max_processes": 200,
		"autostart":     false,
		"log_file":      "/tmp/entware/logs/monitor.log",
		"log_max_size":  1048576,
	}
}
