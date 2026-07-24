package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWatchdogStatus_Stopped(t *testing.T) {
	tmp := t.TempDir()
	oldPid := wrapperPidFile
	wrapperPidFile = filepath.Join(tmp, "nonexistent.pid")
	defer func() { wrapperPidFile = oldPid }()

	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleWatchdogStatus)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["running"] != false {
		t.Errorf("expected running=false, got %v", result["running"])
	}
}

func TestWatchdogStatus_GETOnly(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "POST")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleWatchdogStatus)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["error"] != "Method not allowed" {
		t.Errorf("expected 'Method not allowed', got '%s'", result["error"])
	}
}

func TestWatchdogStatus_ReturnsDefaultConfig(t *testing.T) {
	tmp := t.TempDir()
	oldCfg := wrapperConfig
	wrapperConfig = filepath.Join(tmp, "noexist.json")
	defer func() { wrapperConfig = oldCfg }()

	oldPid := wrapperPidFile
	wrapperPidFile = filepath.Join(tmp, "noexist.pid")
	defer func() { wrapperPidFile = oldPid }()

	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleWatchdogStatus)

	var result map[string]json.RawMessage
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	cfg, ok := result["config"]
	if !ok || len(cfg) == 0 {
		t.Fatal("missing 'config' key")
	}
	var cfgObj map[string]interface{}
	if err := json.Unmarshal(cfg, &cfgObj); err != nil {
		t.Fatal("config is not valid JSON")
	}
	if cfgObj["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", cfgObj["enabled"])
	}
}
