package services

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCheckDeps_OutputsJSON(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleCheckDeps)

	var result DepsResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result.OverallStatus == "" {
		t.Fatal("missing overall_status")
	}
	if result.Timestamp == "" {
		t.Fatal("missing timestamp")
	}
}

func TestCheckDeps_GETOnly(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "POST")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleCheckDeps)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["error"] != "Method not allowed" {
		t.Errorf("expected 'Method not allowed', got '%s'", result["error"])
	}
}

func TestCheckDeps_BaseFields(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleCheckDeps)

	var result DepsResult
	json.Unmarshal(body, &result)

	// These should always be boolean
	if result.Base.Sed != (lookPath("sed")) {
		t.Errorf("Sed mismatch")
	}
	if result.Base.Awk != (lookPath("awk")) {
		t.Errorf("Awk mismatch")
	}
	if result.Base.Grep != (lookPath("grep")) {
		t.Errorf("Grep mismatch")
	}
}

func TestCheckDeps_Sections(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleCheckDeps)

	var result DepsResult
	json.Unmarshal(body, &result)

	valid := map[string]bool{"ok": true, "missing": true, "partial": true}
	for name, val := range map[string]string{
		"packages":   result.Sections.Packages,
		"services":   result.Sections.Services,
		"monitoring": result.Sections.Monitoring,
		"network":    result.Sections.Network,
		"logger":     result.Sections.Logger,
		"smart":      result.Sections.Smart,
	} {
		if !valid[val] {
			t.Errorf("section %s has invalid status: %q", name, val)
		}
	}
}

func TestCheckDeps_NewFields(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleCheckDeps)

	var result DepsResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// curl/bash/brctl должны быть bool, brctl_path — string
	if result.Deps.Curl != (lookPath("curl") || lookPath("/opt/bin/curl")) {
		t.Errorf("Curl mismatch")
	}
	if result.Deps.Bash != (lookPath("/opt/bin/bash") || lookPath("bash")) {
		t.Errorf("Bash mismatch")
	}
	if result.Deps.Brctl != (func() bool { _, ok := lookPathWithPath("brctl"); return ok }()) {
		t.Errorf("Brctl mismatch")
	}
}

func TestCheckDeps_SyntaxFields(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleCheckSyntax)

	var result SyntaxResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result.Timestamp == "" {
		t.Fatal("missing timestamp")
	}
	if result.Results == nil {
		t.Fatal("missing results")
	}
	for _, f := range result.Results {
		if f.Status != "ok" && f.Status != "error" {
			t.Errorf("file %s has invalid status: %q", f.File, f.Status)
		}
	}
}

func TestReadPid_NotFound(t *testing.T) {
	pid := readPid("/nonexistent/pid")
	if pid != 0 {
		t.Errorf("expected 0, got %d", pid)
	}
}
