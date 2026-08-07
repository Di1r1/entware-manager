package monitor

import (
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
)

const monitorConfigFile = "/opt/web_entware/monitor_config.json"
const monitorPIDFile = "/tmp/entware/pid/watchdog.pid"

func HandleConfig() {
	if IsGET() {
		data := readConfigFromFile()
		if data == nil {
			data = defaultConfig()
		} else {
			// Auto-migrate: add max_processes if missing
			if _, ok := data["max_processes"]; !ok {
				data["max_processes"] = 200
				saveConfigToFile(data)
			}
		}
		WriteJSON(data)
		return
	}

	if IsPOST() {
		body, err := io.ReadAll(os.Stdin)
		if err != nil || len(body) == 0 {
			WriteJSON(map[string]string{"status": "error", "message": "Empty request"})
			return
		}

		var cfg map[string]interface{}
		if err := json.Unmarshal(body, &cfg); err != nil {
			WriteJSON(map[string]string{"status": "error", "message": "Invalid JSON"})
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
		WriteJSON(map[string]string{"status": "ok"})
		return
	}

	NotAllowed()
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
		"log_file":      "/tmp/entware/logs/monitor.log",
		"log_max_size":  1048576,
	}
}
