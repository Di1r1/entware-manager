package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	watchdogScript = "/opt/web_entware/watchdog.sh"
)

func HandleAction() {
	if !IsPOST() {
		NotAllowed()
		return
	}

	body := readPOSTBody()
	params := parseFormBody(body)
	action := params["action"]

	switch action {
	case "start", "stop", "restart":
		handleDaemonAction(action)
	case "kill":
		handleKill(params["pid"])
	case "clearlog":
		handleClearLog()
	default:
		logMonitor("ERROR", "Неизвестное действие: "+action)
		WriteJSON(map[string]string{"status": "error", "message": "Неизвестное действие"})
	}
}

func handleDaemonAction(action string) {
	logMonitor("INFO", "Запрос на "+strings.ToUpper(action)+" демона")

	cmd := exec.Command(watchdogScript, action)
	err := cmd.Run()

	if action == "start" || action == "restart" {
		time.Sleep(1 * time.Second)
		pid, err2 := readPIDFile()
		if err2 == nil && pidAlive(pid) {
			logMonitor("INFO", fmt.Sprintf("Демон %s (PID: %d)", map[string]string{"start":"запущен","restart":"перезапущен"}[action], pid))
			logAction("INFO", fmt.Sprintf("Демон защиты %s (PID: %d)", map[string]string{"start":"запущен","restart":"перезапущен"}[action], pid))
			WriteJSON(map[string]interface{}{"status": "ok", "message": fmt.Sprintf("Демон %s", map[string]string{"start":"запущен","restart":"перезапущен"}[action]), "pid": pid})
		} else {
			logMonitor("ERROR", "Демон не запустился")
			WriteJSON(map[string]string{"status": "error", "message": "Демон не запустился"})
		}
		return
	}

	if err != nil {
		logMonitor("ERROR", "Не удалось выполнить действие: "+action)
		WriteJSON(map[string]string{"status": "error", "message": "Не удалось " + action})
		return
	}

	logMonitor("INFO", "Демон "+action)
	logAction("INFO", "Демон защиты "+action)
	WriteJSON(map[string]string{"status": "ok", "message": "Демон " + action})
}

func handleKill(pidStr string) {
	pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
	if err != nil || !pidAlive(pid) {
		logMonitor("WARN", "Попытка убить несуществующий процесс "+pidStr)
		WriteJSON(map[string]string{"status": "error", "message": "Процесс не найден"})
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		WriteJSON(map[string]string{"status": "error", "message": "Процесс не найден"})
		return
	}

	if err := proc.Signal(syscall.SIGKILL); err != nil {
		WriteJSON(map[string]string{"status": "error", "message": "Не удалось убить процесс"})
		return
	}

	logMonitor("INFO", fmt.Sprintf("Убит процесс %d по запросу пользователя", pid))
	WriteJSON(map[string]string{"status": "ok", "message": "Процесс убит"})
}

func handleClearLog() {
	logFilePath := getLogFilePath()
	if logFilePath == "" {
		logFilePath = "/tmp/entware/logs/monitor.log"
	}
	os.Truncate(logFilePath, 0)
	logMonitor("INFO", "Лог очищен")
	WriteJSON(map[string]string{"status": "ok", "message": "Лог очищен"})
}

func getLogFilePath() string {
	cfg := readConfigFromFile()
	if cfg != nil {
		if path, ok := cfg["log_file"].(string); ok {
			return path
		}
	}
	return ""
}

func logMonitor(level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logDir := "/tmp/entware/logs"
	os.MkdirAll(logDir, 0755)
	logFile := fmt.Sprintf("%s/%s.log", logDir, time.Now().Format("2006-01-02"))
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] [%s] [ACTION] %s\n", timestamp, level, message)
}

func logAction(level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
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
	fmt.Fprintf(f, "[%s] [%s] [%s] [%d] [monitor_action] %s\n", timestamp, level, ip, os.Getpid(), message)
}
