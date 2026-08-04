// Di1r1
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
	"/index.html":                  true,
	"/style.css":                   true,
	"/entware.js":                  true,
	"/monitor.js":                  true,
	"/network.js":                  true,
	"/smart.js":                    true,
	"/modal.js":                    true,
	"/theme.js":                    true,
	"/icons.svg":                   true,
	"/version.json":                true,
	"/lib/utils.js":                true,
	"/menu/menu.js":                true,
	"/logger/system_sources.json":  true,
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
	if !staticWhitelist[clean] {
		http.NotFound(w, r)
		return
	}
	full := filepath.Join(webRoot, clean)
	http.ServeFile(w, r, full)
}
