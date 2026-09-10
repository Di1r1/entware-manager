package stats

import (
	"encoding/json"
	"strings"
	"testing"

	"entware-manager/internal/network"
	"entware-manager/internal/packages"
)

// Форма ответа dashboard.cgi: все 7 ключей всегда, провал подсбора — null.
func TestDashboardResponseShape(t *testing.T) {
	out, err := json.Marshal(dashboardResponse{})
	if err != nil {
		t.Fatalf("marshal dashboardResponse: %v", err)
	}
	s := string(out)
	for _, k := range []string{`"status"`, `"temperature"`, `"network_stats"`, `"update_check"`, `"upgradable"`, `"version"`, `"events"`} {
		if !strings.Contains(s, k+":null") {
			t.Errorf("key %s missing or not null in empty response: %s", k, s)
		}
	}
}

// mustDashboardJSON: значение — как есть, ошибка — null.
func TestMustDashboardJSON(t *testing.T) {
	if got := mustDashboardJSON(map[string]string{"a": "b"}); string(got) != `{"a":"b"}` {
		t.Errorf("mustDashboardJSON(map) = %s, want {\"a\":\"b\"}", got)
	}
	if got := mustDashboardJSON(nil); string(got) != "null" {
		t.Errorf("mustDashboardJSON(nil) = %s, want null", got)
	}
}

// Быстрые коллекторы без сети и тяжёлых сэмплов.
func TestDashboardFastCollectors(t *testing.T) {
	if b := CollectVersion(); !json.Valid(b) {
		t.Errorf("CollectVersion returned invalid JSON: %s", b)
	}
	if evs := network.CollectEvents("all", 30); evs == nil {
		t.Error("CollectEvents returned nil, want empty slice")
	}
	if pkgs := packages.CollectUpgradable(); pkgs == nil {
		t.Error("CollectUpgradable returned nil, want empty slice")
	}
}
