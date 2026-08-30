// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Обнаружение и опрос сервисов моста.
//
// Каталог v1: порт ≠ сигнатура (qBittorrent тоже :8080) — каждая запись
// проверяется по маркеру (путь + ожидание). Пробы параллельные с таймаутом,
// общий бюджет ограничен; результат кэшируется в памяти процесса.
package bridge

import (
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
	maxExtraBody   = 1024 * 1024
	maxConcurrency = 8
	cacheTTL       = 30 * time.Second
)

// CatalogEntry — встроенная сигнатура известного сервиса.
type CatalogEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Ports      []int  `json:"ports"`            // кандидаты: native-прошивка и Entware могут слушать разные порты
	Path       string `json:"path"`             // путь для пробы
	Mark       string `json:"mark,omitempty"`   // подстрока в теле/заголовках (опц.)
	Expect     string `json:"expect,omitempty"` // "json" | ""
	StatusPath string `json:"status_path,omitempty"`
}

// BuiltInCatalog — известные Entware-сервисы (v1).
func BuiltInCatalog() []CatalogEntry {
	return []CatalogEntry{
		{ID: "koffe", Name: "Koffe VPN", Ports: []int{9097}, Path: "/?action=version", Expect: "json", StatusPath: "/?action=status"},
		{ID: "adguard", Name: "AdGuard Home", Ports: []int{8080}, Path: "/control/status", Expect: "json", StatusPath: "/control/status"},
		{ID: "ttyd", Name: "Терминал ttyd", Ports: []int{7681}, Path: "/"},
		{ID: "transmission", Name: "Transmission", Ports: []int{8090, 9091}, Path: "/transmission/rpc", StatusPath: "/transmission/rpc"},
		{ID: "syncthing", Name: "Syncthing", Ports: []int{8384}, Path: "/"},
		{ID: "netdata", Name: "Netdata", Ports: []int{19999}, Path: "/api/v1/info", Expect: "json", StatusPath: "/api/v1/info"},
	}
}

// ServiceState — состояние одного сервиса в выдаче discovery.
type ServiceState struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	State       string `json:"state"` // running | auth_required | absent | disabled
	Detail      string `json:"detail,omitempty"`
	HasManifest bool   `json:"has_manifest,omitempty"` // карточка из файла-манифеста (можно удалить)
	CanCtl      bool   `json:"can_ctl,omitempty"`      // манифест задаёт init → доступны кнопки управления
}

// bridgeDirVar — каталог манифестов (переопределяется для тестов).
var bridgeDirVar = "/opt/web_entware/bridge"

var (
	cacheMu      sync.Mutex
	cachedResult []ServiceState
	cachedAt     time.Time
)

// InvalidateCache сбрасывает кэш Discover — вызывается после сохранения или
// удаления манифеста, чтобы свежий модуль сразу появился/исчез на вкладке.
func InvalidateCache() {
	cacheMu.Lock()
	cachedResult = nil
	cachedAt = time.Time{}
	cacheMu.Unlock()
}

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

