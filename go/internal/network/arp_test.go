package network

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchRCIHostnames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[
			{"ip": "192.168.3.97", "hostname": "localhost", "name": "TV Samsung"},
			{"ip": "192.168.3.134", "hostname": "cuco-plug-cp5d-miap3E3D", "name": "cuco-plug-cp5d-miap3E3D - Home network - 2026-07-07 01:17"},
			{"ip": "192.168.3.78", "hostname": "chuangmi-camera-021a04", "name": ""},
			{"hostname": "no-ip-device", "name": "без IP"}
		]`))
	}))
	defer server.Close()

	orig := rciHotspotURL
	rciHotspotURL = server.URL
	defer func() { rciHotspotURL = orig }()

	names := fetchRCIHostnames()

	// name приоритетнее hostname
	if got := names["192.168.3.97"]; got != "TV Samsung" {
		t.Errorf("name для .97 = %q, want TV Samsung", got)
	}
	// name с суффиксом берётся как есть
	if got := names["192.168.3.134"]; got != "cuco-plug-cp5d-miap3E3D - Home network - 2026-07-07 01:17" {
		t.Errorf("name для .134 = %q", got)
	}
	// пустой name → hostname
	if got := names["192.168.3.78"]; got != "chuangmi-camera-021a04" {
		t.Errorf("hostname fallback для .78 = %q, want chuangmi-camera-021a04", got)
	}
	// запись без ip отфильтрована
	if len(names) != 3 {
		t.Errorf("len(names) = %d, want 3", len(names))
	}
}

func TestFetchRCIHostnames_Error(t *testing.T) {
	// RCI недоступен — пустая map без паники
	orig := rciHotspotURL
	rciHotspotURL = "http://127.0.0.1:1/rci/show/ip/hotspot/host"
	defer func() { rciHotspotURL = orig }()

	names := fetchRCIHostnames()
	if len(names) != 0 {
		t.Errorf("при ошибке RCI ожидали пустую map, got %v", names)
	}
}
