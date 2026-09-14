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
	"/i18n.js":                    true,
	"/locales/ru.json":            true,
	"/locales/en.json":            true,
	"/locales/tr.json":            true,
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

	// Locale runtime and dictionaries change independently while the i18n layer
	// is being refined. Do not let a browser keep an older translation engine
	// after an update or language switch.
	if clean == "/i18n.js" || strings.HasPrefix(clean, "/locales/") {
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}

	full := filepath.Join(webRoot, clean)
	http.ServeFile(w, r, full)
}
