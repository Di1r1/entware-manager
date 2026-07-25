package network

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func HandleAction() {
	if !IsGET() && !IsPOST() {
		NotAllowed()
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
		WriteJSON(map[string]string{"status": "error", "message": "Неизвестное действие: " + action})
	}
}

func handleStart() {
	if pid := readPID(); pid > 0 && pidAlive(pid) {
		WriteJSON(map[string]interface{}{"status": "error", "message": "Демон уже запущен", "pid": pid})
		return
	}
	os.Remove(PidFile)

	cmd := exec.Command(WatchdogScript, "start")
	if err := cmd.Run(); err != nil {
		WriteJSON(map[string]string{"status": "error", "message": "Не удалось запустить демон"})
		return
	}
	time.Sleep(1 * time.Second)

	if pid := readPID(); pid > 0 {
		WriteJSON(map[string]interface{}{"status": "ok", "message": "Демон запущен", "pid": pid})
	} else {
		WriteJSON(map[string]string{"status": "error", "message": "Не удалось запустить демон"})
	}
}

func handleStop() {
	cmd := exec.Command(WatchdogScript, "stop")
	cmd.Run()
	WriteJSON(map[string]string{"status": "ok", "message": "Демон остановлен"})
}

func handleRestart() {
	cmd := exec.Command(WatchdogScript, "restart")
	if err := cmd.Run(); err != nil {
		WriteJSON(map[string]string{"status": "error", "message": "Не удалось перезапустить демон"})
		return
	}
	time.Sleep(1 * time.Second)

	if pid := readPID(); pid > 0 {
		WriteJSON(map[string]interface{}{"status": "ok", "message": "Демон перезапущен", "pid": pid})
	} else {
		WriteJSON(map[string]string{"status": "error", "message": "Не удалось перезапустить демон"})
	}
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


