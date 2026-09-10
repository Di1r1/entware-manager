// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package stats

import (
	"encoding/json"

	"entware-manager/internal/cgiutil"
	"entware-manager/internal/monitor"
	"entware-manager/internal/network"
	"entware-manager/internal/packages"
)

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

func mustDashboardJSON(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

// HandleDashboard склеивает 7 read-only подсборов в один JSON.
// Параметров наружу нет: events всегда limit=30&tag=all.
func HandleDashboard() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	cgiutil.WriteJSON(dashboardResponse{
		Status:       mustDashboardJSON(monitor.CollectStatus()),
		Temperature:  mustDashboardJSON(monitor.CollectTemperature()),
		NetworkStats: mustDashboardJSON(network.CollectNetworkStats()),
		UpdateCheck:  mustDashboardJSON(CollectUpdateCheck()),
		Upgradable:   mustDashboardJSON(packages.CollectUpgradable()),
		Version:      json.RawMessage(CollectVersion()),
		Events:       mustDashboardJSON(map[string][]network.LogEvent{"events": network.CollectEvents("all", 30)}),
	})
}
