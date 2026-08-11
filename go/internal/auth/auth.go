// Di1r1
// Общий пакет авторизации Entware Manager.
//
// Единая, fail-closed проверка пароля из /opt/web_entware/auth_config.json
// и Origin-чек для защиты действий от CSRF.
package auth

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

// ConfigPath — путь к конфигу авторизации (переменная для тестов).
var ConfigPath = "/opt/web_entware/auth_config.json"

type config struct {
	Enabled      bool   `json:"enabled"`
	PasswordHash string `json:"password_hash"`
	Password     string `json:"password"`
}

// LoadConfig читает auth_config.json.
// Возвращает (cfg, ok): ok=false если файл отсутствует или повреждён.
func LoadConfig() (config, bool) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return config{}, false
	}
	var cfg config
	if json.Unmarshal(data, &cfg) != nil {
		return config{}, false
	}
	return cfg, true
}

// EnabledReports разрешён ли доступ по конфигу.
// Правило fail-closed:
//   - файл отсутствует / битый → deny (false)
//   - enabled=true, но нет hash и нет plain → deny
//   - enabled=false в ВАЛИДНОМ конфиге → allow (явный выбор хозяина)
func EnabledReports() (allow bool, denyReason string) {
	cfg, ok := LoadConfig()
	if !ok {
		return false, "Пароль не настроен. Установите пароль в разделе «Защита»."
	}
	if !cfg.Enabled {
		return true, ""
	}
	if cfg.PasswordHash == "" && cfg.Password == "" {
		return false, "Пароль не настроен. Установите пароль в разделе «Защита»."
	}
	return true, ""
}

// CheckPassword проверяет введённый пароль (fail-closed).
func CheckPassword(password string) bool {
	allow, _ := EnabledReports()
	if !allow {
		return false
	}
	cfg, _ := LoadConfig()
	if !cfg.Enabled {
		return true
	}
	if cfg.PasswordHash != "" {
		h := sha256.Sum256([]byte(password))
		return fmt.Sprintf("%x", h) == cfg.PasswordHash
	}
	if cfg.Password != "" {
		return password == cfg.Password
	}
	return false
}

// Enabled — включена ли защита паролем (true = требуется вход).
// false = пароль не настроен / конфиг отсутствует или битый / enabled=false —
// панель открыта. Совпадает с go.cgi (lighttpd-режим): без валидного конфига
// защита не включается.
func Enabled() bool {
	cfg, ok := LoadConfig()
	if !ok {
		return false // нет конфига → защита не настроена → панель открыта
	}
	return cfg.Enabled && (cfg.PasswordHash != "" || cfg.Password != "")
}

// SHA256Hex — хеш для хранения в конфиге.
func SHA256Hex(password string) string {
	h := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", h)
}

// IsCrossSiteOrigin проверяет Origin/Sec-Fetch-Site заголовки.
// Правило: пустой Origin → allow (старые клиенты, curl);
// иначе host(и port) из Origin должны совпадать с HTTP_HOST;
// Sec-Fetch-Site=cross-site → deny.
func IsCrossSiteOrigin() bool {
	if sf := os.Getenv("HTTP_SEC_FETCH_SITE"); sf == "cross-site" {
		return true
	}
	origin := os.Getenv("HTTP_ORIGIN")
	if origin == "" {
		return false
	}
	if origin == "null" {
		return true
	}
	host := strings.TrimSpace(os.Getenv("HTTP_HOST"))
	if host == "" {
		return false
	}
	originHost, ok := originHost(origin)
	if !ok {
		return true
	}
	// HTTP_HOST может содержать порт ("192.168.3.1:8087") — сравниваем полный.
	return host != originHost
}

// originHost извлекает host[:port] из Origin (scheme://host[:port]).
func originHost(origin string) (string, bool) {
	rest := strings.TrimPrefix(origin, "http://")
	rest = strings.TrimPrefix(rest, "https://")
	if rest == origin {
		// нет схемы — некорректный Origin
		return "", false
	}
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		rest = rest[:i]
	}
	if _, _, err := net.SplitHostPort(rest); err == nil {
		return rest, true
	}
	// без порта
	if h := strings.Trim(rest, "[]"); h != "" && !strings.ContainsAny(h, " \t\r\n") {
		return h, true
	}
	return "", false
}
