// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// entware-bridge — эндпоинты универсального моста сервисов:
//
//	bridge_discover (GET)   — сканирование loopback-сервисов (каталог+манифесты)
//	bridge_status?id= (GET) — прокси статуса приложения
//	bridge_action?id=&action= (POST) — выполнение действия (пароль панели,
//	                           Origin-чек, rate-limit; confirm-действия требуют
//	                           повторного ввода пароля)
package main

import (
	"os"
	"time"

	"entware-manager/internal/auth"
	"entware-manager/internal/bridge"
	_ "entware-manager/internal/buildinfo"
	"entware-manager/internal/cgiutil"
)

var bridgeDir = "/opt/web_entware/bridge"

func main() {
	switch os.Getenv("ENDPOINT") {
	case "bridge_discover":
		handleDiscover()
	case "bridge_status":
		handleStatus()
	case "bridge_action":
		handleAction()
	case "bridge_prefs":
		handlePrefs()
	case "bridge_stats":
		handleStats()
	case "bridge_auth":
		handleAuthSave()
	case "bridge_watch":
		handleWatch()
	default:
		cgiutil.WriteError("неизвестный эндпоинт")
	}
}

// handlePrefs — GET: текущие настройки; POST (password+Origin): {id, enabled, notifications}.
func handlePrefs() {
	if !cgiutil.IsGET() {
		if !cgiutil.IsPOST() {
			cgiutil.NotAllowed()
			return
		}
		if auth.IsCrossSiteOrigin() {
			cgiutil.WriteError(auth.CrossSiteDeny)
			return
		}
		params := cgiutil.ParseFormBody(cgiutil.ReadPOSTBody())
		id := params["id"]
		if id == "" || !isValidBridgeID(id) {
			cgiutil.WriteError("плохой id")
			return
		}
		// prefs не управляют приложениями (только мониторингом в мосте),
		// поэтому достаточно сессии панели + CSRF-чека выше.
		pf := bridge.LoadPrefs()
		m := pf.Modules[id]
		m.Enabled = params["enabled"] != "false"
		m.Notifications = params["notifications"] != "false"
		pf.Modules[id] = m
		if err := bridge.SavePrefs(pf); err != nil {
			cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "не удалось сохранить"})
			return
		}
		cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "id": id, "enabled": m.Enabled, "notifications": m.Notifications})
		return
	}
	// GET
	pf := bridge.LoadPrefs()
	out := map[string]bridge.ModulePrefs{}
	for id, m := range pf.Modules {
		out[id] = m
	}
	cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "modules": out})
}

func isValidBridgeID(id string) bool {
	return bridge.ValidID(id)
}

// handleWatch — мониторинг модулей для Telegram (вызывается шлюзом по curl,
// без сессии: только внутренние пробы и запись переходов в суточный лог).
func handleWatch() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}
	events := bridge.RunWatch(bridgeDirVarPath())
	cgiutil.WriteJSON(map[string]interface{}{
		"status":  "ok",
		"events":  len(events),
		"checked": time.Now().Format("15:04:05"),
	})
}

func handleDiscover() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}
	services := bridge.Discover(bridgeDirVarPath())
	cgiutil.WriteJSON(map[string]interface{}{
		"status":   "ok",
		"services": services,
	})
}

func handleStatus() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}
	id := cgiutil.GetQueryParam("id")
	if id == "" {
		cgiutil.WriteError("укажите id")
		return
	}
	var (
		sp  *bridge.StatusProxy
		err error
	)
	block := cgiutil.GetQueryParam("block")
	switch {
	case block == "" || block == "status":
		sp, err = bridge.ProxyStatus(bridgeDirVarPath(), id)
	case block == "stats":
		sp, err = bridge.ProxyStats(bridgeDirVarPath(), id)
	default:
		sp, err = bridge.ProxyExtra(bridgeDirVarPath(), id, block)
	}
	if err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "result": sp})
}

func handleAction() {
	if !cgiutil.IsPOST() {
		cgiutil.NotAllowed()
		return
	}
	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteError(auth.CrossSiteDeny)
		return
	}
	params := cgiutil.ParseFormBody(cgiutil.ReadPOSTBody())
	id, actionID := params["id"], params["action"]
	password := params["password"]
	if id == "" || actionID == "" || password == "" {
		cgiutil.WriteError("нужны id, action и password")
		return
	}

	m, err := bridge.LoadManifest(bridgeDirVarPath(), id)
	if err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}

	var act *bridge.Action
	for i := range m.Actions {
		if m.Actions[i].ID == actionID {
			act = &m.Actions[i]
			break
		}
	}
	if act == nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "действие не найдено"})
		return
	}

	if !auth.CheckPassword(password) {
		time.Sleep(500 * time.Millisecond)
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "Неверный пароль"})
		return
	}
	if !bridge.RateLimitAction(id, actionID, 2*time.Second) {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "Слишком часто — подождите пару секунд"})
		return
	}

	res, err := bridge.RunAction(m, act)
	if err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	cgiutil.WriteJSON(map[string]interface{}{
		"status": "ok",
		"result": res,
	})
}
