// Di1r1
// Package server — собственный HTTP-сервер Entware Manager.
//
// Заменяет lighttpd на роутерах, где стоит сторонний lighttpd
// (nfqws/zapret и т.п.) и общий конфиг конфликтует по server.port.
//
// Схема:
//
//	/entware-manager/  — статика из белого списка файлов (web/ каталог)
//	/entware-cgi/…     — CGI-вызовы через subprocess-glue:
//	                      маппинг имени .cgi -> бинарник + ENDPOINT
//	                      (повторяет go.cgi), запуск подпроцесса с
//	                      CGI-окружением, ответ прокидывается в HTTP.
package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"entware-manager/internal/auth"
)

const (
	defaultPort    = 8087
	defaultTimeout = 300 // секунд на CGI-запрос (opkg install может идти долго)
)

// Пути — переопределяются через env для локального тестирования.
var (
	webRoot      = envOr("EWM_WEB_ROOT", "/opt/web_entware")
	serverConfig = envOr("EWM_SERVER_CONFIG", "/opt/web_entware/server_config.json")
	pidFile      = envOr("EWM_PID_FILE", "/opt/var/run/entware-server.pid")
	logFile      = envOr("EWM_LOG_FILE", "/opt/var/log/entware/server.log")
	cgiGoDir     = envOr("EWM_GO_DIR", "/opt/web_entware/cgi-bin/go")
)

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

// Config — настройки сервера из server_config.json.
type Config struct {
	Port    int `json:"port"`
	Timeout int `json:"timeout"` // секунд на CGI-запрос
}

// LoadConfig читает server_config.json (при ошибке — значения по умолчанию).
func LoadConfig() Config {
	cfg := Config{Port: defaultPort, Timeout: defaultTimeout}
	data, err := os.ReadFile(serverConfig)
	if err != nil {
		return cfg
	}
	var c Config
	if json.Unmarshal(data, &c) != nil {
		return cfg
	}
	if c.Port > 0 && c.Port < 65536 {
		cfg.Port = c.Port
	}
	if c.Timeout > 0 {
		cfg.Timeout = c.Timeout
	}
	return cfg
}

// NewHandler собирает маршруты сервера.
func NewHandler() http.Handler {
	registerProxyBackends()
	mux := http.NewServeMux()
	mux.HandleFunc("/entware-manager/", handleStatic)
	mux.HandleFunc("/entware-cgi/", handleCGI)
	// Прокси встроенных сервисов на едином origin 8087 (см. proxy.go).
	// Гейт сессии: если пароль панели настроен, прокси доступны только
	// после входа (иначе любой LAN-клиент дёргал бы RDP/терминал без пароля).
	mux.Handle("/ws", authGate(newWebSocketProxy()))
	for _, b := range proxyBackends {
		mux.Handle(b.prefix, authGate(handleRemoteProxy(b)))
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/entware-manager/", http.StatusFound)
	})
	return mux
}

// authGate закрывает прокси-маршруты, если включён пароль панели.
// Поведение повторяет гейт из handleCGI (cgi.go): без валидной сессии — 401.
func authGate(next http.Handler) http.Handler {
	if next == nil {
		return http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth.Enabled() && !auth.SessionValidCookie(auth.TokenFromHeader(r.Header.Get("Cookie"))) {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SetupLogging открывает лог-файл (каталог создаёт init-скрипт).
func SetupLogging() *os.File {
	if err := os.MkdirAll("/opt/var/log/entware", 0755); err != nil {
		return nil
	}
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil
	}
	log.SetOutput(f)
	return f
}

// WritePID пишет pid-файл атомарно (для init-скрипта и watchdog).
func WritePID() {
	if err := os.MkdirAll("/opt/var/run", 0755); err != nil {
		return
	}
	tmp := pidFile + ".tmp"
	_ = os.WriteFile(tmp, []byte(strconv.Itoa(os.Getpid())+"\n"), 0644)
	_ = os.Rename(tmp, pidFile)
}

// PidFile возвращает путь к pid-файлу (для удаления при выходе).
func PidFile() string {
	return pidFile
}
