// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package rdp

import (
	"encoding/json"
	"os"
	"strings"
)

// Константы проекта (RULES п.4 — без хардкодов на лету).
const (
	ConfigName     = "rdp_config.json"
	WebRootEnv     = "RDP_WEB_ROOT"
	defaultWebRoot = "/opt/web_entware"

	ProxyBinEnv      = "RDP_PROXY_BIN"
	defaultProxyBin  = "/opt/web_entware/cgi-bin/go/grdp-proxy"
	ProxyStatic      = "/opt/web_entware/static/rdp/"
	ProxyPidFile     = "/opt/var/run/grdp-proxy.pid"
	ProxyInitScript  = "/opt/etc/init.d/S90grdp-proxy"
	defaultProxyPort = 9099
)

// Config — единый конфиг RDP-модуля (без паролей).
type Config struct {
	ProxyPort    int      `json:"proxy_port"`
	ProxyHost    string   `json:"proxy_host,omitempty"`
	BinPath      string   `json:"bin_path"`
	StaticDir    string   `json:"static_dir"`
	Enabled      bool     `json:"enabled"`
	TargetHost   string   `json:"target_host,omitempty"`
	TargetPort   int      `json:"target_port,omitempty"`
	AllowSubnets []string `json:"allow_subnets,omitempty"`
}

// UnmarshalJSON принимает allow_subnets и как массив, и как строку CIDR через запятую.
func (c *Config) UnmarshalJSON(data []byte) error {
	type alias Config
	var raw struct {
		alias
		AllowSubnets json.RawMessage `json:"allow_subnets"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = Config(raw.alias)
	if len(raw.AllowSubnets) == 0 {
		return nil
	}
	var arr []string
	if err := json.Unmarshal(raw.AllowSubnets, &arr); err == nil {
		c.AllowSubnets = arr
		return nil
	}
	var s string
	if err := json.Unmarshal(raw.AllowSubnets, &s); err == nil {
		c.AllowSubnets = splitCIDR(s)
	}
	return nil
}

// splitCIDR разбивает строку "cidr1,cidr2" на список (без пустых).
func splitCIDR(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func webRoot() string {
	if wr := os.Getenv(WebRootEnv); wr != "" {
		return wr
	}
	return defaultWebRoot
}

func configPath() string {
	return webRoot() + "/" + ConfigName
}

// LoadConfig читает rdp_config.json. При отсутствии — дефолты.
func LoadConfig() (Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return DefaultConfig(), nil
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}
	if cfg.ProxyPort < 1 || cfg.ProxyPort > 65535 {
		cfg.ProxyPort = defaultProxyPort
	}
	cfg.fillDefaults()
	return cfg, nil
}

// DefaultConfig возвращает конфиг по умолчанию.
func DefaultConfig() Config {
	return Config{
		ProxyPort: defaultProxyPort,
		BinPath:   defaultProxyBin,
		StaticDir: ProxyStatic,
	}
}

func (c *Config) fillDefaults() {
	if c.BinPath == "" {
		c.BinPath = defaultProxyBin
	}
	if c.StaticDir == "" {
		c.StaticDir = ProxyStatic
	}
	if c.ProxyPort < 1 || c.ProxyPort > 65535 {
		c.ProxyPort = defaultProxyPort
	}
}

// SaveConfig атомарно пишет конфиг (temp + mv).
func SaveConfig(cfg Config) error {
	cfg.fillDefaults()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := configPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, configPath())
}
