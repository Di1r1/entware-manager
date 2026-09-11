package monitor

import (
	"encoding/json"
	"strings"
	"testing"
)

// CollectProcs: контракт shell procs.cgi — полный слепок за окно 1с.
func TestCollectProcsShape(t *testing.T) {
	resp := CollectProcs()

	if len(resp.Processes) == 0 {
		t.Fatal("expected non-empty process list")
	}
	p := resp.Processes[0]
	if p.PID <= 0 {
		t.Errorf("bad pid: %+v", p)
	}
	if p.Command == "" || p.Time == "" || p.PCPU == "" {
		t.Errorf("empty fields: %+v", p)
	}
	if resp.Loadavg == "" {
		t.Error("empty loadavg")
	}
	if resp.MemTotal <= 0 {
		t.Error("bad mem_total_kb")
	}

	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, k := range []string{
		`"cpu_count"`, `"loadavg"`, `"mem_total_kb"`, `"mem_avail_kb"`,
		`"mem_swap_kb"`, `"cpu"`, `"processes"`,
		`"pid"`, `"pcpu"`, `"time"`, `"command"`, `"rss_kb"`,
		`"user"`, `"system"`, `"iowait"`, `"idle"`,
	} {
		if !strings.Contains(string(out), k) {
			t.Errorf("missing JSON key %s", k)
		}
	}
}

// cpuBreakdown: проценты за окно, сумма user+system+iowait+idle ≈ 100.
func TestCPUBreakdown(t *testing.T) {
	f1 := []int64{100, 10, 50, 800, 40}
	f2 := []int64{110, 12, 55, 880, 43}
	cpu := cpuBreakdown(f1, f2, 100)
	sum := cpu.User + cpu.System + cpu.Iowait + cpu.Idle
	if sum < 99 || sum > 101 {
		t.Errorf("breakdown sums to %.1f, want ~100: %+v", sum, cpu)
	}
	if cpu.User != 12 || cpu.System != 5 || cpu.Iowait != 3 || cpu.Idle != 80 {
		t.Errorf("wrong split: %+v", cpu)
	}
	if got := cpuBreakdown(nil, f2, 100); got.Idle != 100 {
		t.Errorf("short fields must give idle=100: %+v", got)
	}
}
