package network

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeWatchdogScript(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAction_Start(t *testing.T) {
	tmp := t.TempDir()
	oldScript := WatchdogScript
	oldPidFile := PidFile
	WatchdogScript = filepath.Join(tmp, "watchdog.sh")
	PidFile = filepath.Join(tmp, "watchdog.pid")
	defer func() {
		WatchdogScript = oldScript
		PidFile = oldPidFile
	}()

	writeWatchdogScript(t, WatchdogScript, `#!/bin/sh
echo "$$" > "`+PidFile+`"
`)

	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "action=start")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleAction)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", result["status"])
	}

	if _, err := os.Stat(PidFile); os.IsNotExist(err) {
		t.Error("PID file should exist after start")
	}
}

func TestHandleAction_Stop(t *testing.T) {
	tmp := t.TempDir()
	oldScript := WatchdogScript
	WatchdogScript = filepath.Join(tmp, "watchdog.sh")
	defer func() { WatchdogScript = oldScript }()

	writeWatchdogScript(t, WatchdogScript, `#!/bin/sh
exit 0
`)

	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "action=stop")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleAction)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", result["status"])
	}
}

func TestHandleAction_Restart(t *testing.T) {
	tmp := t.TempDir()
	oldScript := WatchdogScript
	oldPidFile := PidFile
	WatchdogScript = filepath.Join(tmp, "watchdog.sh")
	PidFile = filepath.Join(tmp, "watchdog.pid")
	defer func() {
		WatchdogScript = oldScript
		PidFile = oldPidFile
	}()

	writeWatchdogScript(t, WatchdogScript, `#!/bin/sh
echo "$$" > "`+PidFile+`"
`)

	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "action=restart")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleAction)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", result["status"])
	}
}

func TestHandleAction_UnknownAction(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "action=unknown")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleAction)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["status"] != "error" {
		t.Errorf("expected status 'error', got '%s'", result["status"])
	}
	if result["message"] != "Неизвестное действие: unknown" {
		t.Errorf("expected unknown action message, got '%s'", result["message"])
	}
}

func TestHandleAction_NotGET(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "POST")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleAction)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["error"] != "Method not allowed" {
		t.Errorf("expected 'Method not allowed', got '%s'", result["error"])
	}
}
