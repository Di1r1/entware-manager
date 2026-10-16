package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListenAddress(t *testing.T) {
	cases := []struct {
		name string
		bind string
		port int
		want string
	}{
		{name: "all interfaces", bind: "", port: 8087, want: ":8087"},
		{name: "ipv4", bind: "192.168.10.1", port: 8087, want: "192.168.10.1:8087"},
		{name: "ipv6", bind: "::1", port: 8443, want: "[::1]:8443"},
		{name: "hostname", bind: "localhost", port: 8087, want: "localhost:8087"},
		{name: "trim spaces", bind: "  127.0.0.1  ", port: 8087, want: "127.0.0.1:8087"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ListenAddress(tc.bind, tc.port); got != tc.want {
				t.Fatalf("ListenAddress(%q, %d) = %q, want %q", tc.bind, tc.port, got, tc.want)
			}
		})
	}
}

func TestLoadConfigBind(t *testing.T) {
	oldConfigPath := serverConfig
	t.Cleanup(func() { serverConfig = oldConfigPath })

	serverConfig = filepath.Join(t.TempDir(), "server_config.json")
	if err := os.WriteFile(serverConfig, []byte(`{"bind":" 192.168.10.1 ","port":18087,"timeout":42,"tls":true,"tls_port":18443}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := LoadConfig()
	if cfg.Bind != "192.168.10.1" {
		t.Fatalf("Bind = %q, want %q", cfg.Bind, "192.168.10.1")
	}
	if cfg.Port != 18087 {
		t.Fatalf("Port = %d, want 18087", cfg.Port)
	}
	if cfg.Timeout != 42 {
		t.Fatalf("Timeout = %d, want 42", cfg.Timeout)
	}
	if !cfg.TLS {
		t.Fatal("TLS = false, want true")
	}
	if cfg.TLSPort != 18443 {
		t.Fatalf("TLSPort = %d, want 18443", cfg.TLSPort)
	}
}

func TestLoadConfigWithoutBindKeepsBackwardCompatibleDefaults(t *testing.T) {
	oldConfigPath := serverConfig
	t.Cleanup(func() { serverConfig = oldConfigPath })

	serverConfig = filepath.Join(t.TempDir(), "server_config.json")
	if err := os.WriteFile(serverConfig, []byte(`{"port":8087}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := LoadConfig()
	if cfg.Bind != "" {
		t.Fatalf("Bind = %q, want empty (all interfaces)", cfg.Bind)
	}
	if got := ListenAddress(cfg.Bind, cfg.Port); got != ":8087" {
		t.Fatalf("listen address = %q, want %q", got, ":8087")
	}
}
