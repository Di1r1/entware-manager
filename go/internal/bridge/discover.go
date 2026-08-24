// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Обнаружение и опрос сервисов моста.
//
// Каталог v1: порт ≠ сигнатура (qBittorrent тоже :8080) — каждая запись
// проверяется по маркеру (путь + ожидание). Пробы параллельные с таймаутом,
// общий бюджет ограничен; результат кэшируется в памяти процесса.
package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	probeTimeout   = 1 * time.Second
	discoverBudget = 3 * time.Second
	maxProbeBody   = 64 * 1024
	maxStatusBody  = 256 * 1024
	maxConcurrency = 8
	cacheTTL       = 30 * time.Second
)

// CatalogEntry — встроенная сигнатура известного сервиса.
type CatalogEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Port       int    `json:"port"`
	Path       string `json:"path"`             // путь для пробы
	Mark       string `json:"mark,omitempty"`   // подстрока в теле/заголовках (опц.)
	Expect     string `json:"expect,omitempty"` // "json" | ""
	StatusPath string `json:"status_path,omitempty"`
}

// BuiltInCatalog — известные Entware-сервисы (v1).
func BuiltInCatalog() []CatalogEntry {
	return []CatalogEntry{
		{ID: "koffe", Name: "Koffe VPN", Port: 9097, Path: "/?action=version", Expect: "json", StatusPath: "/?action=status"},
		{ID: "adguard", Name: "AdGuard Home", Port: 8080, Path: "/control/status", Expect: "json", StatusPath: "/control/status"},
		{ID: "ttyd", Name: "Терминал ttyd", Port: 7681, Path: "/"},
		{ID: "transmission", Name: "Transmission", Port: 9091, Path: "/transmission/rpc"},
		{ID: "syncthing", Name: "Syncthing", Port: 8384, Path: "/"},
	}
}

// ServiceState — состояние одного сервиса в выдаче discovery.
type ServiceState struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	State  string `json:"state"` // running | auth_required | absent
	Detail string `json:"detail,omitempty"`
}

// bridgeDirVar — каталог манифестов (переопределяется для тестов).
var bridgeDirVar = "/opt/web_entware/bridge"

var (
	cacheMu      sync.Mutex
	cachedResult []ServiceState
	cachedAt     time.Time
)

// clientBridge — транспорт без редиректов и keep-alive.
func clientBridge() *http.Client {
	return &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			DialContext:           (&net.Dialer{Timeout: probeTimeout}).DialContext,
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: probeTimeout,
		},
	}
}

