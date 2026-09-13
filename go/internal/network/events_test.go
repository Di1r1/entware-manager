package network

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"entware-manager/internal/packages"
)

func writeLogFile(t *testing.T, dir string, lines []string) {
	t.Helper()
	logFile := filepath.Join(dir, time.Now().Format("2006-01-02")+".log")
	data := []byte(stringsJoin(lines, "\n"))
	if err := os.WriteFile(logFile, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func stringsJoin(elems []string, sep string) string {
	var buf bytes.Buffer
	for i, e := range elems {
		if i > 0 {
			buf.WriteString(sep)
		}
		buf.WriteString(e)
	}
	return buf.String()
}

func TestParseLogFile_ReturnsEvents(t *testing.T) {
	tmp := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmp
	defer func() { LogDir = oldLogDir }()

	writeLogFile(t, tmp, []string{
		`[2026-04-02 10:00:00] [INFO] [192.168.1.1] [1234] [network] dhcp renew (eth0)`,
		`[2026-04-02 10:01:00] [WARN] [192.168.1.1] [1234] [other] ping timeout`,
		`[2026-04-02 10:02:00] [ERROR] [192.168.1.1] [1234] [network] eth0 interface_down (was UP)`,
	})

	events := parseLogFile("network", 20)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].Event != "interface_down" {
		t.Errorf("expected first event 'interface_down' (newest), got '%s'", events[0].Event)
	}
	if events[1].Event != "renew" {
		t.Errorf("expected second event 'renew', got '%s'", events[1].Event)
	}
	if events[0].Level != "ERROR" {
		t.Errorf("expected level ERROR, got '%s'", events[0].Level)
	}
}

// tag=monitor работает тем же generic-фильтром [tag], что и network:
// [monitor]-строки лежат в том же дневном логе (п. 3.1 запроса NOC).
func TestParseLogFile_MonitorTag(t *testing.T) {
	tmp := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmp
	defer func() { LogDir = oldLogDir }()

	writeLogFile(t, tmp, []string{
		`[2026-04-02 10:00:00] [INFO] [1.2.3.4] [1234] [network] dhcp renew (eth0)`,
		`[2026-04-02 10:01:00] [INFO] [1.2.3.4] [1234] [monitor] watchdog started`,
		`[2026-04-02 10:02:00] [WARN] [1.2.3.4] [1234] [monitor] high temp (58)`,
	})

	events := parseLogFile("monitor", 20)
	if len(events) != 2 {
		t.Fatalf("expected 2 monitor events, got %d", len(events))
	}
	if events[0].Level != "WARN" {
		t.Errorf("expected newest level WARN, got '%s'", events[0].Level)
	}
}

func TestParseLogFile_RespectsLimit(t *testing.T) {
	tmp := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmp
	defer func() { LogDir = oldLogDir }()

	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, `[2026-04-02 10:00:00] [INFO] [1.2.3.4] [1234] [network] test event (`+itoa(i)+`)`)
	}
	writeLogFile(t, tmp, lines)

	events := parseLogFile("network", 3)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestParseLogFile_EmptyLog(t *testing.T) {
	tmp := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmp
	defer func() { LogDir = oldLogDir }()

	writeLogFile(t, tmp, []string{})
	events := parseLogFile("network", 20)
	if events == nil || len(events) != 0 {
		t.Fatalf("expected empty events, got %d", len(events))
	}
}

func TestParseLogFile_NoSuchFile(t *testing.T) {
	tmp := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmp
	defer func() { LogDir = oldLogDir }()

	events := parseLogFile("network", 20)
	if events != nil {
		t.Fatal("expected nil for missing file")
	}
}

func TestParseLogFile_ParsesTimestamp(t *testing.T) {
	tmp := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmp
	defer func() { LogDir = oldLogDir }()

	writeLogFile(t, tmp, []string{
		`[2026-04-02 10:00:00] [INFO] [1.2.3.4] [1234] [network] svc evt (details)`,
	})

	events := parseLogFile("network", 20)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Timestamp != "2026-04-02 10:00:00" {
		t.Errorf("expected timestamp '2026-04-02 10:00:00', got '%s'", events[0].Timestamp)
	}
}

func TestHandleEvents_OutputsJSON(t *testing.T) {
	tmp := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmp
	defer func() { LogDir = oldLogDir }()

	writeLogFile(t, tmp, []string{
		`[2026-04-02 10:00:00] [INFO] [1.2.3.4] [1234] [network] dhcp renew (eth0)`,
	})

	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "limit=10")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleEvents)

	var result map[string]json.RawMessage
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}

	events, ok := result["events"]
	if !ok {
		t.Fatal("missing 'events' key")
	}
	var evts []LogEvent
	if err := json.Unmarshal(events, &evts); err != nil {
		t.Fatalf("failed to parse events: %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evts))
	}
	if evts[0].Event != "renew" {
		t.Errorf("expected event 'renew', got '%s'", evts[0].Event)
	}
}

