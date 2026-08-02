package monitor

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Process struct {
	PID     int    `json:"pid"`
	PCPU    string `json:"pcpu"`
	Time    string `json:"time"`
	Command string `json:"command"`
}

type StatusResponse struct {
	DaemonStatus string    `json:"daemon_status"`
	DaemonPID    string    `json:"daemon_pid"`
	Processes    []Process `json:"processes"`
}

const (
	pidFile = "/tmp/entware/pid/watchdog.pid"
	clkTck  = 100
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
	PID        int
	TotalTicks int64
	StartTicks int64
	Comm       string
	Cmdline    string
}

const cpuSampleInterval = 1000 * time.Millisecond

func getTopProcesses(n int) []Process {
	if readUptime() <= 0 {
		return nil
	}

	readSnapshot := func() (map[int]cpuInfo, int64) {
		entries, err := os.ReadDir("/proc")
		if err != nil {
			return nil, 0
		}

		procs := make(map[int]cpuInfo)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			pid, err := strconv.Atoi(e.Name())
			if err != nil {
				continue
			}
			if pid == os.Getpid() {
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

			utime, _ := strconv.ParseInt(fields[11], 10, 64)     // field 14
			stime, _ := strconv.ParseInt(fields[12], 10, 64)     // field 15
			starttime, _ := strconv.ParseInt(fields[19], 10, 64) // field 22

			cmdline, _ := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
			cmdStr := strings.ReplaceAll(strings.TrimSpace(string(cmdline)), "\x00", " ")

			procs[pid] = cpuInfo{
				PID:        pid,
				TotalTicks: utime + stime,
				StartTicks: starttime,
				Comm:       strings.TrimSpace(statStr[1:idx]),
				Cmdline:    cmdStr,
			}
		}

		total := int64(0)
		if data, err := os.ReadFile("/proc/stat"); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if !strings.HasPrefix(line, "cpu ") {
					continue
				}
				for _, f := range strings.Fields(line)[1:] {
					if v, err := strconv.ParseInt(f, 10, 64); err == nil {
						total += v
					}
				}
				break
			}
		}
		return procs, total
	}

	snap1, total1 := readSnapshot()
	time.Sleep(cpuSampleInterval)
	snap2, total2 := readSnapshot()

	if total2 <= total1 {
		return []Process{}
	}

	var procs []Process
	for pid, p1 := range snap1 {
		p2, ok := snap2[pid]
		if !ok {
			continue
		}

		delta := p2.TotalTicks - p1.TotalTicks
		cpuPct := float64(delta) / float64(total2-total1) * 100
		if cpuPct < 0 {
			cpuPct = 0
		}

		uptime := readUptime()
		seconds := int(uptime - float64(p1.StartTicks)/float64(clkTck))
		if seconds < 0 {
			seconds = 0
		}

		cmd := strings.TrimSpace(p1.Cmdline)
		if cmd == "" {
			cmd = strings.TrimSpace(p1.Comm)
		}

		procs = append(procs, Process{
			PID:     pid,
			PCPU:    fmt.Sprintf("%.1f", cpuPct),
			Time:    formatElapsed(seconds),
			Command: cmd,
		})
	}

	sort.Slice(procs, func(i, j int) bool {
		a, _ := strconv.ParseFloat(procs[i].PCPU, 64)
		b, _ := strconv.ParseFloat(procs[j].PCPU, 64)
		return a > b
	})

	if len(procs) > n {
		procs = procs[:n]
	}

	return procs
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
