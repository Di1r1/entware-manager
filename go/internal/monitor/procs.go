package monitor

import (
	"entware-manager/internal/cgiutil"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ProcsEntry — одна строка топа (контракт NOC: как shell procs.cgi).
type ProcsEntry struct {
	PID     int    `json:"pid"`
	PCPU    string `json:"pcpu"`
	Time    string `json:"time"`
	Command string `json:"command"`
	RssKB   int64  `json:"rss_kb"`
}

// ProcsCPU — разбивка CPU за окно замера, в процентах.
type ProcsCPU struct {
	User   float64 `json:"user"`
	System float64 `json:"system"`
	Iowait float64 `json:"iowait"`
	Idle   float64 `json:"idle"`
}

// ProcsResponse — полный слепок процессов для NOC (все процессы, не топ-N).
type ProcsResponse struct {
	CPUCount  int64        `json:"cpu_count"`
	Loadavg   string       `json:"loadavg"`
	MemTotal  int64        `json:"mem_total_kb"`
	MemAvail  int64        `json:"mem_avail_kb"`
	MemSwap   int64        `json:"mem_swap_kb"`
	CPU       ProcsCPU     `json:"cpu"`
	Processes []ProcsEntry `json:"processes"`
}

func HandleProcs() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	cgiutil.WriteJSON(CollectProcs())
}

// CollectProcs — чистое значение для тестов: полный слепок за окно 1с.
func CollectProcs() ProcsResponse {
	resp := ProcsResponse{
		CPUCount: readCPUCount(),
		Loadavg:  readLoadavg(),
	}

	resp.MemTotal = readMeminfo("MemTotal:")
	resp.MemAvail = readMeminfo("MemAvailable:")
	swapTotal := readMeminfo("SwapTotal:")
	swapFree := readMeminfo("SwapFree:")
	if swapFree > swapTotal {
		swapFree = swapTotal
	}
	resp.MemSwap = swapTotal - swapFree

	snap1, total1 := snapshotProcs()
	cpu1 := readCPUFields()
	time.Sleep(cpuSampleInterval)
	cpu2 := readCPUFields()
	snap2, total2 := snapshotProcs()

	dt := total2 - total1
	if dt <= 0 {
		dt = 1
	}
	resp.CPU = cpuBreakdown(cpu1, cpu2, dt)

	var procs []ProcsEntry
	uptime := readUptime()
	for pid, p1 := range snap1 {
		p2, ok := snap2[pid]
		if !ok {
			continue
		}
		delta := p2.TotalTicks - p1.TotalTicks
		if delta < 0 {
			delta = 0
		}
		seconds := int(uptime - float64(p1.StartTicks)/float64(clkTck))
		if seconds < 0 {
			seconds = 0
		}
		cmd := strings.TrimSpace(p1.Cmdline)
		if cmd == "" {
			cmd = "[" + strings.TrimSpace(p1.Comm) + "]"
		}
		if cmd == "[]" {
			cmd = fmt.Sprintf("[pid %d]", pid)
		}
		procs = append(procs, ProcsEntry{
			PID:     pid,
			PCPU:    fmt.Sprintf("%.1f", float64(delta)*100/float64(dt)),
			Time:    formatElapsed(seconds),
			Command: cmd,
			RssKB:   p2.RssKB,
		})
	}
	sort.Slice(procs, func(i, j int) bool { return procs[i].PID < procs[j].PID })
	if procs == nil {
		procs = []ProcsEntry{}
	}
	resp.Processes = procs

	return resp
}

// readCPUFields — поля строки 'cpu ' из /proc/stat (user.. guest).
func readCPUFields() []int64 {
	if data, err := os.ReadFile("/proc/stat"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if !strings.HasPrefix(line, "cpu ") {
				continue
			}
			var out []int64
			for _, f := range strings.Fields(line)[1:] {
				v, err := strconv.ParseInt(f, 10, 64)
				if err != nil {
					break
				}
				out = append(out, v)
			}
			return out
		}
	}
	return nil
}

// cpuBreakdown — user(+nice), system, iowait, idle в процентах за окно dt.
// Индексы как в shell procs.cgi: user=0+1, system=2, idle=3, iowait=4.
func cpuBreakdown(f1, f2 []int64, dt int64) ProcsCPU {
	zero := ProcsCPU{Idle: 100}
	if len(f1) < 5 || len(f2) < 5 {
		return zero
	}
	at := func(f []int64, i int) int64 {
		if i < len(f) {
			return f[i]
		}
		return 0
	}
	pct := func(d int64) float64 {
		v := float64(d) * 100 / float64(dt)
		rounded, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", v), 64)
		return rounded
	}
	return ProcsCPU{
		User:   pct((at(f2, 0) + at(f2, 1)) - (at(f1, 0) + at(f1, 1))),
		System: pct(at(f2, 2) - at(f1, 2)),
		Iowait: pct(at(f2, 4) - at(f1, 4)),
		Idle:   pct(at(f2, 3) - at(f1, 3)),
	}
}

func readCPUCount() int64 {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0
	}
	var n int64
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "processor") {
			n++
		}
	}
	return n
}

func readLoadavg() string {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(data))
	if len(fields) > 3 {
		fields = fields[:3]
	}
	return strings.Join(fields, " ")
}

// readMeminfo — значение поля /proc/meminfo в КБ (0 если нет).
func readMeminfo(key string) int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, key) {
			continue
		}
		for _, f := range strings.Fields(line)[1:] {
			if v, err := strconv.ParseInt(f, 10, 64); err == nil {
				return v
			}
		}
	}
	return 0
}
