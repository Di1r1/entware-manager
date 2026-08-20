// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package rdp

import (
	"os"
	"strconv"

	"entware-manager/internal/auth"
	"entware-manager/internal/cgiutil"
)

// Handle dispatches подэндпоинты модуля RDP.
func Handle() {
	ep := os.Getenv("ENDPOINT")
	switch ep {
	case "rdp_status":
		HandleStatus()
	case "rdp_config":
		HandleConfig()
	case "rdp_start":
		HandleControl("start")
	case "rdp_stop":
		HandleControl("stop")
	default:
		cgiutil.WriteStatusError("unknown endpoint: " + ep)
	}
}

// HandleStatus отдаёт статус прокси (GET, без пароля — только факт запуска).
func HandleStatus() {
	if !cgiutil.IsGET() {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Метод не поддерживается"})
		return
	}
	inst := Status()
	cfg, _ := LoadConfig()
	cgiutil.WriteJSON(map[string]interface{}{
		"status":     "ok",
		"state":      inst.State,
		"pid":        inst.PID,
		"port":       inst.Port,
		"enabled":    cfg.Enabled,
		"proxy_port": cfg.ProxyPort,
	})
}

// HandleConfig: GET — чтение конфига (публично, без паролей в нём);
// POST — обновление конфига (только пароль + Origin).
func HandleConfig() {
	if cgiutil.IsPOST() {
		handleConfigPost()
		return
	}
	cfg, err := LoadConfig()
	if err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "Ошибка чтения конфига: " + err.Error()})
		return
	}
	cgiutil.WriteJSON(map[string]interface{}{
		"status":        "ok",
		"proxy_port":    cfg.ProxyPort,
		"proxy_host":    cfg.ProxyHost,
		"target_host":   cfg.TargetHost,
		"target_port":   cfg.TargetPort,
		"enabled":       cfg.Enabled,
		"allow_subnets": cfg.AllowSubnets,
	})
}

func handleConfigPost() {
	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "Запрос из недоверенного источника (CSRF)"})
		return
	}
	params := cgiutil.ParseFormBody(cgiutil.ReadPOSTBody())
	if !auth.CheckPassword(params["password"]) {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "Неверный пароль"})
		return
	}
	cfg, _ := LoadConfig()
	loadedPort := cfg.ProxyPort

	if v := params["target_host"]; v != "" {
		cfg.TargetHost = v
	}
	if v := params["target_port"]; v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "Некорректный порт цели"})
			return
		}
		cfg.TargetPort = p
	}
	if v := params["proxy_port"]; v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "Некорректный порт прокси"})
			return
		}
		if p != loadedPort && portIsBusy(p) {
			cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "Порт " + strconv.Itoa(p) + " занят другим процессом — выберите другой"})
			return
		}
		cfg.ProxyPort = p
	}
	if v := params["proxy_host"]; len(v) < 64 {
		cfg.ProxyHost = v
	}
	if v := params["allow_subnets"]; v != "" {
		cfg.AllowSubnets = splitCIDR(v)
	} else if _, ok := params["allow_subnets"]; ok {
		cfg.AllowSubnets = nil
	}
	if v := params["enabled"]; v == "true" || v == "false" {
		cfg.Enabled = v == "true"
	}

	portChanged := cfg.ProxyPort != loadedPort
	if err := SaveConfig(cfg); err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "Не удалось сохранить конфиг: " + err.Error()})
		return
	}
	if portChanged {
		applyProxyPortChange(cfg.ProxyPort)
	}
	cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "message": "Конфигурация сохранена"})
}
