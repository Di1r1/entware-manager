// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// bridge_manifest (GET ?id=)  — исходный текст манифеста для редактора
// bridge_save   (POST)        — валидация и сохранение {id, body, password}
// bridge_delete (POST)        — удаление {id, password}
package main

import (
	"entware-manager/internal/auth"
	"entware-manager/internal/bridge"
	"entware-manager/internal/cgiutil"
	"time"
)

func handleManifestGet() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}
	id := cgiutil.GetQueryParam("id")
	raw, found, err := bridge.GetManifestRaw(bridgeDirVarPath(), id)
	if err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	cgiutil.WriteJSON(map[string]interface{}{
		"status": "ok",
		"found":  found,
		"id":     id,
		"json":   raw,
	})
}

func handleManifestSave() {
	if !cgiutil.IsPOST() {
		cgiutil.NotAllowed()
		return
	}
	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteError(auth.CrossSiteDeny)
		return
	}
	params := cgiutil.ParseFormBody(cgiutil.ReadPOSTBody())
	id, rawJSON, password := params["id"], params["body"], params["password"]
	if id == "" || rawJSON == "" || password == "" {
		cgiutil.WriteError("нужны id, body и password")
		return
	}
	if !bridge.ValidID(id) {
		cgiutil.WriteError("плохой id")
		return
	}
	if !auth.CheckPassword(password) {
		time.Sleep(500 * time.Millisecond)
		cgiutil.WriteError("Неверный пароль")
		return
	}
	if err := bridge.SaveManifestFile(bridgeDirVarPath(), id, []byte(rawJSON)); err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	bridge.InvalidateCache()
	if m, err := bridge.LoadManifest(bridgeDirVarPath(), id); err == nil {
		if ws := bridge.ManifestWarnings(m); len(ws) > 0 {
			cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "warnings": ws})
			return
		}
	}
	cgiutil.WriteJSON(map[string]interface{}{"status": "ok"})
}

func handleManifestDelete() {
	if !cgiutil.IsPOST() {
		cgiutil.NotAllowed()
		return
	}
	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteError(auth.CrossSiteDeny)
		return
	}
	params := cgiutil.ParseFormBody(cgiutil.ReadPOSTBody())
	id, password := params["id"], params["password"]
	if id == "" || password == "" || !bridge.ValidID(id) {
		cgiutil.WriteError("нужны id и password")
		return
	}
	if !auth.CheckPassword(password) {
		time.Sleep(500 * time.Millisecond)
		cgiutil.WriteError("Неверный пароль")
		return
	}
	// Встроенные сервисы каталога удалять нельзя — их карточки не из файлов.
	// Override-манифест с каталог-id удаляется: после удаления каталог
	// восстановит карточку (HasManifestFile отличит этот кейс).
	if bridge.IsBuiltin(id) && !bridge.HasManifestFile(bridgeDirVarPath(), id) {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "встроенный модуль нельзя удалить"})
		return
	}
	if err := bridge.DeleteManifestFile(bridgeDirVarPath(), id); err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	// Вместе с манифестом удаляем секреты приложения и сохранённую cookie —
	// иначе логин/пароль останутся на диске и «оживут» при пересоздании модуля.
	bridge.DeleteAuthFile(bridgeDirVarPath(), id)
	bridge.ClearSession(bridgeDirVarPath(), id)
	bridge.InvalidateCache()
	cgiutil.WriteJSON(map[string]interface{}{"status": "ok"})
}
