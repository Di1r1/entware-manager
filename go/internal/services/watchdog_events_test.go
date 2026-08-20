package services

import (
	"encoding/json"
	"os"
	"testing"
)

func TestWatchdogEvents_Empty(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "limit=5")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleWatchdogEvents)

	var result map[string]json.RawMessage
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if _, ok := result["events"]; !ok {
		t.Fatal("missing 'events' key")
	}
}

func TestWatchdogEvents_GETOnly(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "POST")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleWatchdogEvents)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["error"] != "Method not allowed" {
		t.Errorf("expected 'Method not allowed', got '%s'", result["error"])
	}
}

func TestParseWatchdogLine(t *testing.T) {
	line := `[2026-04-02 10:00:00] [INFO] [192.168.1.1] [1234] [service] dhcp renew (eth0)`
	evt := parseWatchdogLine(line)

	if evt.Timestamp != "2026-04-02 10:00:00" {
		t.Errorf("expected timestamp '2026-04-02 10:00:00', got '%s'", evt.Timestamp)
	}
	if evt.Level != "INFO" {
		t.Errorf("expected level 'INFO', got '%s'", evt.Level)
	}
	if evt.Service != "dhcp" {
		t.Errorf("expected service 'dhcp', got '%s'", evt.Service)
	}
	if evt.Event != "renew" {
		t.Errorf("expected event 'renew', got '%s'", evt.Event)
	}
	if evt.Details != "eth0" {
		t.Errorf("expected details 'eth0', got '%s'", evt.Details)
	}
}

func TestParseWatchdogLine_WarnLevel(t *testing.T) {
	line := `[2026-04-02 10:01:00] [WARN] [1.2.3.4] [5678] [service] lighttpd high_memory (120MB)`
	evt := parseWatchdogLine(line)

	if evt.Level != "WARN" {
		t.Errorf("expected level 'WARN', got '%s'", evt.Level)
	}
	if evt.Service != "lighttpd" {
		t.Errorf("expected service 'lighttpd', got '%s'", evt.Service)
	}
	if evt.Details != "120MB" {
		t.Errorf("expected details '120MB', got '%s'", evt.Details)
	}
}

func TestParseWatchdogLine_ServiceAction(t *testing.T) {
	line := `[2026-04-02 10:02:00] [INFO] [1.2.3.4] [123] [service] Служба ttyd: запущена`
	evt := parseWatchdogLine(line)

	if evt.Timestamp != "2026-04-02 10:02:00" {
		t.Errorf("expected timestamp, got '%s'", evt.Timestamp)
	}
	if evt.Level != "INFO" {
		t.Errorf("expected level INFO, got '%s'", evt.Level)
	}
	if evt.Service != "Служба" {
		t.Errorf("expected service 'Служба', got '%s'", evt.Service)
	}
	if evt.Event != "ttyd:" {
		t.Errorf("expected event 'ttyd:', got '%s'", evt.Event)
	}
}
