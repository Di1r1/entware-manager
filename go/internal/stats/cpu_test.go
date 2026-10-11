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

func TestStatComm(t *testing.T) {
	stat := []byte(`123 (a(b) c) S 1 1 1 0 -1 4194560 0 0 0 0 250 50 0 0 20 0 1 0 100 0 0`)
	if got := statComm(stat); got != "a(b) c" {
		t.Errorf("comm = %q, want %q", got, "a(b) c")
	}
	if got := statComm([]byte(`6 (ksoftirqd/0) S 2 0 0 0 -1 2129984 0 0 0 0 10 5 0 0 20 0 1 0 100 0 0`)); got != "ksoftirqd/0" {
		t.Errorf("comm ksoftirqd = %q", got)
	}
	if got := statComm([]byte("битые данные")); got != "" {
		t.Errorf("битый stat: comm = %q, want пусто", got)
	}
}

func TestDisplayCmd(t *testing.T) {
	if got := displayCmd("/opt/bin/syncthing serve", "syncthing"); got != "syncthing serve" {
		t.Errorf("argv0 с пробелом: %q", got)
	}
	if got := displayCmd("/sbin/init", "init"); got != "init" {
		t.Errorf("обычный: %q", got)
	}
	// Пустой cmdline (ядерный поток) — [comm], а не пропуск и не ".".
	if got := displayCmd("", "ksoftirqd/0"); got != "[ksoftirqd/0]" {
		t.Errorf("kthread: %q", got)
	}
	// Base("") даёт "." — ловушка: такой процесс тоже идёт в [comm].
	if got := displayCmd("", "kworker/1:0H"); got != "[kworker/1:0H]" {
		t.Errorf("точка-ловушка: %q", got)
	}
	if got := displayCmd("", ""); got != "" {
		t.Errorf("без имени: %q, want пропуск", got)
	}
}

func TestRemainderRow(t *testing.T) {
	// Итог 75%, топы в сумме 3% (сеть в ядерных потоках): остаток виден.
	row, ok := remainderRow(75, 3)
	if !ok {
		t.Fatal("значимый остаток должен показываться")
	}
	if row.Pcts != "72%" {
		t.Errorf("остаток = %q, want 72%%", row.Pcts)
	}
	// Итог сошёлся с топами — строки нет.
	if _, ok := remainderRow(30, 29.8); ok {
		t.Error("остаток <0.5% показывать нельзя")
	}
	// Перекос в другую сторону (округления) — кламп в ноль, строки нет.
	if _, ok := remainderRow(10, 12); ok {
		t.Error("отрицательный остаток должен гаситься")
	}
	// Малый, но значимый остаток — формат как у топов.
	row, ok = remainderRow(5, 4.2)
	if !ok || row.Pcts != "<1%" {
		t.Errorf("малый остаток: ok=%v pcts=%q, want <1%%", ok, row.Pcts)
	}
}