// tag=all (карточка «События» NOC): Service извлекается из тега строки,
// суффикс .cgi нормализуется (login.cgi → login) — иначе фильтры
// Сеть/Службы/Входы не могут совпасть (Service был "").
func TestParseGenericLine_SetsService(t *testing.T) {
	cases := []struct{ line, want string }{
		{`[2026-04-02 10:00:00] [INFO] [127.0.0.1] [1234] [network] eth0 interface_up`, "network"},
		{`[2026-04-02 10:00:00] [INFO] [127.0.0.1] [1234] [service] lighttpd started`, "service"},
		{`[2026-04-02 10:00:00] [INFO] [127.0.0.1] [1234] [monitor] watchdog started`, "monitor"},
		{`[2026-04-02 10:00:00] [WARN] [192.168.1.1] [1234] [login.cgi] Неверный пароль при входе`, "login"},
		{`[2026-04-02 10:00:00] [INFO] [127.0.0.1] [1234] [crontab_update.cgi] cron updated`, "crontab_update"},
		{`[2026-04-02 10:00:00] [INFO] [127.0.0.1] [1234] [packages] Пакет curl: установлен`, "packages"},
		{`[2026-04-02 10:00:00] [INFO] [localhost] [1234] [network] wifi connected`, "network"},
	}
	for _, c := range cases {
		ev := parseGenericLine(c.line, "2026-04-02 10:00:00")
		if ev.Service != c.want {
			t.Errorf("line %q: expected service %q, got %q", c.line, c.want, ev.Service)
		}
		if ev.Event == "" {
			t.Errorf("line %q: expected non-empty event text", c.line)
		}
	}
}

// Запрос tag=login находит строки [login.cgi] (нормализация с обеих сторон).
func TestParseLogFile_LoginTag(t *testing.T) {
	tmp := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmp
	defer func() { LogDir = oldLogDir }()

	writeLogFile(t, tmp, []string{
		`[2026-04-02 10:00:00] [INFO] [127.0.0.1] [1234] [network] dhcp renew (eth0)`,
		`[2026-04-02 10:01:00] [WARN] [192.168.1.1] [1234] [login.cgi] Неверный пароль при входе`,
		`[2026-04-02 10:02:00] [INFO] [192.168.1.1] [1234] [login.cgi] Успешный вход`,
	})

	events := parseLogFile("login", 20)
	if len(events) != 2 {
		t.Fatalf("expected 2 login events, got %d", len(events))
	}
}

// tag=network по-прежнему находит только [network] (классика без изменений).
func TestParseLogFile_NetworkTagUnchanged(t *testing.T) {
	tmp := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmp
	defer func() { LogDir = oldLogDir }()

	writeLogFile(t, tmp, []string{
		`[2026-04-02 10:00:00] [INFO] [127.0.0.1] [1234] [network] dhcp renew (eth0)`,
		`[2026-04-02 10:01:00] [WARN] [192.168.1.1] [1234] [login.cgi] Неверный пароль при входе`,
		`[2026-04-02 10:02:00] [INFO] [127.0.0.1] [1234] [service] lighttpd started`,
	})

	events := parseLogFile("network", 20)
	if len(events) != 1 {
		t.Fatalf("expected 1 network event, got %d", len(events))
	}
	if events[0].Service != "dhcp" || events[0].Event != "renew" {
		t.Errorf("classic network parsing changed: %+v", events[0])
	}
}

func itoa(i int) string {
	return string(rune('0' + i))
}

// tag=packages (п. 3.1/2.2 запроса NOC, вариант Б): читает package_changes.log,
// Service всегда "packages", success → INFO, прочее → WARN, новые сверху.
func TestCollectEvents_PackagesTag(t *testing.T) {
	dir := t.TempDir()
	old := packages.PackageChangesLog
	packages.PackageChangesLog = filepath.Join(dir, "package_changes.log")
	defer func() { packages.PackageChangesLog = old }()

	data := "2026-08-28 19:44:20 | remove | 7-zip | success\n" +
		"2026-08-28 19:46:12 | upgrade | curl | failed\n"
	if err := os.WriteFile(packages.PackageChangesLog, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	events := CollectEvents("packages", 20)
	if len(events) != 2 {
		t.Fatalf("expected 2 package events, got %d", len(events))
	}
	if events[0].Service != "packages" || events[0].Event != "upgrade" || events[0].Details != "curl" {
		t.Errorf("newest event broken: %+v", events[0])
	}
	if events[0].Level != "WARN" {
		t.Errorf("expected WARN for failed, got '%s'", events[0].Level)
	}
	if events[1].Level != "INFO" || events[1].Event != "remove" {
		t.Errorf("second event broken: %+v", events[1])
	}
}
