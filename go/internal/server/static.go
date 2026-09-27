// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package server

import (
	"net/http"
	"path/filepath"
	"strings"
)

// classicWhitelist — белый список файлов классической панели,
// доступных по HTTP под /entware-manager/.
var classicWhitelist = map[string]bool{
	"/index.html":                 true,
	"/style.css":                  true,
	"/entware.js":                 true,
	"/dashboard.js":               true,
	"/modal.js":                   true,
	"/monitor.js":                 true,
	"/network.js":                 true,
	"/smart.js":                   true,
	"/rdp.js":                     true,
	"/theme.js":                   true,
	"/icons.svg":                  true,
	"/favicon.svg":                true,
	"/version.json":               true,
	"/links.json":                 true,
	"/lib/utils.js":               true,
	"/menu/menu.js":               true,
	"/menu/menu.json":             true,
	"/logger/system_sources.json": true,
	"/logger/style.css":           true,
	"/logger/style-noc.css":       true,
	"/noc/index.html":             true,
	"/noc/noc-console.css":        true,
	"/noc/noc-console.js":         true,
	"/noc/theme-bridge.js":        true,
	"/cgi-bin/packages_log.cgi":   true,
	"/cgi-bin/network_events.cgi": true,
}

// classicStaticPrefixes — префиксы встроенных веб-приложений (RDP/ttyd),
// отдаются целиком: там только статические ассеты.
var classicStaticPrefixes = []string{"/static/rdp/", "/static/ttyd/"}

func handleStaticClassic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p := r.URL.Path
	if strings.HasPrefix(p, "/static/") {
		serveClassicFile(w, r, p)
		return
	}
	p = strings.TrimPrefix(p, "/entware-manager")
	if p == "" || p == "/" {
		p = "/index.html"
	}
	// NOC-консоль — единый origin через Go (без lighttpd):
	// голый /noc и /noc/ отдаём как index.html, иначе /noc без
	// слэша давал бы 404 (его нет в whitelist).
	if p == "/noc" || p == "/noc/" {
		p = "/noc/index.html"
	}
	serveClassicFile(w, r, p)
}

func serveClassicFile(w http.ResponseWriter, r *http.Request, p string) {
	if !classicWhitelist[p] && !hasPrefix(p, classicStaticPrefixes) {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(webRoot, p)
	http.ServeFile(w, r, path)
}

func hasPrefix(s string, prefixes []string) bool {
	for _, pref := range prefixes {
		if strings.HasPrefix(s, pref) {
			return true
		}
	}
	return false
}
