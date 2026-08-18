package cgiutil

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestURLDecode(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", "hello"},
		{"plus", "a+b", "a+b"},
		{"encoded plus", "a%2Bb", "a+b"},
		{"percent", "%20", " "},
		{"cyrillic", "%D0%BF%D0%B0%D0%BD%D0%B5%D0%BB%D1%8C", "панель"},
		{"hex case", "%2f", "/"},
		{"broken percent", "100%", "100%"},
		{"broken percent short", "a%b", "a%b"},
		{"mixed", "a%20b%2Bc", "a b+c"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := URLDecode(c.in); got != c.want {
				t.Errorf("URLDecode(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestParseFormBody(t *testing.T) {
	params := ParseFormBody("a=1&b=%D0%BF%D0%B0%D0%BD%D0%B5%D0%BB%D1%8C&c=x+y&d=100%25")
	if params["a"] != "1" {
		t.Errorf("a = %q, want 1", params["a"])
	}
	if params["b"] != "панель" {
		t.Errorf("b = %q, want панель", params["b"])
	}
	if params["c"] != "x y" {
		t.Errorf("c = %q, want 'x y' (plus to space)", params["c"])
	}
	if params["d"] != "100%" {
		t.Errorf("d = %q, want 100%%", params["d"])
	}
	if len(params) != 4 {
		t.Errorf("len = %d, want 4", len(params))
	}
}

func TestParseFormBody_Empty(t *testing.T) {
	params := ParseFormBody("")
	if len(params) != 0 {
		t.Errorf("empty body should yield no params, got %d", len(params))
	}
}

func TestParseFormBody_Malformed(t *testing.T) {
	params := ParseFormBody("novalue&k=v")
	if _, ok := params["novalue"]; ok {
		t.Error("key without '=' should not be present")
	}
	if params["k"] != "v" {
		t.Errorf("k = %q, want v", params["k"])
	}
}

func TestGetQueryParam(t *testing.T) {
	old := os.Getenv("QUERY_STRING")
	os.Setenv("QUERY_STRING", "a=1&b=%D0%BF%D0%B0%D0%BD%D0%B5%D0%BB%D1%8C&c=x+y")
	defer os.Setenv("QUERY_STRING", old)

	if v := GetQueryParam("a"); v != "1" {
		t.Errorf("a = %q, want 1", v)
	}
	if v := GetQueryParam("b"); v != "панель" {
		t.Errorf("b = %q, want панель", v)
	}
	if v := GetQueryParam("c"); v != "x y" {
		t.Errorf("c = %q, want 'x y'", v)
	}
	if v := GetQueryParam("missing"); v != "" {
		t.Errorf("missing = %q, want empty", v)
	}
}

func TestGetParam_QueryOnly(t *testing.T) {
	oldQ := os.Getenv("QUERY_STRING")
	oldM := os.Getenv("REQUEST_METHOD")
	os.Setenv("QUERY_STRING", "action=run")
	os.Setenv("REQUEST_METHOD", "GET")
	defer func() {
		os.Setenv("QUERY_STRING", oldQ)
		os.Setenv("REQUEST_METHOD", oldM)
	}()

	if v := GetParam("action"); v != "run" {
		t.Errorf("action = %q, want run", v)
	}
}

func TestGetParam_PostBody(t *testing.T) {
	oldQ := os.Getenv("QUERY_STRING")
	oldM := os.Getenv("REQUEST_METHOD")
	oldStdin := os.Stdin
	os.Setenv("QUERY_STRING", "")
	os.Setenv("REQUEST_METHOD", "POST")
	f := withFakeStdin(t, "package=curl&x=y")
	defer func() {
		os.Setenv("QUERY_STRING", oldQ)
		os.Setenv("REQUEST_METHOD", oldM)
		os.Stdin = oldStdin
		f.Close()
	}()

	if v := GetParam("package"); v != "curl" {
		t.Errorf("package = %q, want curl", v)
	}
}

func TestWriteJSON(t *testing.T) {
	out := captureStdout(func() {
		WriteJSON(map[string]string{"status": "ok"})
	})
	if !strings.HasPrefix(out, "Content-type: application/json; charset=utf-8\n\n") {
		t.Errorf("missing content-type, got: %q", out)
	}
	if !strings.Contains(out, `"status":"ok"`) {
		t.Errorf("missing payload, got: %q", out)
	}
}

func TestWriteError(t *testing.T) {
	out := captureStdout(func() {
		WriteError("boom")
	})
	if !strings.Contains(out, `"error":"boom"`) {
		t.Errorf("WriteError output should contain {\"error\":\"boom\"}, got: %q", out)
	}
}

func TestWriteStatusError(t *testing.T) {
	out := captureStdout(func() {
		WriteStatusError("denied")
	})
	if !strings.Contains(out, `"status":"error"`) || !strings.Contains(out, `"message":"denied"`) {
		t.Errorf("WriteStatusError output wrong, got: %q", out)
	}
}

func TestNotAllowed(t *testing.T) {
	out := captureStdout(func() {
		NotAllowed()
	})
	if !strings.Contains(out, `"error":"Method not allowed"`) {
		t.Errorf("NotAllowed output wrong, got: %q", out)
	}
}

func TestIsGET_IsPOST(t *testing.T) {
	oldM := os.Getenv("REQUEST_METHOD")
	defer os.Setenv("REQUEST_METHOD", oldM)

	os.Setenv("REQUEST_METHOD", "GET")
	if !IsGET() || IsPOST() {
		t.Error("GET method misdetected")
	}
	os.Setenv("REQUEST_METHOD", "POST")
	if !IsPOST() || IsGET() {
		t.Error("POST method misdetected")
	}
}

func TestReadPOSTBody(t *testing.T) {
	oldStdin := os.Stdin
	f := withFakeStdin(t, "hello body")
	defer func() { os.Stdin = oldStdin; f.Close() }()

	if got := ReadPOSTBody(); got != "hello body" {
		t.Errorf("ReadPOSTBody = %q, want 'hello body'", got)
	}
}

func TestHumanSize(t *testing.T) {
	cases := []struct {
		size int64
		want string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1K"},
		{1536, "1K"},
		{2097152, "2M"},
		{3221225472, "3G"},
	}
	for _, c := range cases {
		if got := HumanSize(c.size); got != c.want {
			t.Errorf("HumanSize(%d): expected %q, got %q", c.size, c.want, got)
		}
	}
}

func withFakeStdin(t *testing.T, content string) *os.File {
	t.Helper()
	f, err := os.CreateTemp("", "cgiutil-stdin-*")
	if err != nil {
		t.Fatalf("create temp stdin: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp stdin: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("seek temp stdin: %v", err)
	}
	os.Stdin = f
	return f
}

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}
