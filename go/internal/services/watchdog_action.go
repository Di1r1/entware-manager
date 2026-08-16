package services

import (
	"os"
	"os/exec"
	"time"

	"entware-manager/internal/auth"
)

var (
	watchdogScript = "/opt/web_entware/service_watchdog.sh"
	watchdogLog    = "/tmp/entware/logs/service_events.log"
)

func HandleWatchdogAction() {
	if !IsGET() && !IsPOST() {
		NotAllowed()
		return
	}
	if IsPOST() && auth.IsCrossSiteOrigin() {
		WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
		return
	}
	action := getQueryParam("action")
	if action == "" {
		body := readPOSTBody()
		params := parseFormBody(body)
		action = params["action"]
	}

	switch action {
	case "start":
		handleWrapperStart()
	case "stop":
		handleWrapperStop()
	case "restart":
		handleWrapperRestart()
	default:
		WriteJSON(map[string]string{"status": "error", "message": "Неизвестное действие: " + action})
	}
}

func handleWrapperStart() {
	if pid := readWrapperPID(); pid > 0 && pidAlive(pid) {
		WriteJSON(map[string]interface{}{"status": "ok", "message": "Демон уже запущен", "pid": pid})
		return
	}

	if _, err := os.Stat(watchdogScript); os.IsNotExist(err) {
		WriteJSON(map[string]string{"status": "error", "message": "Демон не найден: " + watchdogScript})
		return
	}

	os.Remove(wrapperPidFile)

	f, err := os.OpenFile(watchdogLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
	}

	cmd := exec.Command(watchdogScript, "start")
	if f != nil {
		cmd.Stdout = f
		cmd.Stderr = f
	}
	if err := cmd.Run(); err != nil {
		WriteJSON(map[string]string{"status": "error", "message": "Не удалось запустить демон"})
		return
	}
	time.Sleep(1 * time.Second)

	if pid := readWrapperPID(); pid > 0 {
		logAction("INFO", "Демон watchdog запущен вручную (PID: "+itoa(pid)+")")
		WriteJSON(map[string]interface{}{"status": "ok", "message": "Демон запущен", "pid": pid})
	} else {
		logAction("ERROR", "Не удалось запустить демон watchdog")
		WriteJSON(map[string]string{"status": "error", "message": "Не удалось запустить демон"})
	}
}

func handleWrapperStop() {
	cmd := exec.Command(watchdogScript, "stop")
	cmd.Run()
	logAction("INFO", "Демон watchdog остановлен вручную")
	WriteJSON(map[string]string{"status": "ok", "message": "Демон остановлен"})
}

func handleWrapperRestart() {
	cmd := exec.Command(watchdogScript, "restart")
	if err := cmd.Run(); err != nil {
		WriteJSON(map[string]string{"status": "error", "message": "Не удалось перезапустить демон"})
		return
	}
	time.Sleep(1 * time.Second)

	if pid := readWrapperPID(); pid > 0 {
		logAction("INFO", "Демон watchdog перезапущен вручную (PID: "+itoa(pid)+")")
		WriteJSON(map[string]interface{}{"status": "ok", "message": "Демон перезапущен", "pid": pid})
	} else {
		logAction("ERROR", "Не удалось перезапустить демон watchdog")
		WriteJSON(map[string]string{"status": "error", "message": "Не удалось перезапустить демон"})
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 || pos == len(buf) {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