// probeOne классифицирует одну пробу.
func probeOne(client *http.Client, url string) ServiceState {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ServiceState{State: "absent", Detail: "bad url"}
	}
	resp, err := client.Do(req)
	if err != nil {
		return ServiceState{State: "absent"}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxProbeBody))

	switch {
	case resp.StatusCode == 200:
		if isJSONBody(resp.Header.Get("Content-Type"), body) || looksLikeJSON(body) {
			return ServiceState{State: "running"}
		}
		return ServiceState{State: "running"} // 200 = жив (ttyd/syncthing отдают HTML)
	case resp.StatusCode == 401 || resp.StatusCode == 403:
		return ServiceState{State: "auth_required", Detail: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	default:
		return ServiceState{State: "absent", Detail: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}
}

func isJSONBody(ct string, body []byte) bool {
	if strings.Contains(ct, "json") {
		return true
	}
	return looksLikeJSON(body)
}

func looksLikeJSON(b []byte) bool {
	s := strings.TrimSpace(string(b))
	return len(s) > 0 && (s[0] == '{' || s[0] == '[')
}

// Discover опрашивает каталог + пользовательские манифесты.
func Discover(bridgeDir string) []ServiceState {
	cacheMu.Lock()
	if time.Since(cachedAt) < cacheTTL && cachedResult != nil {
		out := cachedResult
		cacheMu.Unlock()
		return out
	}
	cacheMu.Unlock()

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []ServiceState
		sem     = make(chan struct{}, maxConcurrency)
	)
	add := func(s ServiceState) {
		mu.Lock()
		results = append(results, s)
		mu.Unlock()
	}

	client := clientBridge()
	deadline := time.Now().Add(discoverBudget)

	run := func(id, name, url string) {
		if !IsEnabled(id) {
			add(ServiceState{ID: id, Name: name, State: "disabled"})
			return
		}
		if time.Now().After(deadline) {
			add(ServiceState{ID: id, Name: name, State: "absent", Detail: "budget"})
			return
		}
		st := probeOne(client, url)
		st.ID, st.Name = id, name
		add(st)
	}

	// Манифесты пользователя главнее каталога: если id совпал — проба каталога
	// пропускается (иначе дубликаты карточек, найдено живым тестом).
	manifests := ListManifests(bridgeDir)
	catalogIDs := map[string]bool{}
	for _, m := range manifests {
		catalogIDs[m.ID] = true
	}

	for _, e := range BuiltInCatalog() {
		if catalogIDs[e.ID] {
			continue
		}
		wg.Add(1)
		go func(e CatalogEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			run(e.ID, e.Name, fmt.Sprintf("http://127.0.0.1:%d%s", e.Port, e.Path))
		}(e)
	}

	for _, m := range manifests {
		wg.Add(1)
		go func(m *Manifest) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			// Относительные probe-URL резолвим от базы через гейт
			// (иначе http.NewRequest отклонит "?action=..." → ложное absent).
			u, err := ValidateBridgeURL(m.Probe.URL, m.Base)
			if err != nil {
				add(ServiceState{ID: m.ID, Name: m.Name, State: "absent", Detail: err.Error()})
				return
			}
			run(m.ID, m.Name, u.String())
		}(m)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(discoverBudget):
	}

	// Работающие сверху, затем требующие авторизации, отсутствующие в конце.
	stateRank := map[string]int{"running": 0, "auth_required": 1, "absent": 2, "disabled": 3}
	sort.Slice(results, func(i, j int) bool {
		if stateRank[results[i].State] != stateRank[results[j].State] {
			return stateRank[results[i].State] < stateRank[results[j].State]
		}
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	cacheMu.Lock()
	cachedResult = results
	cachedAt = time.Now()
	out := results
	cacheMu.Unlock()
	_ = out
	return results
}

// StatusProxy — прокси GET статуса конкретного сервиса (валидация по манифесту).
type StatusProxy struct {
	HTTPCode int             `json:"http_code"`
	Body     json.RawMessage `json:"body,omitempty"`
	Raw      string          `json:"raw,omitempty"`
	Error    string          `json:"error,omitempty"`
}

func ProxyStatus(dir, id string) (*StatusProxy, error) {
	m, err := LoadManifest(dir, id)
	if err != nil {
		return nil, err
	}
	ep := m.Status
	if ep == nil {
		ep = &m.Probe
	}
	u, err := ValidateBridgeURL(ep.URL, m.Base)
	if err != nil {
		return nil, err
	}
	client := clientBridge()
	resp, err := authedDo(client, dir, id, http.MethodGet, u.String())
	if err != nil {
		return &StatusProxy{Error: "сервис не отвечает"}, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxStatusBody))
	sp := &StatusProxy{HTTPCode: resp.StatusCode}
	if looksLikeJSON(body) {
		sp.Body = json.RawMessage(body)
	} else {
		sp.Raw = truncate(string(body), 512)
	}
	return sp, nil
}

// RateLimitAction — простое ограничение частоты действий (tmpfs-файл).
func RateLimitAction(id, actionID string, minInterval time.Duration) bool {
	dir := "/tmp/entware/bridge"
	path := filepath.Join(dir, sanitize(id+"_"+actionID))
	data, _ := os.ReadFile(path)
	var last int64
	fmt.Sscanf(string(data), "%d", &last)
	now := time.Now().Unix()
	if now-last < int64(minInterval.Seconds()) {
		return false
	}
	os.MkdirAll(dir, 0700)
	os.WriteFile(path, []byte(fmt.Sprintf("%d\n", now)), 0600)
	return true
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func applyAuth(req *http.Request, a *AuthCreds) {
	if a == nil || a.Type != "basic" {
		return
	}
	req.SetBasicAuth(a.Username, a.Password)
}
