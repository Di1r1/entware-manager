package services

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ttydInstance struct {
	Port  int    `json:"port"`
	State string `json:"state"`
	PID   string `json:"pid"`
	Mode  string `json:"mode"`
}

type ttydStatus struct {
	Status   string       `json:"status"`
	HTop     ttydInstance `json:"htop"`
	Terminal ttydInstance `json:"terminal"`
}

func HandleTTYDControl() {
	if IsGET() {
		WriteJSON(getTTYDStatus())
		return
	}

	if IsPOST() {
		body := readPOSTBody()
		params := parseFormBody(body)
		action := params["action"]
		portStr := params["port"]
		pass := params["pass"]
		mode := params["mode"]

		port, err := strconv.Atoi(portStr)
		if err != nil || portStr == "" {
			WriteJSON(map[string]string{"status": "error", "message": "Некорректный порт"})
			return
		}

		switch action {
		case "start":
			WriteJSON(startTTYD(port, pass, mode))
		case "stop":
			WriteJSON(stopTTYD(port))
		case "restart":
			WriteJSON(restartTTYD(port, pass, mode))
		default:
			WriteJSON(map[string]string{"status": "error", "message": "Неизвестное действие"})
		}
		return
	}

	NotAllowed()
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
		if strings.Contains(pi.Cmdline, "ttyd") {
			if strings.Contains(pi.Cmdline, "8089") {
				st.HTop.State = "running"
				st.HTop.PID = strconv.Itoa(pi.PID)
			}
			if strings.Contains(pi.Cmdline, "9089") {
				st.Terminal.State = "running"
				st.Terminal.PID = strconv.Itoa(pi.PID)
				if strings.Contains(pi.Cmdline, "telnet") {
					st.Terminal.Mode = "telnet"
				} else {
					st.Terminal.Mode = "entware"
				}
			}
		}
	}

	return st
}

func startTTYD(port int, pass string, mode string) map[string]string {
	if mode == "" {
		mode = "entware"
	}

	var args []string
	args = append(args, "-p", strconv.Itoa(port), "-W")

	if port == 9089 {
		args = append(args, "--permit-any-origin")
		if pass != "" {
			args = append(args, "-c", "admin:"+pass)
		}
		switch mode {
		case "telnet":
			args = append(args, "telnet", "127.0.0.1")
		default:
			args = append(args, "/opt/bin/bash")
		}
	} else if port == 8089 {
		args = append(args, "htop")
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
		if !strings.Contains(pi.Cmdline, "ttyd") {
			continue
		}
		if !strings.Contains(pi.Cmdline, strconv.Itoa(port)) {
			continue
		}

		proc, err := os.FindProcess(pi.PID)
		if err != nil {
			continue
		}
		if err := proc.Kill(); err != nil {
			continue
		}

		time.Sleep(500 * time.Millisecond)

		if isTTYDRunning(port) {
			return map[string]string{"status": "error", "message": "Не удалось остановить ttyd на порту " + strconv.Itoa(port)}
		}

		logAction("INFO", fmt.Sprintf("ttyd остановлен на порту %d", port))
		return map[string]string{"status": "ok", "message": fmt.Sprintf("ttyd на порту %d остановлен", port)}
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
	portStr := strconv.Itoa(port)
	for _, pi := range procMap {
		if strings.HasPrefix(pi.State, "Z") {
			continue
		}
		if strings.Contains(pi.Cmdline, "ttyd") && strings.Contains(pi.Cmdline, portStr) {
			return true
		}
	}
	return false
}
