package server

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"entware-manager/internal/auth"
)

func TestResolveEndpointFlat(t *testing.T) {
	cases := []struct {
		name     string
		dir      string
		binary   string
		endpoint string
		ok       bool
	}{
		{name: "network_arp", dir: "cgi-bin", binary: "net", endpoint: "network_arp", ok: true},
		{name: "stats", dir: "cgi-bin", binary: "stats", endpoint: "stats", ok: true},
		{name: "update_run", dir: "cgi-bin", binary: "stats", endpoint: "update_run", ok: true},
		{name: "temperature", dir: "cgi-bin", binary: "monitor", endpoint: "temperature", ok: true},
		{name: "api", dir: "cgi-bin", binary: "pkg", endpoint: "api", ok: true},
		{name: "smart", dir: "cgi-bin", binary: "smart", endpoint: "", ok: true},
		{name: "unknown", dir: "cgi-bin", binary: "", endpoint: "", ok: false},
	}
	for _, c := range cases {
		res, ok := resolveEndpoint(c.dir, c.name)
		if ok != c.ok {
			t.Errorf("resolveEndpoint(%q,%q) ok=%v want %v", c.dir, c.name, ok, c.ok)
			continue
		}
		if ok {
			if res.binary != c.binary || res.endpoint != c.endpoint {
				t.Errorf("resolveEndpoint(%q,%q) = (%s,%s) want (%s,%s)",
					c.dir, c.name, res.binary, res.endpoint, c.binary, c.endpoint)
			}
		}
	}
}

func TestResolveEndpointSubdir(t *testing.T) {
	cases := []struct {
		dir, name string
		binary    string
		endpoint  string
	}{
		{"network", "arp", "net", "network_arp"},
		{"logger", "config", "logger", "logger_config"},
		{"monitor", "status", "monitor", "monitor_status"},
		{"service_watchdog", "status", "services", "service_watchdog_status"},
	}
	for _, c := range cases {
		res, ok := resolveEndpoint(c.dir, c.name)
		if !ok || res.binary != c.binary || res.endpoint != c.endpoint {
			t.Errorf("resolveEndpoint(%q,%q) = (%v,%v,%v) want (%v,%v)",
				c.dir, c.name, res, ok, res.endpoint, c.binary, c.endpoint)
		}
	}
	if _, ok := resolveEndpoint("nope", "x"); ok {
		t.Error("unknown subdir should not resolve")
	}
}

