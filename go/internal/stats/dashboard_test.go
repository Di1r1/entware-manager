package stats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"

	"entware-manager/internal/cache"
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

// captureDashboard запускает HandleDashboard с заданным QUERY_STRING,
// перехватывая stdout (CGI-контракт), и возвращает сырой вывод.
func captureDashboard(t *testing.T, query string) string {
	t.Helper()
	oldQuery, oldQuerySet := os.LookupEnv("QUERY_STRING")
	oldMethod, oldMethodSet := os.LookupEnv("REQUEST_METHOD")
	if query == "" {
		os.Unsetenv("QUERY_STRING")
	} else {
		os.Setenv("QUERY_STRING", query)
	}
	os.Setenv("REQUEST_METHOD", "GET")
	t.Cleanup(func() {
		if oldQuerySet {
			os.Setenv("QUERY_STRING", oldQuery)
		} else {
			os.Unsetenv("QUERY_STRING")
		}
		if oldMethodSet {
			os.Setenv("REQUEST_METHOD", oldMethod)
		} else {
			os.Unsetenv("REQUEST_METHOD")
		}
	})

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	HandleDashboard()
	w.Close()
	var out bytes.Buffer
	io.Copy(&out, r)
	os.Stdout = old
	return out.String()
}

func dashboardJSONBody(out string) string {
	idx := strings.Index(out, "\n\n")
	if idx < 0 {
		return out
	}
	return out[idx+2:]
}

// TestDashboardQueryParams проверяет обработку параметров keys= и events_limit=.
func TestDashboardQueryParams(t *testing.T) {
	// Герметичный кэш: CollectUpdateCheck не должен ходить в GitHub API.
	cache.SetBaseDir(t.TempDir())
	if err := cache.Put("update_check", []byte(`{"latest":"1.0.0","error":""}`)); err != nil {
		t.Fatalf("cache.Put: %v", err)
	}

	fullKeys := []string{"status", "temperature", "network_stats", "update_check", "upgradable", "version", "events"}
	sort.Strings(fullKeys)

	tests := []struct {
		name     string
		query    string
		wantCode int
		wantKeys []string // ожидаемые ключи ответа (отсортированные)
		wantErr  bool
	}{
		{"no keys -> full response", "", http.StatusOK, fullKeys, false},
		{"one key", "keys=status", http.StatusOK, []string{"status"}, false},
		{"multiple keys", "keys=temperature,version", http.StatusOK, []string{"temperature", "version"}, false},
		{"unknown key -> 400", "keys=foobar", http.StatusBadRequest, nil, true},
		{"events_limit>200 -> 400", "events_limit=250", http.StatusBadRequest, nil, true},
		{"non-numeric events_limit -> 400", "events_limit=abc", http.StatusBadRequest, nil, true},
		{"duplicate keys ignored", "keys=status, status ,status", http.StatusOK, []string{"status"}, false},
		{"events with limit", "keys=events&events_limit=10", http.StatusOK, []string{"events"}, false},
		{"events_limit=0 with keys=events", "keys=events&events_limit=0", http.StatusOK, []string{"events"}, false},
		{"events_limit negative -> 400", "events_limit=-5", http.StatusBadRequest, nil, true},
		{"keys empty parts ignored", "keys=status,,", http.StatusOK, []string{"status"}, false},
		{"keys with spaces trimmed", "keys= status , temperature", http.StatusOK, []string{"status", "temperature"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureDashboard(t, tt.query)

			if tt.wantErr {
				if !strings.Contains(out, fmt.Sprintf("Status: %d", tt.wantCode)) {
					t.Errorf("ожидался Status: %d, получено:\n%s", tt.wantCode, dashboardFirstLines(out, 6))
				}
				if !strings.Contains(out, `"error"`) {
					t.Errorf("ожидался ключ error в ответе 400:\n%s", dashboardFirstLines(out, 6))
				}
				if !strings.Contains(out, `"allowed"`) {
					t.Errorf("ожидался ключ allowed в ответе 400:\n%s", dashboardFirstLines(out, 6))
				}
				return
			}

			body := dashboardJSONBody(out)
			var m map[string]json.RawMessage
			if err := json.Unmarshal([]byte(body), &m); err != nil {
				t.Fatalf("ответ не является JSON: %v, тело: %s", err, body)
			}
			got := dashboardSortedKeys(m)
			if !dashboardStringSliceEqual(got, tt.wantKeys) {
				t.Errorf("ключи ответа = %v, want %v (query=%q)", got, tt.wantKeys, tt.query)
			}
		})
	}
}

func dashboardFirstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

func dashboardSortedKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func dashboardStringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
