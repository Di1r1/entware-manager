package services

import (
	"entware-manager/internal/cgiutil"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"entware-manager/internal/auth"
)

type ttydInstance struct {
	Port  int    `json:"port"`
	State string `json:"state"`
	PID   string `json:"pid"`
	Mode  string `json:"mode"`
}

// ttydIndexHTML — путь к форку index.html ttyd (добавляет перехват вставки).
// Пустой — использовать встроенный index.html ttyd (вставка только Shift+Insert/правая кнопка).
const ttydIndexHTML = "/opt/web_entware/static/ttyd/index.html"

type ttydStatus struct {
	Status   string       `json:"status"`
	HTop     ttydInstance `json:"htop"`
	Terminal ttydInstance `json:"terminal"`
}

func HandleTTYDControl() {
	if cgiutil.IsGET() {
		cgiutil.WriteJSON(getTTYDStatus())
		return
	}

	if cgiutil.IsPOST() {
		if auth.IsCrossSiteOrigin() {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
			return
		}
		body := cgiutil.ReadPOSTBody()
		params := cgiutil.ParseFormBody(body)
		action := params["action"]
		portStr := params["port"]
		pass := params["pass"]
		mode := params["mode"]

		port, err := strconv.Atoi(portStr)
		if err != nil || portStr == "" {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Некорректный порт"})
			return
		}

		switch action {
		case "start":
			cgiutil.WriteJSON(startTTYD(port, pass, mode))
		case "stop":
			cgiutil.WriteJSON(stopTTYD(port))
		case "restart":
			cgiutil.WriteJSON(restartTTYD(port, pass, mode))
		default:
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Неизвестное действие"})
		}
		return
	}

	cgiutil.NotAllowed()
}

func getTTYDStatus() ttydStatus {
	st := ttydStatus{
		Status:   "ok",
		HTop:     ttydInstance{Port: 8089, State: "stopped", PID: "", Mode: "htop"},
		Terminal: ttydInstance{Port: 9089, State: "stopped", PID: "", Mode: "entware"},
	}

	procMap := scanProc()
	for _, pi := range procMap {
		if strings.HasPrefix(pi.State, "Z") {
			continue
		}
		if isTTYDOnPort(pi.Cmdline, 8089) {
			st.HTop.State = "running"
			st.HTop.PID = strconv.Itoa(pi.PID)
		}
		if isTTYDOnPort(pi.Cmdline, 9089) {
			st.Terminal.State = "running"
			st.Terminal.PID = strconv.Itoa(pi.PID)
			if strings.Contains(pi.Cmdline, "telnet") {
				st.Terminal.Mode = "telnet"
			} else {
				st.Terminal.Mode = "entware"
			}
		}
	}

	return st
}

func startTTYD(port int, pass string, mode string) map[string]string {
	if mode == "" {
		mode = "entware"
	}
	if pass == "" {
		return map[string]string{"status": "error", "message": "Пароль обязателен: терминал доступен извне через прокси, без пароля запуск запрещён"}
	}

	var args []string
	args = append(args, "-p", strconv.Itoa(port), "-W", "--permit-any-origin", "-c", "admin:"+pass)

	// Форк index.html: xterm.js 5.4 по Ctrl+V шлёт в PTY литеральный ^V
	// (не вставляет). Свой index.html добавляет перехват Ctrl+V/Cmd+V/
	// Shift+Insert → term.paste() (работает и по HTTP).
	if idx := ttydIndexHTML; idx != "" {
		args = append(args, "-I", idx)
	}

	if port == 9089 {
		args = append(args, "-i", "lo", "--base-path", "/terminal")
		switch mode {
		case "telnet":
			args = append(args, "telnet", "127.0.0.1")
		default:
			args = append(args, "/opt/bin/bash")
		}
	} else if port == 8089 {
		args = append(args, "-i", "lo", "--base-path", "/htop", "htop")
	} else {
		return map[string]string{"status": "error", "message": "Неизвестный порт"}
	}

	cmd := exec.Command("ttyd", args...)
	if err := cmd.Start(); err != nil {
		return map[string]string{"status": "error", "message": "Не удалось запустить ttyd: " + err.Error()}
	}

	time.Sleep(1 * time.Second)

	if isTTYDRunning(port) {
		logAction("INFO", fmt.Sprintf("ttyd запущен на порту %d mode=%s", port, mode))
		return map[string]string{"status": "ok", "message": fmt.Sprintf("ttyd запущен на порту %d", port)}
	}

	return map[string]string{"status": "error", "message": "Не удалось запустить ttyd на порту " + strconv.Itoa(port)}
}

func stopTTYD(port int) map[string]string {
	procMap := scanProc()
	for _, pi := range procMap {
		if strings.HasPrefix(pi.State, "Z") {
			continue
		}
		if !isTTYDOnPort(pi.Cmdline, port) {
			continue
		}

		proc, err := os.FindProcess(pi.PID)
		if err != nil {
			continue
		}
		if err := proc.Kill(); err != nil {
			continue
		}

		// Даём процессу и его детям время умереть: дочерние процессы ttyd
		// могут переживать родителя на сотни миллисекунд.
		for i := 0; i < 4; i++ {
			time.Sleep(500 * time.Millisecond)
			if !isTTYDRunning(port) {
				logAction("INFO", fmt.Sprintf("ttyd остановлен на порту %d", port))
				return map[string]string{"status": "ok", "message": fmt.Sprintf("ttyd на порту %d остановлен", port)}
			}
		}

		return map[string]string{"status": "error", "message": "Не удалось остановить ttyd на порту " + strconv.Itoa(port)}
	}

	return map[string]string{"status": "error", "message": "ttyd на порту " + strconv.Itoa(port) + " не найден"}
}

func restartTTYD(port int, pass string, mode string) map[string]string {
	stopTTYD(port)
	time.Sleep(1 * time.Second)
	return startTTYD(port, pass, mode)
}

func isTTYDRunning(port int) bool {
	procMap := scanProc()
	for _, pi := range procMap {
		if strings.HasPrefix(pi.State, "Z") {
			continue
		}
		if isTTYDOnPort(pi.Cmdline, port) {
			return true
		}
	}
	return false
}

// isTTYDOnPort определяет, что процесс — это именно демон ttyd (argv[0] == "ttyd"),
// слушающий указанный порт. Матчинг по подстроке "ttyd"/порта в cmdline запрещён:
// он ловит собственный процесс CGI (cmdline содержит "ttyd_control", "--port 8089").
func isTTYDOnPort(cmdline string, port int) bool {
	args := strings.Fields(cmdline)
	if len(args) == 0 {
		return false
	}
	if filepath.Base(args[0]) != "ttyd" {
		return false
	}
	portStr := strconv.Itoa(port)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-p" || a == "--port" {
			if i+1 < len(args) && args[i+1] == portStr {
				return true
			}
			continue
		}
		if strings.HasPrefix(a, "--port=") && a[len("--port="):] == portStr {
			return true
		}
		if a == portStr {
			return true
		}
	}
	return false
}
