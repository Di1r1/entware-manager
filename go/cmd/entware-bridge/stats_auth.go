// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// bridge_stats  (GET, ?id=) — прокси блока статистики приложения.
// bridge_auth   (POST)     — сохранение учётных данных приложения из UI:
//
//	{id, password(панели), username, app_password}
//	→ пишет <id>.auth.json (0600) и чистит протухшую
//	сессию. Позволяет подключать защищённые сервисы
//	не заходя по SSH.
package main

import (
	"fmt"
	"time"

	"entware-manager/internal/auth"
	"entware-manager/internal/bridge"
	"entware-manager/internal/cgiutil"
)

func handleStats() {
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
	if cgiutil.GetQueryParam("block") == "status" {
		sp, err = bridge.ProxyStatus(bridgeDirVarPath(), id)
	} else {
		sp, err = bridge.ProxyStats(bridgeDirVarPath(), id)
	}
	if err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "result": sp})
}

func handleAuthSave() {
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
	password := params["password"]
	if id == "" || !bridge.ValidID(id) || password == "" {
		cgiutil.WriteError("нужны id и password панели")
		return
	}
	if !auth.CheckPassword(password) {
		time.Sleep(500 * time.Millisecond)
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "Неверный пароль"})
		return
	}

	// Сброс авторизации: удаляем сохранённые секреты (<id>.auth.json) и
	// сессию приложения (/tmp/entware/bridge/<id>.session). Идемпотентен.
	if params["clear"] == "1" {
		bridge.DeleteAuthFile(bridgeDirVarPath(), id)
		bridge.ClearSession(bridgeDirVarPath(), id)
		cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "message": "Учётные данные «" + id + "» сброшены"})
		return
	}

	credType := params["cred_type"]
	if credType != "basic" && credType != "cookie_login" && credType != "api_key" {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "cred_type должен быть basic или cookie_login"})
		return
	}
	if err := bridge.ValidateAuthCreds(credType, params["login_url"]); err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	creds := bridge.AuthCreds{
		Type:     credType,
		Username: params["username"],
		Password: params["app_password"],
		LoginURL: params["login_url"],
	}
	if err := bridge.SaveAuth(bridgeDirVarPath(), id, creds); err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "не удалось сохранить"})
		return
	}
	bridge.ClearSession(bridgeDirVarPath(), id)
	cgiutil.WriteJSON(map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Учётные данные «%s» сохранены — попробуйте обновить статус", id),
	})
}

func bridgeDirVarPath() string {
	return bridge.BridgeDir()
}
