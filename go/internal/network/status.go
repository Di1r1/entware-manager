package network

import (
	"encoding/json"
	"entware-manager/internal/cgiutil"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const pidFile = "/tmp/entware/pid/network_watchdog.pid"
const stateFile = "/tmp/entware/pid/network_watchdog_state.json"

type WatchdogStatus struct {
	Running   bool   `json:"running"`
	PID       string `json:"pid"`
	Uptime    string `json:"uptime"`
	LastCheck string `json:"last_check"`
}

type WatchdogState struct {
	Timestamp string `json:"timestamp"`
}

func HandleStatus() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	st := getWatchdogStatus()
	cgiutil.WriteJSON(st)
}

func getWatchdogStatus() WatchdogStatus {
	status := WatchdogStatus{
		Running:   false,
		PID:       "",
		Uptime:    "0",
		LastCheck: "",
	}

	pid, err := readPidFile()
	if err == nil {
		if pidAlive(pid) {
			status.Running = true
			status.PID = strconv.Itoa(pid)
			status.Uptime = calcUptime(pid)
		}
	}

	timestamp, err := readStateTimestamp()
	if err == nil && timestamp != "" {
		status.LastCheck = timestamp
	}

	return status
}

func readPidFile() (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func pidAlive(pid int) bool {
	statusPath := fmt.Sprintf("/proc/%d/status", pid)
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "State:") {
			state := strings.TrimSpace(line[6:])
			if strings.HasPrefix(state, "Z") {
				return false
			}
			return true
		}
	}
	return false
}

func calcUptime(pid int) string {
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return "0"
	}
	// Find the closing ) of comm field, then fields 20-22 are after it
	idx := strings.LastIndex(string(data), ")")
	if idx < 0 {
		return "0"
	}
	fields := strings.Fields(string(data)[idx+1:])
	if len(fields) < 20 {
		return "0"
	}
	// Field 20 in 0-indexed Go slice is fields[19] (start_time)
	startTicks, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil || startTicks == 0 {
		return "0"
	}

	uptimeData, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "0"
	}
	uptimeParts := strings.Fields(string(uptimeData))
	if len(uptimeParts) == 0 {
		return "0"
	}
	sysUptime, err := strconv.ParseFloat(uptimeParts[0], 64)
	if err != nil {
		return "0"
	}

	// startTicks is in USER_HZ (100) — convert to seconds
	uptimeSec := int(sysUptime) - int(startTicks/100)
	if uptimeSec <= 0 {
		return "0"
	}
	minutes := uptimeSec / 60
	seconds := uptimeSec % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

func readStateTimestamp() (string, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return "", err
	}
	var st WatchdogState
	if err := json.Unmarshal(data, &st); err != nil {
		return "", err
	}
	if st.Timestamp == "" || st.Timestamp == "null" {
		return "", nil
	}
	return st.Timestamp, nil
}
