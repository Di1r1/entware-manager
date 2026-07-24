package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWatchdogConfig_GET_Default(t *testing.T) {
	tmp := t.TempDir()
	oldCfg := wrapperConfig
	wrapperConfig = filepath.Join(tmp, "noexist.json")
	defer func() { wrapperConfig = oldCfg }()

	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleWatchdogConfig)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", result["enabled"])
	}
}

func TestWatchdogConfig_GET_Existing(t *testing.T) {
	tmp := t.TempDir()
	oldCfg := wrapperConfig
	wrapperConfig = filepath.Join(tmp, "config.json")
	defer func() { wrapperConfig = oldCfg }()

	expected := `{"enabled":false,"interval":30}`
	if err := os.WriteFile(wrapperConfig, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleWatchdogConfig)

	if !strings.Contains(string(body), "enabled") {
		t.Errorf("expected config content in output, got: %s", string(body))
	}
}

func TestWatchdogConfig_POST_Valid(t *testing.T) {
	tmp := t.TempDir()
	oldCfg := wrapperConfig
	wrapperConfig = filepath.Join(tmp, "config.json")
	defer func() { wrapperConfig = oldCfg }()

	setStdin(t, `{"enabled":false,"interval":60}`)

	os.Setenv("REQUEST_METHOD", "POST")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleWatchdogConfig)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", result["status"])
	}

	saved, err := os.ReadFile(wrapperConfig)
	if err != nil {
		t.Fatal("config file was not written")
	}
	if !strings.Contains(string(saved), "enabled") {
		t.Errorf("expected valid JSON in config, got: %s", string(saved))
	}
}

func TestWatchdogConfig_POST_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	oldCfg := wrapperConfig
	wrapperConfig = filepath.Join(tmp, "config.json")
	defer func() { wrapperConfig = oldCfg }()

	setStdin(t, `not json`)

	os.Setenv("REQUEST_METHOD", "POST")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleWatchdogConfig)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["status"] != "error" {
		t.Errorf("expected status 'error', got '%s'", result["status"])
	}
}

func TestWatchdogConfig_MethodNotAllowed(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "PUT")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleWatchdogConfig)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["error"] != "Method not allowed" {
		t.Errorf("expected 'Method not allowed', got '%s'", result["error"])
	}
}
