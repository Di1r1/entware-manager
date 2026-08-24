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
	default:
		cgiutil.WriteError("неизвестный эндпоинт")
	}
}

func handleDiscover() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}
	services := bridge.Discover(bridgeDir)
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
	sp, err := bridge.ProxyStatus(bridgeDir, id)
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

	m, err := bridge.LoadManifest(bridgeDir, id)
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
