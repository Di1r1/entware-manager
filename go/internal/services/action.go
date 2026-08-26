package services

import (
	"entware-manager/internal/cgiutil"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"entware-manager/internal/auth"
)

func HandleServiceAction() {
	var name, action string

	if cgiutil.IsPOST() {
		if auth.IsCrossSiteOrigin() {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
			return
		}
		body := cgiutil.ReadPOSTBody()
		params := cgiutil.ParseFormBody(body)
		name = params["name"]
		action = params["action"]
	} else {
		cgiutil.NotAllowed()
		return
	}

	if name == "" || action == "" {
		cgiutil.WriteJSON(map[string]string{"error": "name and action required"})
		return
	}

	// Валидация имени службы: только буквы/цифры/"_"/"-" — защита от path traversal
	// (name="../../bin/reboot" не должен приводить к выполнению произвольного файла).
	if !serviceNameRe.MatchString(name) || len(name) > 64 {
		cgiutil.WriteJSON(map[string]string{"error": "Недопустимое имя службы"})
		return
	}

	switch action {
	case "start", "stop", "restart", "enable", "disable":
	default:
		cgiutil.WriteJSON(map[string]string{"error": "Недопустимое действие: " + action})
		return
	}

	// Find the service script
	script := findScript(name)
	if script == "" {
		cgiutil.WriteJSON(map[string]string{"error": "Служба " + name + " не найдена или не исполняема"})
		return
	}

	switch action {
	case "start", "stop", "restart":
		cmd := exec.Command(script, action)
		cmd.Env = append(os.Environ(), "HOME=/opt/root", "PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin")
		err := cmd.Run()
		if err == nil {
			logAction("INFO", fmt.Sprintf("Служба %s: %s", name, action))
			cgiutil.WriteJSON(map[string]string{"status": "ok"})
		} else {
			logAction("ERROR", fmt.Sprintf("Ошибка при %s службы %s", action, name))
			cgiutil.WriteJSON(map[string]string{"error": "Ошибка при выполнении " + action})
		}

	case "enable":
		base := filepath.Base(script)
		dir := filepath.Dir(script)
		var newName string
		if strings.HasPrefix(base, "S") {
			cgiutil.WriteJSON(map[string]string{"status": "ok"})
			return
		}
		if strings.HasPrefix(base, "K") {
			newName = "S" + base[1:]
		} else {
			newName = "S" + base
		}
		err := os.Rename(script, filepath.Join(dir, newName))
		if err == nil {
			logAction("INFO", fmt.Sprintf("Служба %s: включен автозапуск", name))
			cgiutil.WriteJSON(map[string]string{"status": "ok"})
		} else {
			cgiutil.WriteJSON(map[string]string{"error": "Не удалось включить автозапуск"})
		}

	case "disable":
		base := filepath.Base(script)
		dir := filepath.Dir(script)
		var newName string
		if strings.HasPrefix(base, "K") {
			cgiutil.WriteJSON(map[string]string{"status": "ok"})
			return
		}
		if strings.HasPrefix(base, "S") {
			newName = "K" + base[1:]
		} else {
			cgiutil.WriteJSON(map[string]string{"error": "Не удалось отключить автозапуск"})
			return
		}
		err := os.Rename(script, filepath.Join(dir, newName))
		if err == nil {
			logAction("INFO", fmt.Sprintf("Служба %s: отключен автозапуск", name))
			cgiutil.WriteJSON(map[string]string{"status": "ok"})
		} else {
			cgiutil.WriteJSON(map[string]string{"error": "Не удалось отключить автозапуск"})
		}
	}
}

func findScript(name string) string {
	// Имя службы в UI приходит как "80lighttpd" (префикс S + номер) — отрезаем цифры.
	// Спец-случай для S80entware-lighttpd убран: собственный lighttpd больше не
	// используется (режим entware-server), остаётся только стандартный S80lighttpd.
	for _, prefix := range []string{"S", "K", ""} {
		candidate := filepath.Join(servicesDir, prefix+name)
		fi, err := os.Stat(candidate)
		if err == nil && fi.Mode().IsRegular() && (fi.Mode().Perm()&0111) != 0 {
			return candidate
		}
	}
	return ""
}

// logAction пишет действия панели со службами (запуск/остановка, автозапуск,
// ttyd) в единый дневной суточный лог с тегом [service] — тот же, что у фактов
// демона service_watchdog. Все события служб (факты + действия) в одном логе
// и доходят до Telegram.
func logAction(level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
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
	fmt.Fprintf(f, "[%s] [%s] [%s] [%d] [service] %s\n", timestamp, level, ip, os.Getpid(), message)
}

// ServiceAction выполняет start/stop/restart для службы по имени.
// Экспортировано для интерактивного бота Telegram (v1.13): переиспользует
// findScript + логирование действий панели.
func ServiceAction(name, action string) error {
	if !serviceNameRe.MatchString(name) || len(name) > 64 {
		return fmt.Errorf("недопустимое имя службы")
	}
	switch action {
	case "start", "stop", "restart":
	default:
		return fmt.Errorf("недопустимое действие: %s", action)
	}
	script := findScript(name)
	if script == "" {
		return fmt.Errorf("служба %s не найдена или не исполняема", name)
	}
	if err := exec.Command(script, action).Run(); err != nil {
		logAction("ERROR", fmt.Sprintf("Ошибка при %s службы %s", action, name))
		return fmt.Errorf("ошибка при выполнении %s", action)
	}
	logAction("INFO", fmt.Sprintf("Служба %s: %s", name, action))
	return nil
}
