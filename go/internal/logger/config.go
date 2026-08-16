package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const configFile = "/opt/web_entware/logger/config.json"
const systemLogFile = "/opt/var/log/entware/system.log"

type LoggerConfig struct {
	Enabled bool `json:"enabled"`
}

func HandleConfig() {
	if IsGET() {
		if hasQuery("pretty") {
			showPrettyConfig()
			return
		}
		data, err := os.ReadFile(configFile)
		if err != nil {
			WriteJSON(LoggerConfig{Enabled: true})
			return
		}
		var cfg LoggerConfig
		if json.Unmarshal(data, &cfg) != nil {
			WriteJSON(LoggerConfig{Enabled: true})
			return
		}
		WriteJSON(cfg)
		return
	}

	if IsPOST() {
		body, err := io.ReadAll(os.Stdin)
		if err != nil || len(body) == 0 {
			WriteJSON(map[string]string{"status": "error", "message": "Empty request"})
			return
		}

		var newCfg LoggerConfig
		if err := json.Unmarshal(body, &newCfg); err != nil {
			WriteJSON(map[string]string{"status": "error", "message": "Invalid JSON"})
			return
		}

		oldCfg := LoggerConfig{Enabled: true}
		if oldData, err := os.ReadFile(configFile); err == nil {
			json.Unmarshal(oldData, &oldCfg)
		}

		if newCfg.Enabled != oldCfg.Enabled {
			msg := "Логирование ОТКЛЮЧЕНО"
			if newCfg.Enabled {
				msg = "Логирование ВКЛЮЧЕНО"
			}
			logSystemEvent("INFO", msg+" (было: enabled="+fmt.Sprintf("%v", oldCfg.Enabled)+")")
		}

		os.WriteFile(configFile, body, 0644)
		WriteJSON(map[string]string{"status": "ok"})
		return
	}

	NotAllowed()
}

func showPrettyConfig() {
	fmt.Print("Content-type: text/html; charset=utf-8\n\n")

	enabled := true
	if data, err := os.ReadFile(configFile); err == nil {
		var cfg LoggerConfig
		if json.Unmarshal(data, &cfg) == nil {
			enabled = cfg.Enabled
		}
	}

	statusClass := "status-on"
	statusText := "● Включено"
	if !enabled {
		statusClass = "status-off"
		statusText = "● Отключено"
	}

	fmt.Print(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Настройки логирования</title>
	<link rel="stylesheet" href="/entware-manager/style.css?v=32">
	<script src="/entware-manager/theme.js?v=2"></script>
	<script>
		if (window.Theme) Theme.init();
	</script>
	<style>
		body { padding: 20px; background: var(--content-bg); color: var(--text-primary); font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
		.config-card { background: var(--modal-bg); border: 1px solid var(--border-color); border-radius: 12px; padding: 24px; max-width: 500px; margin: 0 auto; box-shadow: var(--shadow); }
		.config-item { display: flex; justify-content: space-between; padding: 12px 0; border-bottom: 1px solid var(--border-color); }
		.config-item:last-child { border-bottom: none; }
		.config-label { font-weight: 500; }
		.config-value { font-family: monospace; color: var(--accent); }
		.status-on { color: #2ecc71; }
		.status-off { color: #e74c3c; }
		.btn-back { display: inline-block; margin-top: 20px; padding: 10px 20px; background: var(--btn-gradient); color: var(--btn-text); border-radius: 8px; text-decoration: none; box-shadow: var(--btn-shadow); }
	</style>
</head>
<body>
<div class="config-card">
	<h2>Настройки логирования</h2>
	<div class="config-item">
		<span class="config-label">Состояние</span>
		<span class="config-value ` + statusClass + `">` + statusText + `</span>
	</div>
	<div class="config-item">
		<span class="config-label">Файлы логов</span>
		<span class="config-value">/tmp/entware/logs/</span>
	</div>
	<div class="config-item">
		<span class="config-label">Архив логов</span>
		<span class="config-value">/opt/var/log/entware/</span>
	</div>
	<div class="config-item">
		<span class="config-label">Системные события</span>
		<span class="config-value">/opt/var/log/entware/system.log</span>
	</div>
	<div class="config-item">
		<span class="config-label">Конфиг</span>
		<span class="config-value">/opt/web_entware/logger/config.json</span>
	</div>
	<a href="/entware-cgi/logger/view.cgi" class="btn-back">← К логам</a>
</div>
</body>
</html>`)
}

func logSystemEvent(level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	ip := os.Getenv("REMOTE_ADDR")
	if ip == "" {
		ip = "localhost"
	}
	os.MkdirAll(strings.TrimSuffix(systemLogFile, "/system.log"), 0755)
	f, err := os.OpenFile(systemLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] [%s] [%s] %s\n", timestamp, level, ip, message)
}