// classify определяет состояние по ответу пробы.
// Любой осмысленный ответ HTTP = сервис жив. 401/403/409 выделяются как
// «нужна авторизация», 405 — эндпоинт существует, но метод не тот
// (Transmission RPC принимает только POST).
func classify(code int, contentType string, body []byte) ServiceState {
	switch {
	case code >= 200 && code < 400:
		return ServiceState{State: "running"}
	case code == 401 || code == 403 || code == 409:
		return ServiceState{State: "auth_required", Detail: fmt.Sprintf("HTTP %d", code)}
	case code == 405:
		// Осознанная эвристика: чужой сервис на порту-кандидате, отвечающий
		// 405 на путь Transmission, будет показан как Transmission. Для
		// домашнего роутера риск принят (порты проверяются в порядке
		// специфичности). См. кворум v1.15.5 MINOR-2.
		return ServiceState{State: "running"}
	default:
		return ServiceState{State: "absent", Detail: fmt.Sprintf("HTTP %d", code)}
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

	// Манифесты пользователя главнее каталога: если id совпал — проба каталога
	// пропускается (иначе дубликаты карточек, найдено живым тестом).
	manifests := ListManifests(bridgeDir)
	catalogIDs := map[string]bool{}
	for _, m := range manifests {
		catalogIDs[m.ID] = true
	}

	// Один снимок процессов на весь проход (P2 кворума v1.15.7): горутины
	// манифестов с process-детектом читают готовый слайс.
	var procSnap []procEntry
	for _, m := range manifests {
		if len(m.Process) > 0 {
			procSnap = snapshotProcs()
			break
		}
	}

	for _, e := range BuiltInCatalog() {
		if catalogIDs[e.ID] {
			continue
		}
		if !IsEnabled(e.ID) {
			add(ServiceState{ID: e.ID, Name: e.Name, State: "disabled"})
			continue
		}
		if time.Now().After(deadline) {
			add(ServiceState{ID: e.ID, Name: e.Name, State: "absent", Detail: "budget"})
			continue
		}
		wg.Add(1)
		go func(e CatalogEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			// Пробуем порты-кандидаты по порядку: первый ответивший
			// (running/auth_required) выигрывает; все absent → absent.
			var best ServiceState
			for _, port := range e.Ports {
				resp, err := authedDo(client, bridgeDirVar, e.ID, http.MethodGet,
					fmt.Sprintf("http://127.0.0.1:%d%s", port, e.Path), "")
				if err != nil {
					best = ServiceState{State: "absent"}
					continue
				}
				body, _ := io.ReadAll(io.LimitReader(resp.Body, maxProbeBody))
				resp.Body.Close()
				best = classify(resp.StatusCode, resp.Header.Get("Content-Type"), body)
				if best.State != "absent" {
					break
				}
			}
			best.ID, best.Name = e.ID, e.Name
			add(best)
		}(e)
	}

	for _, m := range manifests {
		wg.Add(1)
		go func(m *Manifest) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if !IsEnabled(m.ID) {
				add(ServiceState{ID: m.ID, Name: m.Name, State: "disabled", HasManifest: true, CanCtl: m.Init != ""})
				return
			}
			// process-детект: процесс = источник истины, probe игнорируется
			// полностью (кворум v1.15.7). auth_required у таких модулей не бывает.
			if len(m.Process) > 0 {
				st := ServiceState{ID: m.ID, Name: m.Name, HasManifest: true, CanCtl: m.Init != ""}
				if pids := matchProcs(procSnap, m.Process); len(pids) > 0 {
					st.State = "running"
					st.Detail = fmt.Sprintf("PID %d", pids[0])
					if len(pids) > 1 {
						st.Detail += fmt.Sprintf(" (+%d)", len(pids)-1)
					}
				} else {
					st.State = "absent"
					st.Detail = "процесс не найден"
				}
				add(st)
				return
			}
			if time.Now().After(deadline) {
				add(ServiceState{ID: m.ID, Name: m.Name, State: "absent", Detail: "budget", HasManifest: true})
				return
			}
			// Относительные probe-URL резолвим от базы через гейт
			// (иначе http.NewRequest отклонит "?action=..." → ложное absent).
			u, err := ValidateBridgeURL(m.Probe.URL, m.Base)
			if err != nil {
				add(ServiceState{ID: m.ID, Name: m.Name, State: "absent", Detail: err.Error(), HasManifest: true, CanCtl: m.Init != ""})
				return
			}
			// Порты-кандидаты манифеста (native/entware): первый не-absent выигрывает.
			if len(m.Ports) > 0 {
				best := ServiceState{State: "absent"}
				for _, port := range m.Ports {
					resp, err := authedDo(client, bridgeDirVar, m.ID,
						m.Probe.MethodOrGET(),
						fmt.Sprintf("http://127.0.0.1:%d%s?%s", port, u.Path, u.RawQuery),
						m.Probe.Body)
					if err != nil {
						continue
					}
					body, _ := io.ReadAll(io.LimitReader(resp.Body, maxProbeBody))
					resp.Body.Close()
					best = classify(resp.StatusCode, resp.Header.Get("Content-Type"), body)
					if best.State != "absent" {
						break
					}
				}
				best.ID, best.Name = m.ID, m.Name
				best.HasManifest = true
				best.CanCtl = m.Init != ""
				add(best)
				return
			}
			// Обычный манифест без портов-кандидатов — пробуем resolved URL
			// с методом/телом из probe (aria2 и другие JSON-RPC отвечают только на POST).
			resp, err := authedDo(client, bridgeDirVar, m.ID, m.Probe.MethodOrGET(), u.String(), m.Probe.Body)
			if err != nil {
				add(ServiceState{ID: m.ID, Name: m.Name, State: "absent", HasManifest: true})
				return
			}
			body, _ := io.ReadAll(io.LimitReader(resp.Body, maxProbeBody))
			resp.Body.Close()
			st := classify(resp.StatusCode, resp.Header.Get("Content-Type"), body)
			st.ID, st.Name = m.ID, m.Name
			st.HasManifest = true
			st.CanCtl = m.Init != ""
			add(st)
		}(m)
	}

	// Ждём ВСЕГДА: ранний выход оставлял бы горутины, дописывающие в results
	// после возврата функции (data race, найден кворумом). Пробы сами ограничены
	// таймаутом 1с и семофором — хвост здесь не длиннее бюджета.
	wg.Wait()

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

func proxyEndpoint(dir, id string, ep *Endpoint) (*StatusProxy, error) {
	m, err := LoadManifest(dir, id)
	if err != nil {
		return nil, err
	}
	u, err := ValidateBridgeURL(ep.URL, m.Base)
	if err != nil {
		return nil, err
	}
	client := clientBridge()
	resp, err := authedDo(client, dir, id, ep.MethodOrGET(), u.String(), ep.Body)
	if err != nil {
		return &StatusProxy{Error: "сервис не отвечает"}, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxExtraBody))
	sp := &StatusProxy{HTTPCode: resp.StatusCode}
	var v interface{}
	if json.Unmarshal(body, &v) != nil {
		if len(body) >= maxExtraBody {
			return &StatusProxy{Error: "ответ сервиса слишком большой"}, nil
		}
		sp.Raw = truncate(string(body), 512)
		return sp, nil
	}
	out, err2 := json.Marshal(v)
	if err2 != nil {
		sp.Raw = truncate(string(body), 512)
		return sp, nil
	}
	sp.Body = json.RawMessage(out)
	return sp, nil
}

// ProxyStatus — прокси блока статуса (manifest.status или probe).
func ProxyStatus(dir, id string) (*StatusProxy, error) {
	m, err := LoadManifest(dir, id)
	if err != nil {
		return nil, err
	}
	ep := m.Status
	if ep == nil {
		if m.Probe.URL == "" {
			// process-модуль без адреса: не ошибка валидации, а честное «нет данных»
			return &StatusProxy{Error: "у модуля нет HTTP-источника статуса"}, nil
		}
		ep = &m.Probe
	}
	return proxyEndpoint(dir, id, ep)
}

// ProxyStats — прокси блока статистики (manifest.stats).
func ProxyStats(dir, id string) (*StatusProxy, error) {
	m, err := LoadManifest(dir, id)
	if err != nil {
		return nil, err
	}
	if m.Stats == nil {
		return nil, fmt.Errorf("у сервиса нет блока статистики")
	}
	return proxyEndpoint(dir, id, m.Stats)
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
