// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Обратное проксирование встроенных сервисов (ttyd, grdp-proxy)
// на едином origin панели (порт 8087).
//
// Мотивация: при удалённом доступе (Keenetic Remote/KeenDNS) наружу
// проброшен только 8087, а ttyd (9089/8089) и grdp-proxy (9099)
// сидят на отдельных портах и из интернета недоступны. Проксирование
// подпутей на тот же origin решает и mixed-content при HTTPS.
//
// Сервисы при этом должны слушать ТОЛЬКО loopback (-i lo для ttyd,
// -listen 127.0.0.1 для grdp-proxy) — тогда прямые порты закрыты даже
// в LAN, а доступ идёт только через панель.
package server

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// backend — один проксируемый сервис.
type backend struct {
	prefix string // URL-префикс на панели, напр. /terminal/
	target string // http://127.0.0.1:9089
	// ttydPort — порт ttyd-бэкенда (9089/8089). Если > 0, для встроенного
	// вида (iframe) прокси сам подставляет Basic Auth из cmdline процесса
	// ttyd, чтобы браузер не показывал системное окно пароля при каждой
	// загрузке вкладки. 0 — без подстановки (rdp, прочие).
	ttydPort int
}

var proxyBackends []backend

// rdpConfig — локальная копия полей rdp_config.json, нужных для прокси.
type rdpConfig struct {
	ProxyPort int `json:"proxy_port"`
}

// rdpProxyPort возвращает порт grdp-proxy из rdp_config.json (единая точка
// конфигурации). Кэшируется на 5 секунд, чтобы не читать файл на каждый
// запрос /rdp/ и /ws.
var (
	rdpPortCache   int
	rdpPortCached  bool
	rdpPortChecked time.Time
	rdpPortMu      sync.Mutex
)

func rdpProxyPort() int {
	rdpPortMu.Lock()
	defer rdpPortMu.Unlock()
	if rdpPortCached && time.Since(rdpPortChecked) < 5*time.Second {
		return rdpPortCache
	}
	port := 9099
	if data, err := os.ReadFile(filepath.Join(webRoot, "rdp_config.json")); err == nil {
		var cfg rdpConfig
		if json.Unmarshal(data, &cfg) == nil && cfg.ProxyPort > 0 && cfg.ProxyPort <= 65535 {
			port = cfg.ProxyPort
		}
	}
	rdpPortCache = port
	rdpPortCached = true
	rdpPortChecked = time.Now()
	return port
}

// registerProxyBackends настраивает список бэкендов (вызывается из NewHandler).
// Порт RDP читается из rdp_config.json динамически — прокси подхватывает
// смену порта без перезапуска entware-server.
func registerProxyBackends() {
	proxyBackends = []backend{
		{prefix: "/terminal/", target: "http://127.0.0.1:9089", ttydPort: 9089},
		{prefix: "/htop/", target: "http://127.0.0.1:8089", ttydPort: 8089},
		{prefix: "/rdp/", target: "http://127.0.0.1:" + strconv.Itoa(rdpProxyPort())},
	}
}

// handleRemoteProxy проксирует подпуть на loopback-бэкенд.
func handleRemoteProxy(b backend) http.Handler {
	u, err := url.Parse(b.target)
	if err != nil {
		log.Printf("proxy: bad target %q: %v", b.target, err)
		return http.NotFoundHandler()
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.FlushInterval = -1
	// Для /rdp/ порт grdp-proxy может смениться в rdp_config.json — берём
	// актуальный при каждом запросе.
	if b.prefix == "/rdp/" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t, perr := url.Parse("http://127.0.0.1:" + strconv.Itoa(rdpProxyPort()))
			if perr != nil {
				http.Error(w, "proxy: bad rdp target", http.StatusBadGateway)
				return
			}
			rdpProxy := httputil.NewSingleHostReverseProxy(t)
			rdpProxy.FlushInterval = -1
			rdpProxy.ServeHTTP(w, r)
		})
	}
	// ttyd-бэкенд: подстановка Basic Auth для встроенного вида (iframe),
	// чтобы вкладка Терминал/htop открывалась без системного окна пароля.
	if b.ttydPort > 0 {
		return ttydAuthHandler(b.ttydPort, proxy)
	}
	return proxy
}

// newWebSocketProxy проксирует WebSocket grdp-proxy (/ws).
// Клиент grdpwasm hardcoded'ит путь /ws и подключается на location.host,
// поэтому на панели монтируем ровно /ws (без префикса).
// Порт берём при каждом запросе (паритет с /rdp/ и rdpPingHandler):
// смена порта в rdp_config.json подхватывается без рестарта сервера.
func newWebSocketProxy() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := url.Parse("http://127.0.0.1:" + strconv.Itoa(rdpProxyPort()))
		if err != nil {
			http.Error(w, "proxy: bad rdp target", http.StatusBadGateway)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(u)
		proxy.FlushInterval = -1
		proxy.ServeHTTP(w, r)
	})
}

