// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package rdp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"entware-manager/internal/auth"
	"entware-manager/internal/cgiutil"
)

// HandleControl обрабатывает rdp_start / rdp_stop (только POST).
func HandleControl(action string) {
	if !cgiutil.IsPOST() {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Метод не поддерживается — используйте POST"})
		return
	}

	// защита: пароль + Origin-чек (CSRF)
	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Запрос из недоверенного источника (CSRF)"})
		return
	}
	password := cgiutil.ParseFormBody(cgiutil.ReadPOSTBody())["password"]
	allow, reason := auth.EnabledReports()
	if !allow {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": reason})
		return
	}
	if !auth.CheckPassword(password) {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Неверный пароль"})
		return
	}

	cfg, err := LoadConfig()
	if err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Ошибка чтения конфига: " + err.Error()})
		return
	}

	// Управление через init-скрипт S90grdp-proxy (идемпотентно, PID-файл).
	// init-скрипт сам читает порт из rdp_config.json.
	if _, statErr := os.Stat(ProxyInitScript); statErr != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "init-скрипт " + ProxyInitScript + " не найден — установите RDP-модуль"})
		return
	}

	out, err := runInit(action)
	if err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Ошибка: " + err.Error() + " " + strings.TrimSpace(out)})
		return
	}

	time.Sleep(500 * time.Millisecond)
	inst := Status()

	if action == "start" && inst.State != "running" {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Прокси не поднялся"})
		return
	}
	if action == "stop" && inst.State == "running" {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Прокси не остановился"})
		return
	}

	if action == "start" {
		logRDPAction("INFO", "RDP-прокси запущен (порт "+fmt.Sprintf("%d", cfg.ProxyPort)+")")
		cgiutil.WriteJSON(map[string]string{
			"status":  "ok",
			"message": "Прокси запущен на порту " + fmt.Sprintf("%d", cfg.ProxyPort),
			"port":    fmt.Sprintf("%d", cfg.ProxyPort),
			"pid":     inst.PID,
		})
		return
	}
	logRDPAction("INFO", "RDP-прокси остановлен")
	cgiutil.WriteJSON(map[string]string{"status": "ok", "message": "Прокси остановлен"})
}

// logRDPAction пишет событие RDP в дневной суточный лог с тегом [rdp] —
// его читает Telegram-шлюз (source=system).
func logRDPAction(level, msg string) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	ip := os.Getenv("REMOTE_ADDR")
	if ip == "" {
		ip = "localhost"
	}
	logDir := "/tmp/entware/logs"
	os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] [%s] [%s] [%d] [rdp] %s\n", ts, level, ip, os.Getpid(), msg)
}

func runInit(cmd string) (string, error) {
	c := exec.Command(ProxyInitScript, cmd)
	out, err := c.CombinedOutput()
	return string(out), err
}
