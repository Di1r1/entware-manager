package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"entware-manager/internal/auth"
)

const (
	watchdogScript = "/opt/web_entware/watchdog.sh"
)

func HandleAction() {
	if !IsPOST() {
		NotAllowed()
		return
	}

	if auth.IsCrossSiteOrigin() {
		WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
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

	output, err := exec.Command(watchdogScript, action).CombinedOutput()
	outStr := strings.TrimSpace(string(output))

	if action == "start" || action == "restart" {
		if err != nil {
			if strings.Contains(outStr, "Already running") {
				pid, _ := readPIDFile()
				logMonitor("INFO", "Демон уже запущен (PID: "+strconv.Itoa(pid)+")")
				WriteJSON(map[string]interface{}{"status": "ok", "message": "Демон уже запущен", "pid": pid})
			} else {
				logMonitor("ERROR", "Не удалось "+action+" демон: "+outStr)
				WriteJSON(map[string]string{"status": "error", "message": "Не удалось " + action + " демон"})
			}
			return
		}
		time.Sleep(1 * time.Second)
		pid, err2 := readPIDFile()
		if err2 == nil && pidAlive(pid) {
			verb := map[string]string{"start": "запущен", "restart": "перезапущен"}[action]
			// Полная строка с PID пишется самим демоном (watchdog.sh daemon_loop):
			// «[monitor] Демон запущен (PID $$), ENABLED=…» — здесь дубли не пишем.
			WriteJSON(map[string]interface{}{"status": "ok", "message": fmt.Sprintf("Демон %s", verb), "pid": pid})
		} else {
			logMonitor("ERROR", "Демон не запустился: "+outStr)
			WriteJSON(map[string]string{"status": "error", "message": "Демон не запустился"})
		}
		return
	}

	if err != nil {
		logMonitor("ERROR", "Не удалось выполнить действие: "+action+": "+outStr)
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
	logFile := fmt.Sprintf("/tmp/entware/logs/%s.log", time.Now().Format("2006-01-02"))

	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			logMonitor("INFO", "Лог очищен")
			WriteJSON(map[string]string{"status": "ok", "message": "Лог очищен"})
			return
		}
		WriteJSON(map[string]string{"status": "error", "message": "Не удалось очистить лог"})
		return
	}

	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "[monitor]") || strings.Contains(lower, "[action]") {
			continue
		}
		kept = append(kept, line)
	}

	if err := os.WriteFile(logFile, []byte(strings.Join(kept, "\n")), 0644); err != nil {
		WriteJSON(map[string]string{"status": "error", "message": "Не удалось очистить лог"})
		return
	}
	logMonitor("INFO", "Лог очищен")
	WriteJSON(map[string]string{"status": "ok", "message": "Лог очищен"})
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
