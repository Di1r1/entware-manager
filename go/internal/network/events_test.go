package network

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func itoa(i int) string {
	return string(rune('0' + i))
}
