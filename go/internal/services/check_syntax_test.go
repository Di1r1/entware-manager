package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckSyntax_GETOnly(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "POST")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleCheckSyntax)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["error"] != "Method not allowed" {
		t.Errorf("expected 'Method not allowed', got '%s'", result["error"])
	}
}

func TestCheckSyntax_WithTestDir(t *testing.T) {
	orig := webEntwareDir
	webEntwareDir = t.TempDir()
	defer func() { webEntwareDir = orig }()

	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleCheckSyntax)

	var result SyntaxResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result.Results == nil {
		t.Fatal("results should not be nil")
	}
}

func TestCheckSyntax_FindsFiles(t *testing.T) {
	orig := webEntwareDir
	webEntwareDir = t.TempDir()
	defer func() { webEntwareDir = orig }()

	os.MkdirAll(filepath.Join(webEntwareDir, "cgi-bin"), 0755)
	os.WriteFile(filepath.Join(webEntwareDir, "cgi-bin", "test.cgi"), []byte("#!/bin/sh\necho ok\n"), 0755)
	os.MkdirAll(filepath.Join(webEntwareDir, "lib"), 0755)
	os.WriteFile(filepath.Join(webEntwareDir, "lib", "test.sh"), []byte("#!/bin/sh\necho ok\n"), 0755)

	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleCheckSyntax)

	var result SyntaxResult
	json.Unmarshal(body, &result)

	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(result.Results), result.Results)
	}

	files := make(map[string]bool)
	for _, r := range result.Results {
		files[r.File] = true
	}
	if !files["cgi-bin/test.cgi"] {
		t.Error("missing cgi-bin/test.cgi")
	}
	if !files["lib/test.sh"] {
		t.Error("missing lib/test.sh")
	}
}

func TestCheckSyntax_ReportsSyntaxError(t *testing.T) {
	orig := webEntwareDir
	webEntwareDir = t.TempDir()
	defer func() { webEntwareDir = orig }()

	os.MkdirAll(filepath.Join(webEntwareDir, "cgi-bin"), 0755)
	// Bad syntax — missing "
	os.WriteFile(filepath.Join(webEntwareDir, "cgi-bin", "bad.cgi"), []byte("#!/bin/sh\necho \"unclosed\n"), 0755)

	os.Setenv("REQUEST_METHOD", "GET")
	defer os.Unsetenv("REQUEST_METHOD")

	body := captureStdout(t, HandleCheckSyntax)

	var result SyntaxResult
	json.Unmarshal(body, &result)

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Status != "error" {
		t.Errorf("expected error status, got %s", result.Results[0].Status)
	}
	if result.TotalErrors != 1 {
		t.Errorf("expected 1 total error, got %d", result.TotalErrors)
	}
}
