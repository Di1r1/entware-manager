package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeScript(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestWatchdogAction_Start(t *testing.T) {
	tmp := t.TempDir()
	oldScript := watchdogScript
	oldPid := wrapperPidFile
	watchdogScript = filepath.Join(tmp, "watchdog.sh")
	wrapperPidFile = filepath.Join(tmp, "watchdog.pid")
	defer func() {
		watchdogScript = oldScript
		wrapperPidFile = oldPid
	}()

	writeScript(t, watchdogScript, `#!/bin/sh
echo "$$" > "`+wrapperPidFile+`"
`)

	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "action=start")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleWatchdogAction)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", result["status"])
	}
}

func TestWatchdogAction_Stop(t *testing.T) {
	tmp := t.TempDir()
	oldScript := watchdogScript
	watchdogScript = filepath.Join(tmp, "watchdog.sh")
	defer func() { watchdogScript = oldScript }()

	writeScript(t, watchdogScript, `#!/bin/sh
exit 0
`)

	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "action=stop")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleWatchdogAction)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", result["status"])
	}
}

func TestWatchdogAction_UnknownAction(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "action=unknown")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleWatchdogAction)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["message"] != "Неизвестное действие: unknown" {
		t.Errorf("expected unknown action message, got '%s'", result["message"])
	}
}

func TestWatchdogAction_ScriptNotFound(t *testing.T) {
	tmp := t.TempDir()
	oldScript := watchdogScript
	watchdogScript = filepath.Join(tmp, "nonexistent.sh")
	defer func() { watchdogScript = oldScript }()

	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "action=start")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	body := captureStdout(t, HandleWatchdogAction)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["status"] != "error" {
		t.Errorf("expected status 'error', got '%s'", result["status"])
	}
	if result["message"] != "Демон не найден: " + watchdogScript {
		t.Errorf("expected 'not found' message, got '%s'", result["message"])
	}
}
