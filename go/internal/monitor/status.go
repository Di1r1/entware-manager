package monitor

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Process struct {
	PID     int     `json:"pid"`
	PCPU    string  `json:"pcpu"`
	Time    string  `json:"time"`
	Command string  `json:"command"`
}

type StatusResponse struct {
	DaemonStatus string    `json:"demon_status"`
	DaemonPID    string    `json:"demon_pid"`
	Processes    []Process `json:"processes"`
}

const (
	pidFile  = "/tmp/entware/pid/watchdog.pid"
	clkTck   = 100
)

func HandleStatus() {
	if !IsGET() {
		NotAllowed()
		return
	}

	daemonStatus := "stopped"
	daemonPID := ""

	if pid, err := readPIDFile(); err == nil {
		if pidAlive(pid) {
			daemonStatus = "running"
			daemonPID = strconv.Itoa(pid)
		} else {
			os.Remove(pidFile)
		}
	}

	topProcs := getTopProcesses(5)
	if topProcs == nil {
		topProcs = []Process{}
	}

	WriteJSON(StatusResponse{
		DaemonStatus: daemonStatus,
		DaemonPID:    daemonPID,
		Processes:    topProcs,
	})
}

func readPIDFile() (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func pidAlive(pid int) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
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

type cpuInfo struct {
	PID      int
	TotalTicks int64
	StartTicks int64
	Comm     string
	Cmdline  string
}

func getTopProcesses(n int) []Process {
	uptime := readUptime()
	if uptime <= 0 {
		return nil
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	var procs []cpuInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}

		statData, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			continue
		}
		statStr := string(statData)

		idx := strings.LastIndex(statStr, ")")
		if idx < 0 {
			continue
		}
		fields := strings.Fields(statStr[idx+1:])
		if len(fields) < 22 {
			continue
		}

		utime, _ := strconv.ParseInt(fields[11], 10, 64)   // field 14
		stime, _ := strconv.ParseInt(fields[12], 10, 64)   // field 15
		starttime, _ := strconv.ParseInt(fields[19], 10, 64) // field 22

		totalTicks := utime + stime

		comm := strings.TrimSpace(statStr[1:idx])

		cmdline, _ := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		cmdStr := strings.ReplaceAll(strings.TrimSpace(string(cmdline)), "\x00", " ")

		procs = append(procs, cpuInfo{
			PID:        pid,
			TotalTicks: totalTicks,
			StartTicks: starttime,
			Comm:       comm,
			Cmdline:    cmdStr,
		})
	}

	sort.Slice(procs, func(i, j int) bool {
		cpuI := float64(procs[i].TotalTicks) / float64(clkTck) / (uptime - float64(procs[i].StartTicks)/float64(clkTck))
		cpuJ := float64(procs[j].TotalTicks) / float64(clkTck) / (uptime - float64(procs[j].StartTicks)/float64(clkTck))
		return cpuI > cpuJ
	})

	if len(procs) > n {
		procs = procs[:n]
	}

	var result []Process
	for _, p := range procs {
		seconds := int(uptime - float64(p.StartTicks)/float64(clkTck))
		cpuPct := float64(p.TotalTicks) / float64(clkTck) / float64(seconds) * 100
		if seconds <= 0 {
			cpuPct = 0
		}

		cmd := strings.TrimSpace(p.Cmdline)
		if cmd == "" {
			cmd = strings.TrimSpace(p.Comm)
		}

		result = append(result, Process{
			PID:     p.PID,
			PCPU:    fmt.Sprintf("%.1f", cpuPct),
			Time:    formatElapsed(seconds),
			Command: cmd,
		})
	}

	return result
}

func readUptime() float64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return 0
	}
	uptime, _ := strconv.ParseFloat(parts[0], 64)
	return uptime
}

func formatElapsed(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm%ds", seconds/60, seconds%60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("%dh%dm", seconds/3600, (seconds%3600)/60)
	}
	return fmt.Sprintf("%dd%dh", seconds/86400, (seconds%86400)/3600)
}
