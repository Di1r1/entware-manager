package server

import (
	"net/http/httptest"
	"testing"
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
