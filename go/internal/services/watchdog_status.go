package services

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

var (
	wrapperPidFile  = "/tmp/entware/pid/service_watchdog.pid"
	wrapperConfig   = "/opt/web_entware/service_config.json"
	wrapperPidState = "/tmp/entware/pid/service_watchdog_pids.json"
)

type WatchdogStatusResponse struct {
	Running bool            `json:"running"`
	PID     interface{}     `json:"pid"`
	Config  json.RawMessage `json:"config"`
	PIDs    json.RawMessage `json:"pids"`
}

func HandleWatchdogStatus() {
	if !IsGET() {
		NotAllowed()
		return
	}

	resp := WatchdogStatusResponse{
		Running: false,
		PID:     nil,
		PIDs:    json.RawMessage("{}"),
	}

	cfg := readWrapperConfig()
	if len(cfg) > 0 {
		resp.Config = cfg
	} else {
		resp.Config = json.RawMessage(defaultServiceConfig)
	}

	if pid := readWrapperPID(); pid > 0 && pidAlive(pid) {
		resp.Running = true
		resp.PID = pid
		resp.PIDs = readWrapperPIDs()
	}

	WriteJSON(resp)
}

func readWrapperPID() int {
	data, err := os.ReadFile(wrapperPidFile)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

func pidAlive(pid int) bool {
	statusPath := "/proc/" + strconv.Itoa(pid) + "/status"
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "State:") {
			state := strings.TrimSpace(line[6:])
			return !strings.HasPrefix(state, "Z")
		}
	}
	return false
}

func readWrapperConfig() json.RawMessage {
	data, err := os.ReadFile(wrapperConfig)
	if err != nil || !json.Valid(data) {
		return nil
	}
	return json.RawMessage(data)
}

func readWrapperPIDs() json.RawMessage {
	data, err := os.ReadFile(wrapperPidState)
	if err != nil || !json.Valid(data) {
		return json.RawMessage("{}")
	}
	return json.RawMessage(data)
}

const defaultServiceConfig = `{
  "enabled": true,
  "interval": 10,
  "mode": "initd",
  "watch_list": ["lighttpd","cron","ttyd","AdGuardHome","koolproxy","xray"],
  "auto_restart": false,
  "exclude_list": ["dropbear","kvas-ws","service_watchdog"],
  "log_to_monitor": true,
  "pid_history_days": 7
}`
