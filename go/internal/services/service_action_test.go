package services

import (
	"encoding/json"
	"os"
	"testing"
)

func TestHandleServiceAction_TraversalNameRejected(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "POST")
	defer os.Unsetenv("REQUEST_METHOD")

	// traversal-имя не должно приводить к выполнению произвольного файла.
	setStdin(t, "name=../../../../bin/reboot&action=start")

	body := captureStdout(t, HandleServiceAction)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["error"] != "Недопустимое имя службы" {
		t.Errorf("expected 'Недопустимое имя службы', got '%s'", result["error"])
	}
}

func TestHandleServiceAction_InvalidNameSymbolsRejected(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "POST")
	defer os.Unsetenv("REQUEST_METHOD")

	setStdin(t, "name=cron;rm+-rf&action=start")

	body := captureStdout(t, HandleServiceAction)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	if result["error"] != "Недопустимое имя службы" {
		t.Errorf("expected 'Недопустимое имя службы', got '%s'", result["error"])
	}
}

func TestHandleServiceAction_ValidNameOK(t *testing.T) {
	os.Setenv("REQUEST_METHOD", "POST")
	defer os.Unsetenv("REQUEST_METHOD")

	setStdin(t, "name=80lighttpd&action=status")

	body := captureStdout(t, HandleServiceAction)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, string(body))
	}
	// Имя валидно — ошибка будет про действие или службу, но не про имя.
	if result["error"] == "Недопустимое имя службы" {
		t.Errorf("valid name rejected: %s", string(body))
	}
}
