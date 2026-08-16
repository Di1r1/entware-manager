package services

import (
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

	if IsPOST() {
		if auth.IsCrossSiteOrigin() {
			WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
			return
		}
		body := readPOSTBody()
		params := parseFormBody(body)
		name = params["name"]
		action = params["action"]
	} else if IsGET() {
		name = getQueryParam("name")
		action = getQueryParam("action")
	} else {
		NotAllowed()
		return
	}

	if name == "" || action == "" {
		WriteJSON(map[string]string{"error": "name and action required"})
		return
	}

	switch action {
	case "start", "stop", "restart", "enable", "disable":
	default:
		WriteJSON(map[string]string{"error": "Недопустимое действие: " + action})
		return
	}

	// Find the service script
	script := findScript(name)
	if script == "" {
		WriteJSON(map[string]string{"error": "Служба " + name + " не найдена или не исполняема"})
		return
	}

	switch action {
	case "start", "stop", "restart":
		cmd := exec.Command(script, action)
		err := cmd.Run()
		if err == nil {
			logAction("INFO", fmt.Sprintf("Служба %s: %s", name, action))
			WriteJSON(map[string]string{"status": "ok"})
		} else {
			logAction("ERROR", fmt.Sprintf("Ошибка при %s службы %s", action, name))
			WriteJSON(map[string]string{"error": "Ошибка при выполнении " + action})
		}

	case "enable":
		base := filepath.Base(script)
		dir := filepath.Dir(script)
		var newName string
		if strings.HasPrefix(base, "S") {
			WriteJSON(map[string]string{"status": "ok"})
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
			WriteJSON(map[string]string{"status": "ok"})
		} else {
			WriteJSON(map[string]string{"error": "Не удалось включить автозапуск"})
		}

	case "disable":
		base := filepath.Base(script)
		dir := filepath.Dir(script)
		var newName string
		if strings.HasPrefix(base, "K") {
			WriteJSON(map[string]string{"status": "ok"})
			return
		}
		if strings.HasPrefix(base, "S") {
			newName = "K" + base[1:]
		} else {
			WriteJSON(map[string]string{"error": "Не удалось отключить автозапуск"})
			return
		}
		err := os.Rename(script, filepath.Join(dir, newName))
		if err == nil {
			logAction("INFO", fmt.Sprintf("Служба %s: отключен автозапуск", name))
			WriteJSON(map[string]string{"status": "ok"})
		} else {
			WriteJSON(map[string]string{"error": "Не удалось отключить автозапуск"})
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
	fmt.Fprintf(f, "[%s] [%s] [%s] [%d] [service_action] %s\n", timestamp, level, ip, os.Getpid(), message)
}
