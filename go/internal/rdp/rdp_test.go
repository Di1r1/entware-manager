package rdp

import (
	"os"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	restore := setEnv(t, WebRootEnv, t.TempDir())
	defer restore()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.ProxyPort != 9099 {
		t.Errorf("ProxyPort = %d, want 9099", cfg.ProxyPort)
	}
	if cfg.BinPath == "" {
		t.Errorf("BinPath empty")
	}
}

func TestSaveLoadConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	restore := setEnv(t, WebRootEnv, dir)
	defer restore()

	cfg := DefaultConfig()
	cfg.ProxyPort = 9100
	cfg.TargetHost = "192.168.3.50"
	cfg.TargetPort = 3389
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig error: %v", err)
	}
	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if got.ProxyPort != 9100 {
		t.Errorf("ProxyPort = %d, want 9100", got.ProxyPort)
	}
	if got.TargetHost != "192.168.3.50" {
		t.Errorf("TargetHost = %q", got.TargetHost)
	}
}

func TestSaveLoadConfigKeepsAllowSubnets(t *testing.T) {
	dir := t.TempDir()
	restore := setEnv(t, WebRootEnv, dir)
	defer restore()

	cfg := DefaultConfig()
	cfg.AllowSubnets = []string{"192.168.3.0/24", "10.1.30.0/24"}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig error: %v", err)
	}
	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if len(got.AllowSubnets) != 2 || got.AllowSubnets[0] != "192.168.3.0/24" || got.AllowSubnets[1] != "10.1.30.0/24" {
		t.Errorf("AllowSubnets after round-trip = %v, want [192.168.3.0/24 10.1.30.0/24]", got.AllowSubnets)
	}
}

func TestUnmarshalAllowSubnetsArray(t *testing.T) {
	dir := t.TempDir()
	restore := setEnv(t, WebRootEnv, dir)
	defer restore()
	os.WriteFile(dir+"/rdp_config.json", []byte(`{"proxy_port":9099,"allow_subnets":["192.168.3.0/24","10.1.30.0/24"]}`), 0644)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if len(cfg.AllowSubnets) != 2 {
		t.Errorf("AllowSubnets = %v, want 2 entries", cfg.AllowSubnets)
	}
}

func TestUnmarshalAllowSubnetsString(t *testing.T) {
	dir := t.TempDir()
	restore := setEnv(t, WebRootEnv, dir)
	defer restore()
	os.WriteFile(dir+"/rdp_config.json", []byte(`{"proxy_port":9099,"allow_subnets":"192.168.3.0/24,10.1.30.0/24"}`), 0644)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if len(cfg.AllowSubnets) != 2 {
		t.Errorf("AllowSubnets = %v, want 2 entries", cfg.AllowSubnets)
	}
}

func TestLoadConfigBadPortFallsBack(t *testing.T) {
	dir := t.TempDir()
	restore := setEnv(t, WebRootEnv, dir)
	defer restore()
	os.WriteFile(dir+"/rdp_config.json", []byte(`{"proxy_port":99999}`), 0644)
	cfg, _ := LoadConfig()
	if cfg.ProxyPort != 9099 {
		t.Errorf("bad port not reset: %d", cfg.ProxyPort)
	}
}

func TestStatusPathNotPanic(t *testing.T) {
	dir := t.TempDir()
	restore := setEnv(t, WebRootEnv, dir)
	defer restore()
	inst := Status()
	if inst.State != "stopped" {
		t.Errorf("state = %q, want stopped (test env, no proxy)", inst.State)
	}
}

func setEnv(t *testing.T, key, val string) func() {
	t.Helper()
	old, existed := os.LookupEnv(key)
	os.Setenv(key, val)
	return func() {
		if existed {
			os.Setenv(key, old)
		} else {
			os.Unsetenv(key)
		}
	}
}
