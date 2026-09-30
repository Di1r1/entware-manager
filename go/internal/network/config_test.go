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

func TestHandleConfig_GET_MigratesFlatToNested(t *testing.T) {
	tmp := t.TempDir()
	oldCfg := ConfigFile
	ConfigFile = filepath.Join(tmp, "config.json")
	defer func() { ConfigFile = oldCfg }()

	flat := `{"autostart":true,"interval":60,"ping_host":"1.1.1.1","sidebar":{"x":1}}`
	if err := os.WriteFile(ConfigFile, []byte(flat), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleConfig)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	wd, ok := result["watchdog"].(map[string]interface{})
	if !ok {
		t.Fatalf("no .watchdog in output: %s", string(body))
	}
	if wd["interval"] != float64(60) {
		t.Errorf("want .watchdog.interval 60, got %v", wd["interval"])
	}
	if wd["ping_host"] != "1.1.1.1" {
		t.Errorf("want .watchdog.ping_host 1.1.1.1, got %v", wd["ping_host"])
	}
	for _, k := range []string{"interval", "ping_host"} {
		if _, exists := result[k]; exists {
			t.Errorf("flat key %q not removed", k)
		}
	}
	if _, exists := result["sidebar"]; !exists {
		t.Errorf("unknown key sidebar lost in migration")
	}

	saved, err := os.ReadFile(ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	var persisted map[string]interface{}
	if err := json.Unmarshal(saved, &persisted); err != nil {
		t.Fatalf("persisted file invalid JSON: %v", err)
	}
	if _, exists := persisted["interval"]; exists {
		t.Errorf("migration not persisted to file")
	}
}

func TestHandleConfig_GET_NestedWinsOverFlat(t *testing.T) {
	tmp := t.TempDir()
	oldCfg := ConfigFile
	ConfigFile = filepath.Join(tmp, "config.json")
	defer func() { ConfigFile = oldCfg }()

	mixed := `{"interval":60,"watchdog":{"interval":30,"watch_wifi":true}}`
	if err := os.WriteFile(ConfigFile, []byte(mixed), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleConfig)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	wd := result["watchdog"].(map[string]interface{})
	if wd["interval"] != float64(30) {
		t.Errorf("nested must win: want 30, got %v", wd["interval"])
	}
	if wd["watch_wifi"] != true {
		t.Errorf("NOC key watch_wifi lost in migration")
	}
	if _, exists := result["interval"]; exists {
		t.Errorf("flat key interval not removed")
	}
}