// rdpPingHandler проксирует /rdp/ping и /ping на grdp-proxy БЕЗ auth-гейта.
// Паритет с lighttpd-режимом: клиент grdpwasm шлёт ping без cookie сессии
// (fetch c credentials=same-origin не срабатывает в iframe при KeenDNS/redirect),
// иначе "RDP: N мс" молча скрывается (клиент глотает 401). Сам пробник
// безвреден — это TCP-RTT до цели, которую grdp-proxy валидирует по
// allow_subnets. Сам клиент /rdp/ остаётся за authGate.
func rdpPingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t, err := url.Parse("http://127.0.0.1:" + strconv.Itoa(rdpProxyPort()))
		if err != nil {
			http.Error(w, "proxy: bad rdp target", http.StatusBadGateway)
			return
		}
		p := httputil.NewSingleHostReverseProxy(t)
		p.FlushInterval = -1
		p.ServeHTTP(w, r)
	})
}

// ttydAuthHandler подставляет Basic Auth ttyd для встроенного вида
// (iframe во вкладках Терминал/htop), чтобы браузер не показывал системное
// окно пароля при каждой загрузке вкладки.
//
// Правило: подстановка — только если это НЕ навигация верхнего уровня
// (Sec-Fetch-Dest != "document") и у запроса ещё нет своего Authorization.
// Прямое открытие /terminal/ или /htop/ в новой вкладке/адресной строке
// (Sec-Fetch-Dest: document) идёт как раньше — ttyd отвечает 401 и пароль
// спрашивает браузер. Свой Authorization (введён в диалоге нового окна
// и закэширован браузером) никогда не перезаписывается.
//
// Пароль берётся из cmdline процесса ttyd (-c user:pass, см. startTTYD):
// он и так виден в ps на роутере, подстановка в loopback-запрос новой
// поверхности не создаёт. Доступ через прокси уже закрыт authGate панели.
func ttydAuthHandler(port int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" &&
			r.Header.Get("Sec-Fetch-Dest") != "" &&
			r.Header.Get("Sec-Fetch-Dest") != "document" {
			if cred, ok := ttydCredLookup(port); ok {
				r.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(cred)))
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ttydCredLookup — точка подстановки пароля ttyd (var для stub'ов в тестах).
var ttydCredLookup = ttydCredFromProc

type ttydCred struct {
	cred    string // "user:pass" из -c
	checked time.Time
}

var (
	ttydCredCache = map[int]ttydCred{}
	ttydCredMu    sync.Mutex
)

// ttydCredFromProc ищет процесс ttyd на порту и возвращает его credential
// из флага -c/--credential. Результат кэшируется на 10 секунд, чтобы не
// сканировать /proc на каждый субресурс iframe.
func ttydCredFromProc(port int) (string, bool) {
	ttydCredMu.Lock()
	if c, ok := ttydCredCache[port]; ok && time.Since(c.checked) < 10*time.Second {
		ttydCredMu.Unlock()
		return c.cred, c.cred != ""
	}
	ttydCredMu.Unlock()

	cred, ok := scanTTYDCredential(port)

	ttydCredMu.Lock()
	ttydCredCache[port] = ttydCred{cred: cred, checked: time.Now()}
	ttydCredMu.Unlock()
	return cred, ok
}

// scanTTYDCredential читает /proc и извлекает значение -c/--credential
// процесса ttyd, слушающего указанный порт. Аргументы разбираются по
// NUL-разделителю argv, поэтому пароль с пробелами не ломает парсинг.
func scanTTYDCredential(port int) (string, bool) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return "", false
	}
	portStr := strconv.Itoa(port)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err != nil || len(raw) == 0 {
			continue
		}
		argv := strings.Split(string(raw), "\x00")
		if len(argv) == 0 || filepath.Base(argv[0]) != "ttyd" {
			continue
		}
		if !ttydArgvHasPort(argv, portStr) {
			continue
		}
		for i := 0; i < len(argv); i++ {
			if (argv[i] == "-c" || argv[i] == "--credential") && i+1 < len(argv) && argv[i+1] != "" {
				return argv[i+1], true
			}
		}
	}
	return "", false
}

// ttydArgvHasPort проверяет, что argv запуска ttyd относится к порту:
// -p/--port <port>, --port=<port> или голый <port>. Матчинг по подстроке
// запрещён — он ловит чужие процессы (паритет с isTTYDOnPort в services).
func ttydArgvHasPort(argv []string, portStr string) bool {
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if a == "-p" || a == "--port" {
			if i+1 < len(argv) && argv[i+1] == portStr {
				return true
			}
			continue
		}
		if strings.HasPrefix(a, "--port=") && a[len("--port="):] == portStr {
			return true
		}
		if a == portStr {
			return true
		}
	}
	return false
}
