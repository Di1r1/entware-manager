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
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// backend — один проксируемый сервис.
type backend struct {
	prefix string // URL-префикс на панели, напр. /terminal/
	target string // http://127.0.0.1:9089
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
		{prefix: "/terminal/", target: "http://127.0.0.1:9089"},
		{prefix: "/htop/", target: "http://127.0.0.1:8089"},
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
	// Для /rdp/ порт grdp-proxy может смениться в rdp_config.json — берём
	// актуальный при каждом запросе.
	if b.prefix == "/rdp/" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t, perr := url.Parse("http://127.0.0.1:" + strconv.Itoa(rdpProxyPort()))
			if perr != nil {
				http.Error(w, "proxy: bad rdp target", http.StatusBadGateway)
				return
			}
			httputil.NewSingleHostReverseProxy(t).ServeHTTP(w, r)
		})
	}
	return proxy
}

// newWebSocketProxy проксирует WebSocket grdp-proxy (/ws).
// Клиент grdpwasm hardcoded'ит путь /ws и подключается на location.host,
// поэтому на панели монтируем ровно /ws (без префикса).
func newWebSocketProxy() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := url.Parse("http://127.0.0.1:" + strconv.Itoa(rdpProxyPort()))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		httputil.NewSingleHostReverseProxy(u).ServeHTTP(w, r)
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
		httputil.NewSingleHostReverseProxy(t).ServeHTTP(w, r)
	})
}
