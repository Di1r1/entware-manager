// Di1r1
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
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// backend — один проксируемый сервис.
type backend struct {
	prefix string // URL-префикс на панели, напр. /terminal/
	target string // http://127.0.0.1:9089
}

var proxyBackends []backend

// registerProxyBackends настраивает список бэкендов (вызывается из NewHandler).
func registerProxyBackends() {
	proxyBackends = []backend{
		{prefix: "/terminal/", target: "http://127.0.0.1:9089"},
		{prefix: "/htop/", target: "http://127.0.0.1:8089"},
		{prefix: "/rdp/", target: "http://127.0.0.1:9099"},
	}
}

// handleRemoteProxy проксирует подпуть на loopback-бэкенд.
func handleRemoteProxy(b backend) http.Handler {
	u, err := url.Parse(b.target)
	if err != nil {
		log.Printf("proxy: bad target %q: %v", b.target, err)
		return http.NotFoundHandler()
	}
	return httputil.NewSingleHostReverseProxy(u)
}

// newWebSocketProxy проксирует WebSocket grdp-proxy (/ws).
// Клиент grdpwasm hardcoded'ит путь /ws и подключается на location.host,
// поэтому на панели монтируем ровно /ws (без префикса).
func newWebSocketProxy() http.Handler {
	u, err := url.Parse("http://127.0.0.1:9099")
	if err != nil {
		return http.NotFoundHandler()
	}
	return httputil.NewSingleHostReverseProxy(u)
}
