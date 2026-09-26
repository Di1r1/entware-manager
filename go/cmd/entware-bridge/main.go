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
	"fmt"
	"os"
	"strings"
	"time"

	"entware-manager/internal/auth"
	"entware-manager/internal/bridge"
	_ "entware-manager/internal/buildinfo"
	"entware-manager/internal/cgiutil"
)

// init — единый источник каталога моста: env EWM_BRIDGE_DIR (тесты/оверрайд)
// или значение по умолчанию из internal/bridge (BridgeDir()).
func init() {
	if d := os.Getenv("EWM_BRIDGE_DIR"); d != "" {
		bridge.SetBridgeDir(d)
	}
}

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
	case "bridge_card":
		handleCard()
	case "bridge_manifest":
		handleManifestGet()
	case "bridge_save":
		handleManifestSave()
	case "bridge_delete":
		handleManifestDelete()
	case "bridge_probe":
		handleProbe()
	case "bridge_processes":
		handleProcesses()
	case "bridge_ctl":
		handleCtl()
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
		if v, ok := params["control"]; ok && v != "" {
			m.Control = v == "true" // отсутствует в запросе → сохраняем прежнее
		}
		pf.Modules[id] = m
		if err := bridge.SavePrefs(pf); err != nil {
			cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "не удалось сохранить"})
			return
		}
		cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "id": id, "enabled": m.Enabled, "notifications": m.Notifications, "control": m.Control})
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

// handleCard — универсальный сканер карточки: все JSON-эндпоинты манифеста
// + значения полей по путям из manifest.fields.
func handleCard() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}
	id := cgiutil.GetQueryParam("id")
	if id == "" {
		cgiutil.WriteError("укажите id")
		return
	}
	card, err := bridge.BuildCard(bridgeDirVarPath(), id)
	if err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	state := bridge.DiscoverState(bridgeDirVarPath(), id)
	cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "state": state, "card": card})
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

// handleCtl — POST: управление сервисом модуля через init.d
// {id, op=start|stop|restart, password}. Рубежи: пароль панели + Origin,
// rate-limit, manifest.init задан, prefs.control включён (галочка
// «управление»), скрипт найден в /opt/etc/init.d.
func handleCtl() {
	if !cgiutil.IsPOST() {
		cgiutil.NotAllowed()
		return
	}
	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteError(auth.CrossSiteDeny)
		return
	}
	params := cgiutil.ParseFormBody(cgiutil.ReadPOSTBody())
	id, op := params["id"], params["op"]
	password := params["password"]
	if id == "" || !bridge.ValidID(id) {
		cgiutil.WriteError("нужны id и op")
		return
	}
	switch op {
	case "start", "stop", "restart":
	default:
		cgiutil.WriteError("недопустимое действие: " + op)
		return
	}
	if auth.Enabled() && !auth.SessionValid() {
		if password == "" || !auth.CheckPassword(password) {
			time.Sleep(500 * time.Millisecond)
			cgiutil.WriteError("требуется авторизация (войдите в панель)")
			return
		}
	}
	if !bridge.RateLimitAction(id, "ctl_"+op, 3*time.Second) {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "Слишком часто — подождите пару секунд"})
		return
	}
	m, err := bridge.LoadManifest(bridgeDirVarPath(), id)
	if err != nil {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	if m.Init == "" {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "модуль не задаёт init — управление недоступно"})
		return
	}
	if !bridge.ControlAllowed(id) {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "включите галочку «управление» на карточке модуля (вкладка Модули)"})
		return
	}
	script := bridge.FindInitScript(m.Init)
	if script == "" {
		cgiutil.WriteJSON(map[string]interface{}{"status": "error", "message": "скрипт " + m.Init + " не найден в /opt/etc/init.d"})
		return
	}
	out, execErr := bridge.RunInitAction(script, op)
	if execErr != nil {
		cgiutil.WriteJSON(map[string]interface{}{
			"status":  "error",
			"message": "ошибка выполнения: " + execErr.Error(),
			"output":  out,
		})
		return
	}
	bridge.AppendDailyLog(fmt.Sprintf("[%s] [INFO] [127.0.0.1] [%d] [bridge] Модуль «%s»: выполнен %s через init.d",
		time.Now().Format("2006-01-02 15:04:05"), os.Getpid(), m.Name, op))
	cgiutil.WriteJSON(map[string]interface{}{
		"status": "ok",
		"op":     op,
		"output": out,
	})
}

func handleDiscover() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}
	services := bridge.Discover(bridgeDirVarPath())
	cgiutil.WriteJSON(map[string]interface{}{
		"status":      "ok",
		"services":    services,
		"port_labels": bridge.PortLabelsDict(),
	})
}

// handleProcesses — GET: живые процессы роутера для сканера манифеста.
// Чтение /proc — пароль не нужен, достаточно сессии (гейт go.cgi/server).
func handleProcesses() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}
	cgiutil.WriteJSON(map[string]interface{}{
		"status":    "ok",
		"processes": bridge.ListProcesses(),
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
	// Недоступный сервис и HTTP >= 400 лежат в StatusProxy, а не в err: без
	// actionVerdict мёртвый модуль отчитывался «действие выполнено» в лог и в
	// Telegram, а UI получал status:"ok" (P1-08).
	status, message := actionVerdict(res)
	if status != "ok" {
		// message уже начинается с «не выполнено:» — свой префикс в шаблоне
		// давал бы «не выполнено — не выполнено: HTTP 500».
		bridge.AppendDailyLog(fmt.Sprintf("[%s] [WARN] [127.0.0.1] [%d] [bridge] Модуль «%s»: действие «%s» — %s",
			time.Now().Format("2006-01-02 15:04:05"), os.Getpid(), logSafe(m.Name), logSafe(act.Label), message))
		cgiutil.WriteJSON(map[string]interface{}{
			"status":  status,
			"message": message,
			"result":  res,
		})
		return
	}
	bridge.AppendDailyLog(fmt.Sprintf("[%s] [INFO] [127.0.0.1] [%d] [bridge] Модуль «%s»: действие «%s» выполнено",
		time.Now().Format("2006-01-02 15:04:05"), os.Getpid(), logSafe(m.Name), logSafe(act.Label)))
	cgiutil.WriteJSON(map[string]interface{}{
		"status": "ok",
		"result": res,
	})
}

// actionVerdict — вердикт по результату действия: что показать в UI и что
// записать в лог моста. Успешный путь (2xx) не меняется: раньше он тоже
// отдавал status:"ok" с тем же result.
//
// nil трактуется как сбой, а не как успех (fail-closed, как в auth):
// RunAction при err==nil всегда отдаёт не-nil, но «нет результата» не должно
// никогда читаться как «сделано».
func actionVerdict(res *bridge.StatusProxy) (status, message string) {
	if res == nil {
		return "error", "не выполнено: нет результата от сервиса"
	}
	if res.Failed() {
		return "error", "не выполнено: " + res.FailMessage()
	}
	return "ok", ""
}

// logSafe — одна строка лога. Имя модуля и подпись действия приходят из
// манифеста, а ValidateManifest проверяет только длину: перевод строки
// прошёл бы и дописал в суточный лог поддельную запись (например, «выполнено»).
var logReplacer = strings.NewReplacer("\r", " ", "\n", " ")

func logSafe(s string) string { return logReplacer.Replace(s) }
