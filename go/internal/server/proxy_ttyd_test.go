package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTTYDArgvHasPort(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		port string
		want bool
	}{
		{name: "dash-p", argv: []string{"ttyd", "-p", "9089", "-W", "-c", "admin:x"}, port: "9089", want: true},
		{name: "long-port", argv: []string{"ttyd", "--port", "8089", "htop"}, port: "8089", want: true},
		{name: "port-eq", argv: []string{"ttyd", "--port=9089", "/opt/bin/bash"}, port: "9089", want: true},
		{name: "bare-port", argv: []string{"ttyd", "8089"}, port: "8089", want: true},
		{name: "other-port", argv: []string{"ttyd", "-p", "8089", "htop"}, port: "9089", want: false},
		{name: "substring-no-match", argv: []string{"ttyd", "-p", "90890"}, port: "9089", want: false},
		{name: "cgi-self", argv: []string{"sh", "-c", "ENDPOINT=ttyd_control port=8089"}, port: "8089", want: false},
	}
	for _, c := range cases {
		if got := ttydArgvHasPort(c.argv, c.port); got != c.want {
			t.Errorf("%s: ttydArgvHasPort(%q, %q) = %v, want %v", c.name, c.argv, c.port, got, c.want)
		}
	}
}

// TestTTYDInjectRule проверяет правило подстановки через тестовый бэкенд:
// iframe/empty/websocket + без своего Authorization → Basic из lookup;
// document/без заголовка/свой Authorization → как есть.
func TestTTYDInjectRule(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	old := ttydCredLookup
	ttydCredLookup = func(port int) (string, bool) { return "admin:s3cret", true }
	defer func() { ttydCredLookup = old }()

	h := handleRemoteProxy(backend{prefix: "/terminal/", target: upstream.URL, ttydPort: 9089})
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:s3cret"))

	cases := []struct {
		name     string
		dest     string // "" — заголовок не ставить
		auth     string
		wantAuth string
	}{
		{name: "iframe", dest: "iframe", wantAuth: want},
		{name: "empty", dest: "empty", wantAuth: want},
		{name: "websocket", dest: "websocket", wantAuth: want},
		{name: "document", dest: "document", wantAuth: ""},
		{name: "no-header", dest: "", wantAuth: ""},
		{name: "own-auth", dest: "iframe", auth: "Basic dXNlcjpvd24=", wantAuth: "Basic dXNlcjpvd24="},
	}
	for _, c := range cases {
		gotAuth = ""
		req := httptest.NewRequest("GET", "/terminal/", nil)
		if c.dest != "" {
			req.Header.Set("Sec-Fetch-Dest", c.dest)
		}
		if c.auth != "" {
			req.Header.Set("Authorization", c.auth)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: code = %d, want 200", c.name, rec.Code)
		}
		if gotAuth != c.wantAuth {
			t.Errorf("%s: upstream Authorization = %q, want %q", c.name, gotAuth, c.wantAuth)
		}
	}
}

// TestTTYDNoCredNoInject: lookup без credential — запрос идёт как есть
// (ttyd ответит 401, браузер спросит пароль — старое поведение).
func TestTTYDNoCredNoInject(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	old := ttydCredLookup
	ttydCredLookup = func(port int) (string, bool) { return "", false }
	defer func() { ttydCredLookup = old }()

	h := handleRemoteProxy(backend{prefix: "/terminal/", target: upstream.URL, ttydPort: 9089})
	req := httptest.NewRequest("GET", "/terminal/", nil)
	req.Header.Set("Sec-Fetch-Dest", "iframe")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if gotAuth != "" {
		t.Errorf("upstream Authorization = %q, want пусто", gotAuth)
	}
}

// TestTTYDNonTTYDNoInject: бэкенд без ttydPort (rdp) — подстановки нет.
func TestTTYDNonTTYDNoInject(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	h := handleRemoteProxy(backend{prefix: "/other/", target: upstream.URL})
	req := httptest.NewRequest("GET", "/other/", nil)
	req.Header.Set("Sec-Fetch-Dest", "iframe")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if gotAuth != "" {
		t.Errorf("upstream Authorization = %q, want пусто", gotAuth)
	}
}
