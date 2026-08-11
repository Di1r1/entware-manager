// Di1r1
package rdp

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"entware-manager/internal/auth"
)

// HandleControl обрабатывает rdp_start / rdp_stop (только POST).
func HandleControl(action string) {
	if !IsPOST() {
		WriteJSON(map[string]string{"status": "error", "message": "Метод не поддерживается — используйте POST"})
		return
	}

	// защита: пароль + Origin-чек (CSRF)
	if auth.IsCrossSiteOrigin() {
		WriteJSON(map[string]string{"status": "error", "message": "Запрос из недоверенного источника (CSRF)"})
		return
	}
	password := parseForm(readPOSTBody())["password"]
	allow, reason := auth.EnabledReports()
	if !allow {
		WriteJSON(map[string]string{"status": "error", "message": reason})
		return
	}
	if !auth.CheckPassword(password) {
		WriteJSON(map[string]string{"status": "error", "message": "Неверный пароль"})
		return
	}

	cfg, err := LoadConfig()
	if err != nil {
		WriteJSON(map[string]string{"status": "error", "message": "Ошибка чтения конфига: " + err.Error()})
		return
	}

	// Управление через init-скрипт S90grdp-proxy (идемпотентно, PID-файл).
	// init-скрипт сам читает порт из rdp_config.json.
	if _, statErr := os.Stat(ProxyInitScript); statErr != nil {
		WriteJSON(map[string]string{"status": "error", "message": "init-скрипт " + ProxyInitScript + " не найден — установите RDP-модуль"})
		return
	}

	out, err := runInit(action)
	if err != nil {
		WriteJSON(map[string]string{"status": "error", "message": "Ошибка: " + err.Error() + " " + strings.TrimSpace(out)})
		return
	}

	time.Sleep(500 * time.Millisecond)
	inst := Status()

	if action == "start" && inst.State != "running" {
		WriteJSON(map[string]string{"status": "error", "message": "Прокси не поднялся"})
		return
	}
	if action == "stop" && inst.State == "running" {
		WriteJSON(map[string]string{"status": "error", "message": "Прокси не остановился"})
		return
	}

	if action == "start" {
		WriteJSON(map[string]string{
			"status":  "ok",
			"message": "Прокси запущен на порту " + fmt.Sprintf("%d", cfg.ProxyPort),
			"port":    fmt.Sprintf("%d", cfg.ProxyPort),
			"pid":     inst.PID,
		})
		return
	}
	WriteJSON(map[string]string{"status": "ok", "message": "Прокси остановлен"})
}

func runInit(cmd string) (string, error) {
	c := exec.Command(ProxyInitScript, cmd)
	out, err := c.CombinedOutput()
	return string(out), err
}
