package services

import (
	"encoding/json"
	"entware-manager/internal/cgiutil"
	"io"
	"os"
	"syscall"
	"time"

	"entware-manager/internal/auth"
)

func HandleWatchdogConfig() {
	switch method := os.Getenv("REQUEST_METHOD"); method {
	case "GET":
		handleWrapperConfigGet()
	case "POST":
		if auth.IsCrossSiteOrigin() {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
			return
		}
		handleWrapperConfigPost()
	default:
		cgiutil.NotAllowed()
	}
}

func handleWrapperConfigGet() {
	data, err := os.ReadFile(wrapperConfig)
	if err != nil || !json.Valid(data) {
		data = []byte(defaultServiceConfig)
	} else {
		var cfg map[string]interface{}
		if json.Unmarshal(data, &cfg) == nil {
			if _, ok := cfg["enabled"]; !ok {
				cfg["enabled"] = true
				if patched, err := json.MarshalIndent(cfg, "", "  "); err == nil {
					data = patched
				}
			}
			if _, ok := cfg["autostart"]; !ok {
				cfg["autostart"] = false
				if patched, err := json.MarshalIndent(cfg, "", "  "); err == nil {
					data = patched
				}
			}
		}
	}
	os.Stdout.WriteString("Content-type: application/json; charset=utf-8\n\n")
	os.Stdout.Write(data)
}

func handleWrapperConfigPost() {
	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Failed to read request"})
		return
	}

	if !json.Valid(body) {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Invalid JSON configuration"})
		return
	}

	var newCfg map[string]interface{}
	if err := json.Unmarshal(body, &newCfg); err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Failed to parse JSON"})
		return
	}

	var existingCfg map[string]interface{}
	existingData, err := os.ReadFile(wrapperConfig)
	if err == nil && json.Valid(existingData) {
		json.Unmarshal(existingData, &existingCfg)
	}

	merged := make(map[string]interface{})
	for k, v := range existingCfg {
		merged[k] = v
	}
	for k, v := range newCfg {
		merged[k] = v
	}

	if _, ok := merged["enabled"]; !ok {
		merged["enabled"] = true
	}
	if _, ok := merged["interval"]; !ok {
		merged["interval"] = 10
	}
	if _, ok := merged["mode"]; !ok {
		merged["mode"] = "initd"
	}
	if _, ok := merged["watch_list"]; !ok {
		merged["watch_list"] = []string{"lighttpd", "cron", "ttyd", "AdGuardHome", "koolproxy", "xray"}
	}
	if _, ok := merged["auto_restart"]; !ok {
		merged["auto_restart"] = false
	}
	if _, ok := merged["autostart"]; !ok {
		merged["autostart"] = false
	}
	if _, ok := merged["exclude_list"]; !ok {
		merged["exclude_list"] = []string{"dropbear", "kvas-ws", "service_watchdog"}
	}
	if _, ok := merged["log_to_monitor"]; !ok {
		merged["log_to_monitor"] = true
	}
	if _, ok := merged["pid_history_days"]; !ok {
		merged["pid_history_days"] = 7
	}

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Failed to marshal config"})
		return
	}

	if err := cgiutil.WriteFileAtomic(wrapperConfig, out, 0644); err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Failed to write config"})
		return
	}

	if pid := readWrapperPID(); pid > 0 && pidAlive(pid) {
		syscall.Kill(pid, syscall.SIGHUP)
		time.Sleep(time.Second)
		cgiutil.WriteJSON(map[string]string{"status": "ok", "message": "Конфигурация сохранена, демон перезагружен"})
	} else {
		cgiutil.WriteJSON(map[string]string{"status": "ok", "message": "Конфигурация сохранена (демон не запущен)"})
	}
}
