// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package stats

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"entware-manager/internal/cgiutil"
	"entware-manager/internal/monitor"
	"entware-manager/internal/network"
	"entware-manager/internal/packages"
)

const (
	dashboardEventsLimitDefault = 30
	dashboardEventsLimitMax     = 200
)

var dashboardAllowedKeys = []string{
	"status", "temperature", "network_stats", "update_check",
	"upgradable", "version", "events",
}

// dashboardResponse — пакетный ответ для тика NOC (один GET вместо ~7).
// Все ключи всегда присутствуют; провал подсбора — null.
// Форматы вложенностей — ровно те, что отдают исходные эндпоинты.
type dashboardResponse struct {
	Status       json.RawMessage `json:"status"`
	Temperature  json.RawMessage `json:"temperature"`
	NetworkStats json.RawMessage `json:"network_stats"`
	UpdateCheck  json.RawMessage `json:"update_check"`
	Upgradable   json.RawMessage `json:"upgradable"`
	Version      json.RawMessage `json:"version"`
	Events       json.RawMessage `json:"events"`
}

type dashboardError struct {
	Error   string   `json:"error"`
	Allowed []string `json:"allowed"`
}

func mustDashboardJSON(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

func allowedDashboardKey(k string) bool {
	for _, a := range dashboardAllowedKeys {
		if a == k {
			return true
		}
	}
	return false
}

func writeDashboardError(msg string, allowed []string) {
	fmt.Print("Status: 400 Bad Request\n")
	cgiutil.WriteJSON(dashboardError{Error: msg, Allowed: allowed})
}

// HandleDashboard склеивает 7 read-only подсборов в один JSON.
// Параметры запроса (QUERY_STRING):
//
//	keys=   — через запятую список полей (только запрошенные попадают в ответ);
//	events_limit= — лимит событий (по умолчанию 30, максимум 200).
//
// Без keys — полный ответ (байт-в-байт как раньше).
func HandleDashboard() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	eventsLimit := dashboardEventsLimitDefault
	if v := cgiutil.GetQueryParam("events_limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > dashboardEventsLimitMax {
			writeDashboardError("events_limit должен быть числом от 0 до 200", dashboardAllowedKeys)
			return
		}
		eventsLimit = n
	}

	var keys []string
	if v := cgiutil.GetQueryParam("keys"); v != "" {
		seen := make(map[string]bool)
		for _, part := range strings.Split(v, ",") {
			k := strings.TrimSpace(part)
			if k == "" {
				continue // пустые элементы игнорируются
			}
			if !allowedDashboardKey(k) {
				writeDashboardError("неизвестный ключ dashboard: "+k, dashboardAllowedKeys)
				return
			}
			if seen[k] {
				continue // дубликаты игнорируются
			}
			seen[k] = true
			keys = append(keys, k)
		}
	}

	status := mustDashboardJSON(monitor.CollectStatus())
	temperature := mustDashboardJSON(monitor.CollectTemperature())
	networkStats := mustDashboardJSON(network.CollectNetworkStats())
	updateCheck := mustDashboardJSON(CollectUpdateCheck())
	upgradable := mustDashboardJSON(packages.CollectUpgradable())
	version := json.RawMessage(CollectVersion())
	events := mustDashboardJSON(map[string][]network.LogEvent{"events": network.CollectEvents("all", eventsLimit)})

	// Без keys — полный ответ (байт-в-байт как раньше).
	if len(keys) == 0 {
		cgiutil.WriteJSON(dashboardResponse{
			Status:       status,
			Temperature:  temperature,
			NetworkStats: networkStats,
			UpdateCheck:  updateCheck,
			Upgradable:   upgradable,
			Version:      version,
			Events:       events,
		})
		return
	}

	// С keys — только запрошенные поля, без null-заглушек.
	filtered := make(map[string]json.RawMessage, len(keys))
	all := map[string]json.RawMessage{
		"status":        status,
		"temperature":   temperature,
		"network_stats": networkStats,
		"update_check":  updateCheck,
		"upgradable":    upgradable,
		"version":       version,
		"events":        events,
	}
	for _, k := range keys {
		filtered[k] = all[k]
	}
	cgiutil.WriteJSON(filtered)
}
