package network

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleConfig_GET_ReturnsDefaultWhenNoFile(t *testing.T) {
	tmp := t.TempDir()
	oldCfg := ConfigFile
	ConfigFile = filepath.Join(tmp, "nonexistent.json")
	defer func() { ConfigFile = oldCfg }()

	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleConfig)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
}

func TestHandleConfig_GET_ReturnsExistingConfig(t *testing.T) {
	tmp := t.TempDir()
	oldCfg := ConfigFile
	ConfigFile = filepath.Join(tmp, "config.json")
	defer func() { ConfigFile = oldCfg }()

	expected := `{"enabled":false,"interval":60}`
	if err := os.WriteFile(ConfigFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleConfig)

	if !strings.Contains(string(body), "enabled") {
		t.Errorf("expected config content in output, got: %s", string(body))
	}
}

func TestHandleConfig_POST_ValidJSON(t *testing.T) {
	tmp := t.TempDir()
	oldCfg := ConfigFile
	ConfigFile = filepath.Join(tmp, "config.json")
	defer func() { ConfigFile = oldCfg }()

	postBody := `{"enabled":true,"interval":60}`
	setStdin(t, postBody)

	os.Setenv("REQUEST_METHOD", "POST")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleConfig)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", result["status"])
	}

	saved, err := os.ReadFile(ConfigFile)
	if err != nil {
		t.Fatal("config file was not written")
	}
	if strings.TrimSpace(string(saved)) != postBody {
		t.Errorf("expected '%s', got '%s'", postBody, string(saved))
	}
}

func TestHandleConfig_POST_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	oldCfg := ConfigFile
	ConfigFile = filepath.Join(tmp, "config.json")
	defer func() { ConfigFile = oldCfg }()

	setStdin(t, `not json`)

	os.Setenv("REQUEST_METHOD", "POST")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleConfig)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["status"] != "error" {
		t.Errorf("expected status 'error', got '%s'", result["status"])
	}
	if result["message"] != "Invalid JSON" {
		t.Errorf("expected message 'Invalid JSON', got '%s'", result["message"])
	}
}

func TestHandleConfig_MethodNotAllowed(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "PUT")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleConfig)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["error"] != "Method not allowed" {
		t.Errorf("expected error 'Method not allowed', got '%s'", result["error"])
	}
}
