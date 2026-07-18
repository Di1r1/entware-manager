package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Service struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Enabled bool     `json:"enabled"`
	PID     string   `json:"pid"`
	PIDs    []string `json:"pids"`
}

const servicesDir = "/opt/etc/init.d"

func HandleServices() {
	if !IsGET() {
		NotAllowed()
		return
	}

	// One-time scan of /proc for all processes
	procMap := scanProc()

	var list []Service

	patterns := []string{servicesDir + "/S*", servicesDir + "/K*"}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		sort.Strings(matches)

		for _, script := range matches {
			fi, err := os.Stat(script)
			if err != nil || fi.IsDir() {
				continue
			}

			fullname := filepath.Base(script)
			name := ""
			enabled := false

			switch {
			case strings.HasPrefix(fullname, "S"):
				name = fullname[1:]
				enabled = true
			case strings.HasPrefix(fullname, "K"):
				name = fullname[1:]
				enabled = false
			default:
				continue
			}

			pids := findPIDs(script, fullname, name, procMap)
			svc := Service{
				Name:    name,
				Enabled: enabled,
			}

			if len(pids) > 0 {
				svc.Status = "running"
				svc.PID = pids[0]
				svc.PIDs = pids
			} else {
				svc.Status = "stopped"
				svc.PID = ""
				svc.PIDs = []string{}
			}

			list = append(list, svc)
		}
	}

	if list == nil {
		list = []Service{}
	}
	WriteJSON(list)
}

type procInfo struct {
	PID     int
	Cmdline string
	State   string
}

func scanProc() map[int]procInfo {
	procMap := make(map[int]procInfo)

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return procMap
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}

		cmdline, _ := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		cmdStr := strings.TrimSpace(string(cmdline))
		cmdStr = strings.ReplaceAll(cmdStr, "\x00", " ")

		statusData, _ := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
		state := ""
		for _, line := range strings.Split(string(statusData), "\n") {
			if strings.HasPrefix(line, "State:") {
				state = strings.TrimSpace(line[6:])
				break
			}
		}

		procMap[pid] = procInfo{
			PID:     pid,
			Cmdline: cmdStr,
			State:   state,
		}
	}

	return procMap
}

func findPIDs(scriptPath, fullname, name string, procMap map[int]procInfo) []string {
	baseName := name
	for len(baseName) > 0 && baseName[0] >= '0' && baseName[0] <= '9' {
		baseName = baseName[1:]
	}

	// 1. PIDFILE from script
	pidfile := getVar(scriptPath, "PIDFILE")
	if pidfile != "" {
		data, err := os.ReadFile(pidfile)
		if err == nil {
			pidStr := strings.TrimSpace(string(data))
			pid, err := strconv.Atoi(pidStr)
			if err == nil {
				if pi, ok := procMap[pid]; ok && !strings.HasPrefix(pi.State, "Z") {
					return []string{pidStr}
				}
			}
		}
	}

	// 2. Standard pid files
	for _, pf := range []string{
		fmt.Sprintf("/tmp/%s.pid", baseName),
		fmt.Sprintf("/var/run/%s.pid", baseName),
		fmt.Sprintf("/opt/var/run/%s.pid", baseName),
		fmt.Sprintf("/tmp/%s.pid", fullname),
		fmt.Sprintf("/var/run/%s.pid", fullname),
	} {
		data, err := os.ReadFile(pf)
		if err == nil {
			pidStr := strings.TrimSpace(string(data))
			pid, err := strconv.Atoi(pidStr)
			if err == nil {
				if pi, ok := procMap[pid]; ok && !strings.HasPrefix(pi.State, "Z") {
					return []string{pidStr}
				}
			}
		}
	}

	// 3. PROCS / NAME / DAEMON from script
	procName := getVar(scriptPath, "PROCS")
	if procName == "" {
		procName = getVar(scriptPath, "NAME")
	}
	if procName == "" {
		procName = getVar(scriptPath, "DAEMON")
	}
	if procName != "" {
		if pids := matchByCmdline(procName, procMap); len(pids) > 0 {
			return pids
		}
	}

	// 4. By base name
	if pids := matchByCmdline(baseName, procMap); len(pids) > 0 {
		return pids
	}

	// 5. By full name (S99name)
	if pids := matchByCmdline(fullname, procMap); len(pids) > 0 {
		return pids
	}

	// 6. By .py file from SCRIPT
	scriptPathVar := getVar(scriptPath, "SCRIPT")
	if scriptPathVar != "" {
		scriptBase := filepath.Base(scriptPathVar)
		if pids := matchByCmdline(scriptBase, procMap); len(pids) > 0 {
			return pids
		}
	}

	return nil
}

func matchByCmdline(pattern string, procMap map[int]procInfo) []string {
	patternLower := strings.ToLower(pattern)
	var pids []string

	// Collect all matching PIDs
	for _, pi := range procMap {
		if strings.HasPrefix(pi.State, "Z") {
			continue
		}
		cmdLower := strings.ToLower(pi.Cmdline)
		if strings.Contains(cmdLower, patternLower) {
			pids = append(pids, strconv.Itoa(pi.PID))
		}
	}

	// Sort pids for deterministic output
	if len(pids) > 0 {
		sort.Slice(pids, func(i, j int) bool {
			ai, _ := strconv.Atoi(pids[i])
			aj, _ := strconv.Atoi(pids[j])
			return ai < aj
		})
		return pids
	}

	return nil
}

func getVar(scriptPath, varName string) string {
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return ""
	}

	prefix := varName + "="
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			val := line[len(prefix):]
			val = strings.Trim(val, "\"'")
			return val
		}
	}
	return ""
}
