package network

import (
	"entware-manager/internal/cgiutil"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"entware-manager/internal/auth"
)

func HandleAction() {
	if !cgiutil.IsPOST() {
		cgiutil.NotAllowed()
		return
	}
	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
		return
	}
	action := GetParam("action")
	switch action {
	case "start":
		handleStart()
	case "stop":
		handleStop()
	case "restart":
		handleRestart()
	default:
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Неизвестное действие: " + action})
	}
}

func handleStart() {
	if pid := readPID(); pid > 0 && pidAlive(pid) {
		cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "message": "Демон уже запущен", "pid": pid})
		return
	}
	os.Remove(PidFile)

	cmd := exec.Command(WatchdogScript, "start")
	if err := cmd.Run(); err != nil {
		logNetworkAction("ERROR", "Не удалось запустить демон сети")
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Не удалось запустить демон"})
		return
	}
	time.Sleep(1 * time.Second)

	if pid := readPID(); pid > 0 {
		logNetworkAction("INFO", "Запрос на START демона сети")
		cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "message": "Демон запущен", "pid": pid})
	} else {
		logNetworkAction("ERROR", "Не удалось запустить демон сети")
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Не удалось запустить демон"})
	}
}

func handleStop() {
	cmd := exec.Command(WatchdogScript, "stop")
	cmd.Run()
	logNetworkAction("INFO", "Запрос на STOP демона сети")
	cgiutil.WriteJSON(map[string]string{"status": "ok", "message": "Демон остановлен"})
}

func handleRestart() {
	cmd := exec.Command(WatchdogScript, "restart")
	if err := cmd.Run(); err != nil {
		logNetworkAction("ERROR", "Не удалось перезапустить демон сети")
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Не удалось перезапустить демон"})
		return
	}
	time.Sleep(1 * time.Second)

	if pid := readPID(); pid > 0 {
		logNetworkAction("INFO", "Запрос на RESTART демона сети")
		cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "message": "Демон перезапущен", "pid": pid})
	} else {
		logNetworkAction("ERROR", "Не удалось перезапустить демон сети")
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Не удалось перезапустить демон"})
	}
}

// logNetworkAction пишет событие в дневной суточный лог с тегом [network] —
// его читает Telegram-шлюз (source=network).
func logNetworkAction(level, msg string) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	ip := os.Getenv("REMOTE_ADDR")
	if ip == "" {
		ip = "localhost"
	}
	logDir := "/tmp/entware/logs"
	os.MkdirAll(logDir, 0755)
	logFile := fmt.Sprintf("%s/%s.log", logDir, time.Now().Format("2006-01-02"))
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] [%s] [%s] [%d] [network] %s\n", ts, level, ip, os.Getpid(), msg)
}

func readPID() int {
	data, err := os.ReadFile(PidFile)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}
