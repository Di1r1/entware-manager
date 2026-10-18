package server

import (
	"net/http/httptest"
	"testing"
)

func TestParseCGIResponse(t *testing.T) {
	// Test 1: Status 400 with JSON body
	out := []byte(`Status: 400
Content-Type: application/json

{"error":"bad request"}
`)
	rec := httptest.NewRecorder()
	writeCGIOutput(rec, out)
	if rec.Code != 400 {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}
	expected := `{"error":"bad request"}
`
	if rec.Body.String() != expected {
		t.Errorf("Expected body %q, got %q", expected, rec.Body.String())
	}
	// Ensure Status header not present
	if rec.Header().Get("Status") != "" {
		t.Errorf("Unexpected Status header: %s", rec.Header().Get("Status"))
	}
}

func TestParseCGIResponse_NoHeaders(t *testing.T) {
	// Test 2: just a body (no \n\n)
	out := []byte(`just a body
`)
	rec := httptest.NewRecorder()
	writeCGIOutput(rec, out)
	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("Expected Content-Type text/plain; charset=utf-8, got %s", rec.Header().Get("Content-Type"))
	}
	expected := `just a body
`
	if rec.Body.String() != expected {
		t.Errorf("Expected body %q, got %q", expected, rec.Body.String())
	}
}

func TestParseCGIResponse_Status401(t *testing.T) {
	// Test 3: Status 401
	out := []byte(`Status: 401
Content-Type: text/plain

Unauthorized
`)
	rec := httptest.NewRecorder()
	writeCGIOutput(rec, out)
	if rec.Code != 401 {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", rec.Header().Get("Content-Type"))
	}
	expected := `Unauthorized
`
	if rec.Body.String() != expected {
		t.Errorf("Expected body %q, got %q", expected, rec.Body.String())
	}
	if rec.Header().Get("Status") != "" {
		t.Errorf("Unexpected Status header: %s", rec.Header().Get("Status"))
	}
}

func TestParseCGIResponse_InvalidStatus(t *testing.T) {
	// Test 4: Status ABC (invalid)
	out := []byte(`Status: ABC
Content-Type: text/plain

Hello
`)
	rec := httptest.NewRecorder()
	writeCGIOutput(rec, out)
	if rec.Code != 200 {
		t.Errorf("Expected status 200 (fallback), got %d", rec.Code)
	}
	// In fallback, we output all headers as HTTP headers, including Status
	if rec.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Status") != "ABC" {
		t.Errorf("Expected Status header ABC, got %s", rec.Header().Get("Status"))
	}
	expected := `Hello
`
	if rec.Body.String() != expected {
		t.Errorf("Expected body %q, got %q", expected, rec.Body.String())
	}
}
