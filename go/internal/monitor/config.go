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

// monitorConfigFile — путь к конфигу. Переменная (не const), чтобы тесты
// могли подменять путь и не трогать /opt роутера.
var monitorConfigFile = "/opt/web_entware/monitor_config.json"

const monitorPIDFile = "/tmp/entware/pid/watchdog.pid"

func HandleConfig() {
	if cgiutil.IsGET() {
		// Чтение без побочных эффектов (G3): дефолты сливаем в памяти,
		// на диск не пишем. Демон имеет свои дефолты при чтении файла.
		cgiutil.WriteJSON(withDefaults(readConfigFromFile()))
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

		if existing := readConfigFromFile(); existing != nil {
			for k, v := range existing {
				if _, ok := cfg[k]; !ok {
					cfg[k] = v
				}
			}
		}

		if err := saveConfigToFile(cfg); err != nil {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Failed to write config"})
			return
		}

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

// withDefaults сливает прочитанный конфиг поверх дефолтов (два уровня:
// топ-ключи и вложенный individual). Неизвестные ключи файла сохраняются,
// диск не трогается.
func withDefaults(data map[string]interface{}) map[string]interface{} {
	merged := defaultConfig()
	if data == nil {
		return merged
	}
	for k, v := range data {
		merged[k] = v
	}
	if ind, ok := data["individual"].(map[string]interface{}); ok {
		if dind, ok := merged["individual"].(map[string]interface{}); ok {
			base := map[string]interface{}{}
			for k, v := range dind {
				base[k] = v
			}
			for k, v := range ind {
				base[k] = v
			}
			merged["individual"] = base
		} else {
			merged["individual"] = ind
		}
	}
	return merged
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

func saveConfigToFile(cfg map[string]interface{}) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return cgiutil.WriteFileAtomic(monitorConfigFile, data, 0644)
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