func TestWriteCGIOutput(t *testing.T) {
	out := []byte("Content-type: application/json; charset=utf-8\n\n{\"ok\":true}")
	rec := httptest.NewRecorder()
	writeCGIOutput(rec, out)
	if rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != "{\"ok\":true}" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestWriteCGIOutputMultiHeader(t *testing.T) {
	out := []byte("Content-type: application/gzip\nContent-Disposition: attachment; filename=x.tar.gz\n\nBIN")
	rec := httptest.NewRecorder()
	writeCGIOutput(rec, out)
	if rec.Header().Get("Content-Disposition") != "attachment; filename=x.tar.gz" {
		t.Errorf("Content-Disposition = %q", rec.Header().Get("Content-Disposition"))
	}
	if rec.Body.String() != "BIN" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestWriteCGIOutputNoHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	writeCGIOutput(rec, []byte("just text"))
	if rec.Body.String() != "just text" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestBuildCGIEnv(t *testing.T) {
	r := httptest.NewRequest("POST", "/entware-cgi/network/config.cgi?x=1", nil)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Requested-With", "XMLHttpRequest")
	env := buildCGIEnv(r, "network_config")
	m := map[string]string{}
	for _, e := range env {
		if i := indexByte(e, '='); i >= 0 {
			m[e[:i]] = e[i+1:]
		}
	}
	if m["REQUEST_METHOD"] != "POST" {
		t.Errorf("REQUEST_METHOD = %q", m["REQUEST_METHOD"])
	}
	if m["QUERY_STRING"] != "x=1" {
		t.Errorf("QUERY_STRING = %q", m["QUERY_STRING"])
	}
	if m["ENDPOINT"] != "network_config" {
		t.Errorf("ENDPOINT = %q", m["ENDPOINT"])
	}
	if m["CONTENT_TYPE"] != "application/json" {
		t.Errorf("CONTENT_TYPE = %q", m["CONTENT_TYPE"])
	}
	if m["HTTP_X_REQUESTED_WITH"] != "XMLHttpRequest" {
		t.Errorf("HTTP_X_REQUESTED_WITH = %q", m["HTTP_X_REQUESTED_WITH"])
	}
	if m["PATH"] != cgiPath {
		t.Errorf("PATH = %q", m["PATH"])
	}
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func TestProxyRoutesRegistered(t *testing.T) {
	h := NewHandler()
	if h == nil {
		t.Fatal("NewHandler returned nil")
	}
	// Все префиксы должны быть в списке бэкендов.
	registerProxyBackends()
	want := map[string]bool{"/terminal/": false, "/htop/": false, "/rdp/": false}
	for _, b := range proxyBackends {
		delete(want, b.prefix)
	}
	for p := range want {
		t.Errorf("proxy backend %q не зарегистрирован", p)
	}
}

// TestProxyRoutingGoMode проверяет, что entware-server (go-режим) монтирует
// подпути /terminal/, /htop/, /rdp/, /ws и не конфликтует со статикой.
func TestProxyRoutingGoMode(t *testing.T) {
	webRoot = t.TempDir() + "/web"
	cgiGoDir = t.TempDir() + "/go"
	os.MkdirAll(webRoot, 0755)
	os.MkdirAll(cgiGoDir, 0755)

	h := NewHandler()
	srv := httptest.NewServer(h)
	defer srv.Close()

	for _, p := range []string{"/terminal/", "/htop/", "/rdp/", "/ws"} {
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatalf("GET %s: %v", p, err)
		}
		resp.Body.Close()
		// 5xx (502/503) — значит маршрут смонтирован и попытался проксировать
		// на loopback-бэкенд, которого нет. Любой другой код (404) — роут не найден.
		if resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("GET %s = %d, want 502/503 (прокси на несуществующий бэкенд)", p, resp.StatusCode)
		}
	}

	// Статика и CGI не должны попадать в прокси.
	resp, err := http.Get(srv.URL + "/entware-manager/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusOK {
		t.Errorf("GET /entware-manager/ = %d", resp.StatusCode)
	}
}

// TestProxyAuthGate проверяет, что при настроенном пароле панели
// прокси-маршруты закрыты без валидной сессии (401), а с сессией — доступны.
func TestProxyAuthGate(t *testing.T) {
	webRoot = t.TempDir() + "/web"
	cgiGoDir = t.TempDir() + "/go"
	os.MkdirAll(webRoot, 0755)
	os.MkdirAll(cgiGoDir, 0755)

	// Временный конфиг авторизации: пароль включён.
	auth.ConfigPath = t.TempDir() + "/auth_config.json"
	os.WriteFile(auth.ConfigPath, []byte(`{"enabled":true,"password_hash":"`+testHash("secret")+`"}`), 0600)
	auth.SessionFile = t.TempDir() + "/panel_session"
	defer func() {
		auth.ConfigPath = "/opt/web_entware/auth_config.json"
		auth.SessionFile = "/opt/var/run/panel_session"
	}()

	h := NewHandler()
	srv := httptest.NewServer(h)
	defer srv.Close()

	// Без cookie сессии — все прокси должны быть закрыты (401), а не 502/200.
	for _, p := range []string{"/terminal/", "/htop/", "/rdp/", "/ws"} {
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatalf("GET %s: %v", p, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("GET %s без сессии = %d, want 401", p, resp.StatusCode)
		}
	}

	// С валидной сессией — прокси доступны (502/503: бэкенда нет, но маршрут открыт).
	token, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"/terminal/", "/htop/", "/rdp/", "/ws"} {
		req, _ := http.NewRequest("GET", srv.URL+p, nil)
		req.AddCookie(&http.Cookie{Name: "panel_session", Value: token})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET %s: %v", p, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("GET %s с сессией = %d, want 502/503", p, resp.StatusCode)
		}
	}
}

func testHash(password string) string {
	h := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", h)
}

// RDP-пинг (/rdp/ping, /ping) открыт БЕЗ сессии (паритет с lighttpd-режимом:
// клиент grdpwasm шлёт ping без cookie), а сам клиент /rdp/ — под гейтом.
func TestProxyRDPPingOpen(t *testing.T) {
	webRoot = t.TempDir() + "/web"
	cgiGoDir = t.TempDir() + "/go"
	os.MkdirAll(webRoot, 0755)
	os.MkdirAll(cgiGoDir, 0755)

	auth.ConfigPath = t.TempDir() + "/auth_config.json"
	os.WriteFile(auth.ConfigPath, []byte(`{"enabled":true,"password_hash":"`+testHash("secret")+`"}`), 0600)
	auth.SessionFile = t.TempDir() + "/panel_session"
	defer func() {
		auth.ConfigPath = "/opt/web_entware/auth_config.json"
		auth.SessionFile = "/opt/var/run/panel_session"
	}()

	h := NewHandler()
	srv := httptest.NewServer(h)
	defer srv.Close()

	// Без сессии: ping открыт (маршрут до grdp-proxy; бэкенда в тесте нет → 502,
	// НО НЕ 401 — гейт не применён). Клиент /rdp/ — под гейтом (401).
	for _, p := range []string{"/rdp/ping?target=10.0.0.1:3389", "/ping?target=10.0.0.1:3389"} {
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatalf("GET %s: %v", p, err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound {
			t.Errorf("GET %s без сессии = %d, want 502 (прокси открыт, не гейт/404)", p, resp.StatusCode)
		}
	}
	resp, err := http.Get(srv.URL + "/rdp/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /rdp/ без сессии = %d, want 401 (клиент остаётся под гейтом)", resp.StatusCode)
	}
}
