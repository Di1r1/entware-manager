// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// tls_config.cgi — включение/выключение self-signed HTTPS панели.
//
// GET  → {"tls":bool,"tls_port":N,"has_cert":bool,"server_running":bool}
// POST → password + enabled + port: правит server_config.json (сохраняя
// существующие port/timeout) и перезапускает entware-server в фоне
// (ответ уходит раньше, чем сервер примет TERM — иначе клиент не получит ok).
// Доступно только в go-режиме (в lighttpd HTTPS настраивается в его конфиге).
package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"entware-manager/internal/auth"
	"entware-manager/internal/cgiutil"
)

var (
	serverConfigPath = "/opt/web_entware/server_config.json"
	serverInitScript = "/opt/etc/init.d/S80entware-server"
)

func HandleTLSConfig() {
	switch os.Getenv("REQUEST_METHOD") {
	case "GET":
		handleTLSGet()
	case "POST":
		handleTLSPost()
	default:
		cgiutil.NotAllowed()
	}
}

func handleTLSGet() {
	cfg := readServerConfigRaw()
	certOK := serverCertExists()
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
		"tls":            cfg.TLS,
		"tls_port":       cfg.TLSPort,
		"has_cert":       certOK,
		"server_running": serverPidAlive(),
	})
}

func handleTLSPost() {
	if auth.IsCrossSiteOrigin() {
		writeSimpleJSON(map[string]interface{}{"status": "error", "message": auth.CrossSiteDeny})
		return
	}
	body := cgiutil.ReadPOSTBody()
	params := cgiutil.ParseFormBody(body)

	// Защита мутации паролем панели (как rdp_config).
	if !auth.CheckPassword(params["password"]) {
		time.Sleep(500 * time.Millisecond)
		writeSimpleJSON(map[string]interface{}{"status": "error", "message": "Неверный пароль"})
		return
	}
	if !serverPidAlive() {
		writeSimpleJSON(map[string]interface{}{"status": "error", "message": "Доступно только в режиме go (entware-server не запущен)"})
		return
	}

	cfg := readServerConfigRaw()
	cfg.TLS = params["enabled"] == "true"
	if p := params["port"]; p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 && n < 65536 {
			cfg.TLSPort = n
		}
	}
	if cfg.TLSPort <= 0 || cfg.TLSPort > 65535 {
		cfg.TLSPort = 8443
	}
	if err := writeServerConfig(cfg); err != nil {
		writeSimpleJSON(map[string]interface{}{"status": "error", "message": "Не удалось сохранить настройки"})
		return
	}

	writeSimpleJSON(map[string]interface{}{
		"status":  "ok",
		"message": tlsRestartMessage(cfg.TLS, cfg.TLSPort),
	})

	// Рестарт — отдельным отсоединённым процессом (setsid): CGI-процесс
	// завершается сразу после вывода ответа, и горутина внутри него умерла бы,
	// не дождавшись запуска. setsid-шелл переживает смерть родителя.
	cmd := exec.Command("/bin/sh", "-c",
		fmt.Sprintf("sleep 1; %s restart >/dev/null 2>&1", serverInitScript))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	_ = cmd.Start()
}

func tlsRestartMessage(enabled bool, port int) string {
	if enabled {
		return fmt.Sprintf("Сохранено. Сервер перезапускается, HTTPS будет на порту %d", port)
	}
	return "Сохранено. Сервер перезапускается, HTTPS будет отключён"
}

// --- вспомогательные ---

// tlsServerConfig — поля server_config.json, которыми управляет этот эндпоинт.
type tlsServerConfig struct {
	Port    int  `json:"port"`
	Timeout int  `json:"timeout,omitempty"`
	TLS     bool `json:"tls"`
	TLSPort int  `json:"tls_port"`
}

func readServerConfigRaw() tlsServerConfig {
	cfg := tlsServerConfig{Port: 8087, TLSPort: 8443}
	data, err := os.ReadFile(serverConfigPath)
	if err != nil {
		return cfg
	}
	var c tlsServerConfig
	if json.Unmarshal(data, &c) != nil {
		return cfg
	}
	if c.Port > 0 {
		cfg.Port = c.Port
	}
	if c.Timeout > 0 {
		cfg.Timeout = c.Timeout
	}
	if c.TLSPort > 0 {
		cfg.TLSPort = c.TLSPort
	}
	cfg.TLS = c.TLS
	return cfg
}

func writeServerConfig(cfg tlsServerConfig) error {
	out, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	tmp := serverConfigPath + ".tmp"
	if err := os.WriteFile(tmp, out, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, serverConfigPath); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func serverPidAlive() bool {
	data, err := os.ReadFile("/opt/var/run/entware-server.pid")
	if err != nil {
		return false
	}
	var pid int
	fmt.Sscanf(string(data), "%d", &pid)
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil // тот же паттерн, что services.pidIsAlive
}

func serverCertExists() bool {
	c := "/opt/web_entware/ssl/panel.crt"
	k := "/opt/web_entware/ssl/panel.key"
	st1, e1 := os.Stat(c)
	st2, e2 := os.Stat(k)
	return e1 == nil && e2 == nil && !st1.IsDir() && !st2.IsDir()
}

func writeSimpleJSON(v map[string]interface{}) {
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	json.NewEncoder(os.Stdout).Encode(v)
}
