// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"entware-manager/internal/auth"
	"entware-manager/internal/cgiutil"
)

const cgiPath = "/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin"

// readArch читает .arch (или определяет по GOARCH из env для тестов).
func readArch() string {
	data, err := os.ReadFile(filepath.Join(webRoot, ".arch"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// flatDispatch — таблица go.cgi: имя .cgi в cgi-bin -> бинарник.
// ENDPOINT в этом случае = имени (кроме smart — он вызывается без ENDPOINT).
var flatDispatch = map[string]string{
	// pkg
	"available": "pkg", "packages": "pkg", "installed": "pkg", "install": "pkg", "remove": "pkg",
	"upgrade": "pkg", "update": "pkg", "upgradable": "pkg", "api": "pkg",
	// stats
	"stats": "stats", "version": "stats", "help": "stats", "links_load": "stats",
	"links_save": "stats", "tmpfs": "stats", "tmpfs_clean": "stats", "view_file": "stats", "delete_file": "stats",
	"auth_config": "stats", "auth_log": "stats", "tls_config": "stats", "crontab": "stats", "crontab_update": "stats",
	"backup": "stats", "backup_restore": "stats", "update_check": "stats",
	"update_run": "stats", "update_status": "stats", "prepare_offline": "stats",
	"login": "stats", "logout": "stats", "session": "stats",
	// net
	"network_interfaces": "net", "network_routes": "net", "network_arp": "net",
	"network_status": "net", "network_stats": "net", "network_events": "net",
	"network_action": "net", "network_wifi": "net",
	// services
	"check_syntax": "services", "check_deps": "services", "services": "services",
	"service_action": "services", "ttyd_control": "services",
	// monitor
	"temperature": "monitor", "wifi_temp": "monitor", "temp_history": "monitor",
	"wifi_temp_history": "monitor", "kill_pid": "monitor",
	// smart
	"smart": "smart",
	// rdp
	"rdp_status": "rdp", "rdp_start": "rdp", "rdp_stop": "rdp", "rdp_config": "rdp",
	// telegram
	"telegram_config": "telegram", "telegram_test": "telegram",
	// bridge
	"bridge_discover": "bridge", "bridge_status": "bridge", "bridge_action": "bridge",
	"bridge_auth": "bridge", "bridge_prefs": "bridge", "bridge_stats": "bridge",
}

// subdirDispatch — подкаталоги /entware-cgi/<dir>/<name>.cgi.
// ENDPOINT = префикс + имя (как в go.cgi).
var subdirDispatch = map[string]struct {
	prefix string
	binary string
}{
	"network":          {prefix: "network_", binary: "net"},
	"logger":           {prefix: "logger_", binary: "logger"},
	"monitor":          {prefix: "monitor_", binary: "monitor"},
	"service_watchdog": {prefix: "service_watchdog_", binary: "services"},
}

// endpointResult — результат маппинга URL в подпроцесс.
type endpointResult struct {
	binary   string // entware-<binary>
	endpoint string // значение ENDPOINT (пустое — не выставлять)
}

// resolveEndpoint мапит путь /entware-cgi/... на (binary, endpoint).
func resolveEndpoint(dir, name string) (endpointResult, bool) {
	if dir == "cgi-bin" {
		bin, ok := flatDispatch[name]
		if !ok {
			return endpointResult{}, false
		}
		ep := name
		if bin == "smart" {
			ep = "" // go.cgi вызывает entware-smart без ENDPOINT
		}
		return endpointResult{binary: bin, endpoint: ep}, true
	}
	sd, ok := subdirDispatch[dir]
	if !ok {
		return endpointResult{}, false
	}
	return endpointResult{binary: sd.binary, endpoint: sd.prefix + name}, true
}

// resolveBinary ищет бинарник: сначала в каталоге архитектуры, затем flat.
func resolveBinary(bin string) (string, error) {
	if arch := readArch(); arch != "" {
		p := filepath.Join(cgiGoDir, arch, "entware-"+bin)
		if isExecutable(p) {
			return p, nil
		}
	}
	p := filepath.Join(cgiGoDir, "entware-"+bin)
	if isExecutable(p) {
		return p, nil
	}
	return "", fmt.Errorf("бинарник не найден: entware-%s", bin)
}

func isExecutable(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir() && (fi.Mode().Perm()&0111) != 0
}

// handleCGI обрабатывает /entware-cgi/... через subprocess-glue.
func handleCGI(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/entware-cgi/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(rest, "/")
	var dir, nameFile string
	switch len(parts) {
	case 1:
		dir, nameFile = "cgi-bin", parts[0]
	case 2:
		dir, nameFile = parts[0], parts[1]
	default:
		http.NotFound(w, r)
		return
	}
	if !strings.HasSuffix(nameFile, ".cgi") {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimSuffix(nameFile, ".cgi")

	res, ok := resolveEndpoint(dir, name)
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Гейт авторизации (go-режим): все CGI, кроме login/logout/session,
	// требуют валидную сессию, если пароль панели настроен.
	if !auth.Enabled() {
		// пароль не настроен — панель открыта
	} else if name != "login" && name != "logout" && name != "session" {
		if !auth.SessionValidCookie(auth.TokenFromHeader(r.Header.Get("Cookie"))) {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
	}

	binPath, err := resolveBinary(res.binary)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}

	cfg := LoadConfig()
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath)
	cmd.Env = buildCGIEnv(r, res.endpoint)
	cmd.Stdin = r.Body

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if len(stdout.Bytes()) > 0 {
			// Подпроцесс что-то отдал — отдаём, даже если exit != 0.
			writeCGIOutput(w, stdout.Bytes())
			return
		}
		msg := strings.TrimSpace(stderr.String())
		if ctx.Err() == context.DeadlineExceeded {
			msg = "timeout: " + name
		} else if msg == "" {
			msg = "exit: " + err.Error()
		}
		writeJSONError(w, "ошибка "+name+": "+msg)
		return
	}
	writeCGIOutput(w, stdout.Bytes())
}

// buildCGIEnv формирует CGI-окружение для подпроцесса.
func buildCGIEnv(r *http.Request, endpoint string) []string {
	env := []string{
		"REQUEST_METHOD=" + r.Method,
		"QUERY_STRING=" + r.URL.RawQuery,
		"REMOTE_ADDR=" + cgiutil.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"), r.Header.Get("X-Real-IP")),
		"PATH=" + cgiPath,
		"GATEWAY_INTERFACE=CGI/1.1",
		"SERVER_SOFTWARE=entware-server",
		"SCRIPT_NAME=" + r.URL.Path,
		"REQUEST_URI=" + r.RequestURI,
	}
	if endpoint != "" {
		env = append(env, "ENDPOINT="+endpoint)
	}
	if r.ContentLength >= 0 {
		env = append(env, "CONTENT_LENGTH="+strconv.FormatInt(r.ContentLength, 10))
	}
	if ct := r.Header.Get("Content-Type"); ct != "" {
		env = append(env, "CONTENT_TYPE="+ct)
	}
	if x := r.Header.Get("X-Requested-With"); x != "" {
		env = append(env, "HTTP_X_REQUESTED_WITH="+x)
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		env = append(env, "HTTP_ORIGIN="+origin)
	}
	if sfs := r.Header.Get("Sec-Fetch-Site"); sfs != "" {
		env = append(env, "HTTP_SEC_FETCH_SITE="+sfs)
	}
	if host := r.Host; host != "" {
		env = append(env, "HTTP_HOST="+host)
	}
	if cookie := r.Header.Get("Cookie"); cookie != "" {
		env = append(env, "HTTP_COOKIE="+cookie)
	}
	return env
}

// writeCGIOutput разбирает CGI-заголовки (до первой пустой строки)
// и отдаёт тело с правильным Content-Type.
func writeCGIOutput(w http.ResponseWriter, out []byte) {
	idx := bytes.Index(out, []byte("\n\n"))
	if idx < 0 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(out)
		return
	}
	header := string(out[:idx])
	body := out[idx+2:]
	for _, line := range strings.Split(header, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch strings.ToLower(k) {
		case "content-type":
			w.Header().Set("Content-Type", v)
		case "content-disposition":
			w.Header().Set("Content-Disposition", v)
		default:
			w.Header().Set(k, v)
		}
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.Write(body)
}

func writeJSONError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, `{"error":%q}`, msg)
}
