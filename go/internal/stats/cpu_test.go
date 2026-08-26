// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package stats

import "testing"

func TestParseCPUTotal(t *testing.T) {
	line := "cpu  100 0 50 800 20 0 0 0 0 0"
	total, idle := parseCPUTotal(line)
	// total = 100+0+50+800+20 = 970; idle = 800+20
	if total != 970 {
		t.Errorf("total = %d, want 970", total)
	}
	if idle != 820 {
		t.Errorf("idle = %d, want 820", idle)
	}
	if t2, i2 := parseCPUTotal("cpu0 1 2 3"); t2 != 0 || i2 != 0 {
		t.Errorf("не-cpu строка должна давать нули: %d/%d", t2, i2)
	}
}

func TestFormatLoadavg(t *testing.T) {
	got := formatLoadavg("0.42 0.38 0.35 2/387 12345")
	if got != "0.42 / 0.38 / 0.35" {
		t.Errorf("loadavg = %q", got)
	}
	if got := formatLoadavg("мусор"); got != "н/д" {
		t.Errorf("короткая строка: %q, want н/д", got)
	}
}

// Скобки в имени процесса: парсер stat должен брать поля после последней ')'.
func TestParseProcStatTicks(t *testing.T) {
	stat := []byte(`123 (a(b) c) S 1 1 1 0 -1 4194560 0 0 0 0 250 50 0 0 20 0 1 0 100 0 0`)
	if got := parseProcStatTicks(stat); got != 300 {
		t.Errorf("ticks = %d, want 300 (utime 250 + stime 50)", got)
	}
	if got := parseProcStatTicks([]byte("битые данные")); got != 0 {
		t.Errorf("битый stat должен давать 0, got %d", got)
	}
}
