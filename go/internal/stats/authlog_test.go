// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseAuthLine(t *testing.T) {
	cases := []struct {
		line    string
		ok      bool
		ip, lvl string
	}{
		{"[2026-08-23 15:04:05] [WARN] [192.168.3.9] [1234] [login.cgi] Неверный пароль при входе", true, "192.168.3.9", "WARN"},
		// проверка чистоты message (хвост "login.cgi]" не должен оставаться)
		{"[2026-08-23 15:04:06] [INFO] [10.0.0.2] [1235] [login.cgi] Успешный вход", true, "10.0.0.2", "INFO"},
		{"[2026-08-23 15:04:07] [WARN] [fe80::1] [1236] [monitor] что-то другое", false, "", ""},
		{"мусорная строка без формата", false, "", ""},
	}
	wantMsg := map[int]string{0: "Неверный пароль при входе", 1: "Успешный вход"}
	for i, c := range cases {
		e, ok := parseAuthLine(c.line)
		if ok != c.ok {
			t.Errorf("parseAuthLine(%q) ok=%v, want %v", c.line, ok, c.ok)
			continue
		}
		if msg, exists := wantMsg[i]; ok && exists && e.Message != msg {
			t.Errorf("message = %q, want %q", e.Message, msg)
		}
		if ok && strings.Contains(e.Message, "login.cgi") {
			t.Errorf("message содержит хвост тега: %q", e.Message)
		}
		if ok && e.IP != c.ip {
			t.Errorf("ip = %q, want %q", e.IP, c.ip)
		}
		if ok && e.Level != c.lvl {
			t.Errorf("level = %q, want %q", e.Level, c.lvl)
		}
	}
}

func TestHandleAuthLog(t *testing.T) {
	dir := t.TempDir()
	logsDir = dir
	now := time.Now()
	today := now.Format("2006-01-02")
	ts := func(offset time.Duration) string {
		return now.Add(offset).Format("2006-01-02 15:04:05")
	}
	content := "[" + ts(-90*time.Minute) + "] [INFO] [192.168.3.5] [100] [login.cgi] Успешный вход\n" +
		"[" + ts(-30*time.Minute) + "] [WARN] [203.0.113.7] [101] [login.cgi] Неверный пароль при входе\n" +
		"[" + ts(-29*time.Minute) + "] [WARN] [other-daemon] событие не про логин\n"
	if err := os.WriteFile(filepath.Join(dir, today+".log"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// прямой тест разбора файла (компоненты HandleAuthLog)
	data, err := os.ReadFile(filepath.Join(dir, today+".log"))
	if err != nil {
		t.Fatal(err)
	}
	var entries []authLogEntry
	failed := 0
	dayAgo := now.Add(-24 * time.Hour)
	for _, ln := range splitLines(string(data)) {
		e, ok := parseAuthLine(ln)
		if !ok {
			continue
		}
		if e.Level == "WARN" && isWithin24h(e.Time, dayAgo, now) {
			failed++
		}
		entries = append(entries, e)
	}
	if len(entries) != 2 {
		t.Fatalf("записей = %d, хочу 2", len(entries))
	}
	if failed != 1 {
		t.Fatalf("failed_24h = %d, хочу 1", failed)
	}
	if entries[1].IP != "203.0.113.7" {
		t.Errorf("IP внешнего посетителя потерян: %q", entries[1].IP)
	}
	out, _ := json.Marshal(entries[0])
	if out == nil {
		t.Error("JSON marshal failed")
	}
}

func TestCollectAuthEntriesOrder(t *testing.T) {
	dir := t.TempDir()
	logsDir = dir
	defer func() { logsDir = "/tmp/entware/logs" }()
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	yContent := "[" + yesterday.Format("2006-01-02") + " 20:00:00] [WARN] [10.0.0.1] [1] [login.cgi] Неверный пароль при входе\n"
	tContent := "[" + now.Format("2006-01-02") + " 08:00:00] [INFO] [10.0.0.2] [2] [login.cgi] Успешный вход\n" +
		"[" + now.Format("2006-01-02") + " 09:00:00] [INFO] [10.0.0.3] [3] [login.cgi] Успешный вход\n"
	if err := os.WriteFile(filepath.Join(dir, yesterday.Format("2006-01-02")+".log"), []byte(yContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, now.Format("2006-01-02")+".log"), []byte(tContent), 0644); err != nil {
		t.Fatal(err)
	}
	entries, _ := collectAuthEntries(now)
	if len(entries) != 3 {
		t.Fatalf("записей = %d, хочу 3", len(entries))
	}
	// свежие сверху: сегодня 09:00, сегодня 08:00, вчера 20:00
	want := []string{"10.0.0.3", "10.0.0.2", "10.0.0.1"}
	for i, w := range want {
		if entries[i].IP != w {
			t.Errorf("entries[%d].IP = %q, хочу %q", i, entries[i].IP, w)
		}
	}
}

func TestIsWithin24h(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.Local)
	dayAgo := now.Add(-24 * time.Hour)
	if !isWithin24h("2026-08-23 11:00:00", dayAgo, now) {
		t.Error("свежая запись должна попадать в окно")
	}
	if isWithin24h("2026-08-22 11:00:00", dayAgo, now) {
		t.Error("запись старше 24ч не должна попадать")
	}
	if isWithin24h("битое время", dayAgo, now) {
		t.Error("битое время → false")
	}
}

func splitLines(s string) []string {
	var res []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			res = append(res, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		res = append(res, s[start:])
	}
	return res
}
