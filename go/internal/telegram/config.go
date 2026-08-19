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
	Enabled    bool       `json:"enabled"`
	BotToken   string     `json:"bot_token,omitempty"`
	ChatID     string     `json:"chat_id,omitempty"`
	Level      string     `json:"level,omitempty"`
	Sources    []string   `json:"sources,omitempty"`
	BotEnabled bool       `json:"bot_enabled,omitempty"`
	Autostart  bool       `json:"autostart,omitempty"`
	ProxyURL   string     `json:"proxy_url,omitempty"`
	Thresholds Thresholds `json:"thresholds,omitempty"`
	Configured bool       `json:"-"`
}

// Threshold — порог для одной метрики (вкл/выкл + значение).
type Threshold struct {
	Enabled bool `json:"enabled"`
	Value   int  `json:"value"`
}

// Thresholds — набор критических порогов для Telegram-уведомлений.
// Отдельный блок, не смешивается с полями бота (token/chat_id/level/sources).
type Thresholds struct {
	CPUTemp   Threshold `json:"cpu_temp"`
	WiFi0Temp Threshold `json:"wifi0_temp"`
	WiFi1Temp Threshold `json:"wifi1_temp"`
	CPULoad   Threshold `json:"cpu_load"`
	RAMUsed   Threshold `json:"ram_used"`
	DiskTemp  Threshold `json:"disk_temp"`
}

// defaultProxyURL — HTTP/SOCKS-прокси для доступа к api.telegram.org.
// Используется, когда провайдер блокирует Telegram (DPI) напрямую, но
// локальный прокси (например, xray/hysteria) обходит блокировку.
const defaultProxyURL = "http://127.0.0.1:10871"

// DefaultConfig возвращает конфиг по умолчанию.
func DefaultConfig() Config {
	return Config{
		Enabled:  false,
		Level:    "ERROR",
		Sources:  []string{"system", "monitor", "packages"},
		ProxyURL: defaultProxyURL,
		Thresholds: Thresholds{
			CPUTemp:   Threshold{Enabled: true, Value: 90},
			WiFi0Temp: Threshold{Enabled: true, Value: 100},
			WiFi1Temp: Threshold{Enabled: true, Value: 100},
			CPULoad:   Threshold{Enabled: true, Value: 95},
			RAMUsed:   Threshold{Enabled: true, Value: 90},
			DiskTemp:  Threshold{Enabled: false, Value: 60},
		},
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
		c.Sources = []string{"system", "monitor", "packages"}
	}
	if c.ProxyURL == "" {
		c.ProxyURL = defaultProxyURL
	}
	// Пороги: дефолт применяется по-метрично, когда value==0 (0 невалиден и для
	// °C, и для %). Это сохраняет и обратную совместимость со старым конфигом
	// (нет поля thresholds → value=0 → дефолт), и осознанный выбор пользователя
	// «выключил метрику» (value!=0 + enabled=false не перезаписывается).
	th := &c.Thresholds
	if th.CPUTemp.Value == 0 {
		th.CPUTemp = Threshold{Enabled: true, Value: 90}
	}
	if th.WiFi0Temp.Value == 0 {
		th.WiFi0Temp = Threshold{Enabled: true, Value: 100}
	}
	if th.WiFi1Temp.Value == 0 {
		th.WiFi1Temp = Threshold{Enabled: true, Value: 100}
	}
	if th.CPULoad.Value == 0 {
		th.CPULoad = Threshold{Enabled: true, Value: 95}
	}
	if th.RAMUsed.Value == 0 {
		th.RAMUsed = Threshold{Enabled: true, Value: 90}
	}
	if th.DiskTemp.Value == 0 {
		th.DiskTemp = Threshold{Enabled: false, Value: 60}
	}
}

// ValidateThresholds проверяет пороги: значение в разумных пределах
// (°C: 0-150, %: 0-100). Возвращает false при невалидном.
func ValidateThresholds(th Thresholds) bool {
	return th.CPUTemp.Value >= 0 && th.CPUTemp.Value <= 150 &&
		th.WiFi0Temp.Value >= 0 && th.WiFi0Temp.Value <= 150 &&
		th.WiFi1Temp.Value >= 0 && th.WiFi1Temp.Value <= 150 &&
		th.CPULoad.Value >= 0 && th.CPULoad.Value <= 100 &&
		th.RAMUsed.Value >= 0 && th.RAMUsed.Value <= 100 &&
		th.DiskTemp.Value >= 0 && th.DiskTemp.Value <= 150
}

// IsValidProxyURL возвращает true для http:// или socks5:// URL прокси.
func IsValidProxyURL(u string) bool {
	if u == "" {
		return true // пусто = без прокси (прямое соединение)
	}
	return len(u) >= 7 && (u[:7] == "http://" || u[:7] == "socks5:")
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
