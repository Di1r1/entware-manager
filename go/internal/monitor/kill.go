package monitor

import (
	"entware-manager/internal/cgiutil"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"entware-manager/internal/auth"
)

func HandleKillPID() {
	var pidStr string

	switch os.Getenv("REQUEST_METHOD") {
	case "POST":
		if auth.IsCrossSiteOrigin() {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
			return
		}
		body, _ := io.ReadAll(os.Stdin)
		params := cgiutil.ParseFormBody(string(body))
		pidStr = params["pid"]
	default:
		cgiutil.WriteJSON(map[string]string{"status": "error", "error": "Метод не поддерживается"})
		return
	}

	pidStr = strings.TrimSpace(pidStr)
	if pidStr == "" {
		cgiutil.WriteJSON(map[string]string{"status": "error", "error": "PID не указан или неверный"})
		return
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		cgiutil.WriteJSON(map[string]string{"status": "error", "error": "PID не указан или неверный"})
		return
	}

	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); os.IsNotExist(err) {
		cgiutil.WriteJSON(map[string]string{"status": "error", "error": fmt.Sprintf("Процесс с PID %d не найден", pid)})
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "error": fmt.Sprintf("Не удалось найти процесс PID=%d", pid)})
		return
	}

	if err := proc.Signal(os.Kill); err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "error": fmt.Sprintf("Не удалось завершить процесс PID=%d", pid)})
		return
	}

	logMonitor("INFO", fmt.Sprintf("Принудительно завершён процесс PID=%d", pid))
	cgiutil.WriteJSON(map[string]string{"status": "ok"})
}
