// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package server

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

// staticWhitelist — белый список файлов, доступных по HTTP.
// Остальное содержимое /opt/web_entware (конфиги, скрипты, cgi-bin)
// наружу не отдаётся вообще.
var staticWhitelist = map[string]bool{
	"/index.html":                 true,
	"/style.css":                  true,
	"/entware.js":                 true,
	"/monitor.js":                 true,
	"/network.js":                 true,
	"/smart.js":                   true,
	"/rdp.js":                     true,
	"/rdp_config.json":            true,
	"/modal.js":                   true,
	"/theme.js":                   true,
	"/icons.svg":                  true,
	"/favicon.svg":                true,
	"/version.json":               true,
	"/lib/utils.js":               true,
	"/menu/menu.js":               true,
	"/menu/menu.json":             true,
	"/logger/system_sources.json": true,
	"/logger/style.css":           true,
	"/static/rdp/index.html":      true,
	"/static/rdp/main.wasm":       true,
	"/static/rdp/wasm_exec.js":    true,
}

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
	"/cgi-bin/packages_log.cgi":  true,
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

// handleStatic отдаёт файлы из белого списка под /entware-manager/.
func handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p := strings.TrimPrefix(r.URL.Path, "/entware-manager")
	if p == "" || p == "/" {
		p = "/index.html"
	}
	clean := path.Clean(p)
	// Каталог RDP-клиента отдаём как index.html: http.ServeFile сам
	// редиректит */index.html → */ (301), а голый */ без этой подмены
	// дал бы 404 (в whitelist только файлы).
	if clean == "/static/rdp" {
		clean = "/static/rdp/index.html"
	}
	if !staticWhitelist[clean] {
		http.NotFound(w, r)
		return
	}
	full := filepath.Join(webRoot, clean)
	http.ServeFile(w, r, full)
}
