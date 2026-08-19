// Package telegram — конфигурация и настройка Telegram-уведомлений/чат-бота.
package telegram

import (
	"encoding/json"
	"os"
)

// Пути проекта (RULES п.4 — без хардкодов на лету).
const (
	WebRootEnv     = "TG_WEB_ROOT"
	defaultWebRoot = "/opt/web_entware"
	ConfigName     = "telegram_config.json"
)

// Config — конфигурация Telegram-шлюза. bot_token не отдаётся наружу.
type Config struct {
	Enabled    bool     `json:"enabled"`
	BotToken   string   `json:"bot_token,omitempty"`
	ChatID     string   `json:"chat_id,omitempty"`
	Level      string   `json:"level,omitempty"`
	Sources    []string `json:"sources,omitempty"`
	BotEnabled bool     `json:"bot_enabled,omitempty"`
	Autostart  bool     `json:"autostart,omitempty"`
	Configured bool     `json:"-"`
}

// DefaultConfig возвращает конфиг по умолчанию.
func DefaultConfig() Config {
	return Config{
		Enabled: false,
		Level:   "ERROR",
		Sources: []string{"system", "monitor"},
	}
}

// IsValidChatID возвращает true, если chat_id — это число (допускается
// ведущий «-» для супергрупп/каналов: «-100…»).
func IsValidChatID(id string) bool {
	if id == "" {
		return false
	}
	if id[0] == '-' {
		id = id[1:]
	}
	if id == "" {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// IsValidLevel возвращает true для допустимого уровня.
func IsValidLevel(l string) bool {
	switch l {
	case "ERROR", "WARN", "INFO", "OFF":
		return true
	}
	return false
}

// RedactToken маскирует токен для вывода в логах/ошибках.
func RedactToken(tok string) string {
	if tok == "" {
		return ""
	}
	if len(tok) <= 6 {
		return "***"
	}
	return tok[:3] + "…" + tok[len(tok)-3:]
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

// LoadConfig читает telegram_config.json. При отсутствии — дефолты.
// Configured=true только если задан bot_token.
func LoadConfig() Config {
	cfg := DefaultConfig()
	data, err := os.ReadFile(configPath())
	if err == nil {
		json.Unmarshal(data, &cfg)
	}
	cfg.fillDefaults()
	cfg.Configured = cfg.BotToken != ""
	return cfg
}

func (c *Config) fillDefaults() {
	if c.Level == "" {
		c.Level = "ERROR"
	}
	if len(c.Sources) == 0 {
		c.Sources = []string{"system", "monitor"}
	}
}

// SaveConfig атомарно пишет конфиг (temp+mv), права 0600.
func SaveConfig(cfg Config) error {
	cfg.fillDefaults()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := configPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, configPath())
}
