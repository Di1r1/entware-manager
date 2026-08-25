// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// bridge_probe (POST) — сканер манифеста для редактора: опрашивает источники
// данных из текста манифеста (не обязательно сохранённого) и возвращает
// дерево путей JSON-ответов. Чтение loopback — пароль панели не нужен,
// достаточно сессии (гейт в entware-server/go.cgi) + Origin-чека.
package main

import (
	"time"

	"entware-manager/internal/auth"
	"entware-manager/internal/bridge"
	"entware-manager/internal/cgiutil"
)

const probeMinInterval = time.Second

func handleProbe() {
	if !cgiutil.IsPOST() {
		cgiutil.NotAllowed()
		return
	}
	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteError(auth.CrossSiteDeny)
		return
	}
	params := cgiutil.ParseFormBody(cgiutil.ReadPOSTBody())
	raw := params["body"]
	if raw == "" {
		cgiutil.WriteError("пустое тело манифеста")
		return
	}
	if !bridge.RateLimitAction("*probe*", "scan", probeMinInterval) {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "Слишком часто — подождите секунду"})
		return
	}
	res := bridge.ProbeManifestData([]byte(raw))
	cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "probe": res})
}
