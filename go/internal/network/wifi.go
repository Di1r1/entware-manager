// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
// Wi-Fi клиенты: таблица подключённых устройств из RCI Keenetic.
package network

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"time"

	"entware-manager/internal/cgiutil"
)

// wifiClient — подключённый Wi-Fi клиент (полные данные из RCI).
type wifiClient struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	SSID     string `json:"ssid"`
	AP       string `json:"ap"`
	RSSI     int    `json:"rssi"`
	Mode     string `json:"mode"`
	TXRate   int    `json:"txrate"`
	Segment  string `json:"segment"`
}

// rciWifiHost — полная запись устройства из RCI hotspot/host.
type rciWifiHost struct {
	MAC       string `json:"mac"`
	IP        string `json:"ip"`
	Hostname  string `json:"hostname"`
	Name      string `json:"name"`
	SSID      string `json:"ssid"`
	AP        string `json:"ap"`
	Mode      string `json:"mode"`
	RSSI      int    `json:"rssi"`
	TXRate    int    `json:"txrate"`
	Link      string `json:"link"`
	Active    bool   `json:"active"`
	Interface struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"interface"`
}

// HandleWiFi — GET /network/wifi.cgi: список Wi-Fi клиентов с телеметрией.
func HandleWiFi() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(rciBase + "/rci/show/ip/hotspot/host")
	if err != nil {
		cgiutil.WriteError("RCI недоступен: " + err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		cgiutil.WriteError("Ошибка чтения ответа RCI")
		return
	}

	var hosts []rciWifiHost
	if json.Unmarshal(body, &hosts) != nil {
		cgiutil.WriteError("Ошибка парсинга ответа RCI")
		return
	}

	var clients []wifiClient
	for _, h := range hosts {
		if h.SSID == "" && h.AP == "" {
			continue // не Wi-Fi клиент (проводной или офлайн-запись)
		}
		name := h.Name
		if name == "" {
			name = h.Hostname
		}
		seg := h.Interface.Name
		if seg == "" {
			seg = h.Interface.ID
		}
		clients = append(clients, wifiClient{
			Name:     name,
			Hostname: h.Hostname,
			IP:       h.IP,
			MAC:      h.MAC,
			SSID:     h.SSID,
			AP:       h.AP,
			RSSI:     h.RSSI,
			Mode:     h.Mode,
			TXRate:   h.TXRate,
			Segment:  seg,
		})
	}

	// Сортировка: сильнейший сигнал первым.
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].RSSI > clients[j].RSSI
	})

	cgiutil.WriteJSON(map[string]interface{}{
		"status":  "ok",
		"clients": clients,
		"total":   len(clients),
	})
}
