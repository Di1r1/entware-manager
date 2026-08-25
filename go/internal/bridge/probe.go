// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Сканер манифеста для редактора: опрашивает источники данных
// (probe/status/stats/extra) из ТЕКСТА манифеста (не обязательно
// сохранённого) и возвращает расплющенное дерево путей JSON-ответов —
// чтобы пользователь видел, какие поля можно добавить в fields[].
// Все URL прогоняются через SSRF-гейт; запросы только на loopback.
package bridge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

const (
	maxProbeDepth   = 4 // глубина рекурсии в JSON-ответе
	maxProbePaths   = 200
	previewMaxRunes = 60
	probeArrayPeek  = 3 // сколько элементов массива показываем как пути
)

// ProbePath — один найденный путь в JSON-ответе источника.
type ProbePath struct {
	Path    string `json:"path"`
	Preview string `json:"preview"` // пример значения (обрезан)
	Guess   string `json:"guess,omitempty"`
}

// ProbeSource — результат опроса одного источника.
type ProbeSource struct {
	Name     string      `json:"name"` // probe | status | stats | extra.<имя>
	URL      string      `json:"url,omitempty"`
	HTTPCode int         `json:"http_code,omitempty"`
	Error    string      `json:"error,omitempty"`
	Paths    []ProbePath `json:"paths,omitempty"`
	Total    int         `json:"total,omitempty"` // всего найдено путей (Paths может быть обрезан)
}

// pathCollector — копит пути с подсчётом общего числа: после лимита
// пути перестают добавляться, но счётчик растёт (для подписи «показано X из Y»).
type pathCollector struct {
	paths []ProbePath
	total int
}

func (c *pathCollector) add(p ProbePath) {
	c.total++
	if len(c.paths) < maxProbePaths {
		c.paths = append(c.paths, p)
	}
}

// ProbeSuggestion — найденный известный API-путь («похоже, это Netdata»).
type ProbeSuggestion struct {
	Path    string `json:"path"`
	Service string `json:"service"`
}

// ProbeResult — итог сканирования текста манифеста.
type ProbeResult struct {
	Valid           bool              `json:"valid"`
	ValidationError string            `json:"validation_error,omitempty"`
	Sources         []ProbeSource     `json:"sources,omitempty"`
	ListenPorts     []int             `json:"listen_ports,omitempty"` // подсказка: открытые TCP-порты роутера
	Suggestions     []ProbeSuggestion `json:"suggestions,omitempty"`  // известные API-пути, отвечающие на этой базе
}

// knownEndpoints — сигнатурные пути популярных сервисов моста.
var knownEndpoints = []struct {
	Path    string
	Service string
}{
	{"/api/v1/info", "Netdata"},
	{"/control/status", "AdGuard Home"},
	{"/rest/system/status", "Syncthing"},
	{"/rpc", "Transmission"},
}

// ProbeManifestData сканирует источники данных по сырому тексту манифеста.
// Мягкий разбор: даже невалидный (недоделанный) манифест отдаёт свои URL —
// чтобы редактирование было итеративным. valid=true только при строгой валидации.
func ProbeManifestData(data []byte) *ProbeResult {
	res := &ProbeResult{}
	if len(data) > MaxManifestSize {
		res.ValidationError = fmt.Sprintf("манифест больше %d байт", MaxManifestSize)
		return res
	}
	var m Manifest
	lenient := json.Unmarshal(data, &m) == nil && m.ID != ""
	if mm, err := ValidateManifestData(data); err == nil {
		res.Valid = true
		m = *mm
	} else {
		res.ValidationError = err.Error()
		if !lenient {
			return res
		}
	}

	client := clientBridge()
	add := func(name string, ep *Endpoint) {
		if ep == nil || ep.URL == "" {
			return
		}
		src := ProbeSource{Name: name}
		u, err := ValidateBridgeURL(ep.URL, m.Base)
		if err != nil {
			src.Error = err.Error()
			res.Sources = append(res.Sources, src)
			return
		}
		src.URL = ep.URL
		resp, derr := authedDo(client, bridgeDirVar, m.ID, ep.MethodOrGET(), u.String(), ep.Body)
		if derr != nil {
			src.Error = describeDialError(derr)
			res.Sources = append(res.Sources, src)
			return
		}
		defer resp.Body.Close()
		src.HTTPCode = resp.StatusCode
		body, _ := ioReadLimit(resp.Body, maxCardBody)
		if resp.StatusCode >= 400 {
			src.Error = describeHTTPError(resp.StatusCode)
			res.Sources = append(res.Sources, src)
			return
		}
		var v interface{}
		if json.Unmarshal(body, &v) != nil {
			src.Error = "ответ не является JSON"
			res.Sources = append(res.Sources, src)
			return
		}
		pc := &pathCollector{}
		flattenJSON("", v, 0, pc)
		src.Paths = pc.paths
		src.Total = pc.total
		res.Sources = append(res.Sources, src)
	}

	// status: как pickEndpoint — фолбэк на probe, если status не задан.
	if m.Status != nil {
		add("status", m.Status)
	} else if m.Probe.URL != "" {
		add("status", &m.Probe)
	} else {
		add("status", nil)
	}
	add("stats", m.Stats)

	names := make([]string, 0, len(m.Extra))
	for n := range m.Extra {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		add("extra."+n, &m.Extra[n].Endpoint)
	}
	if ports := LoopbackListeningPorts(); len(ports) > 0 && len(ports) <= 48 {
		res.ListenPorts = ports
	}
	// Автоопределение сервиса: порт отвечает (HTTP есть), но данные
	// ни по одному пути не получены — пробуем известные API-пути.
	baseReachable := false
	allFailed := len(res.Sources) > 0
	for _, s := range res.Sources {
		if s.HTTPCode > 0 {
			baseReachable = true
		}
		if s.Error == "" {
			allFailed = false
		}
	}
	if baseReachable && allFailed {
		for _, ke := range knownEndpoints {
			if len(res.Suggestions) >= 3 {
				break
			}
			if pathAlreadyUsed(&m, ke.Path) {
				continue
			}
			u, err := ValidateBridgeURL(ke.Path, m.Base)
			if err != nil {
				continue
			}
			resp, derr := authedDo(client, bridgeDirVar, m.ID, http.MethodGet, u.String(), "")
			if derr != nil {
				continue
			}
			body, _ := ioReadLimit(resp.Body, maxCardBody)
			resp.Body.Close()
			if resp.StatusCode < 400 {
				var v interface{}
				if json.Unmarshal(body, &v) == nil || looksLikeJSON(body) { // /rpc Transmission отдаёт текст при GET
					res.Suggestions = append(res.Suggestions, ProbeSuggestion{Path: ke.Path, Service: ke.Service})
				}
			}
		}
	}
	return res
}

// pathAlreadyUsed — путь уже прописан в одном из источников манифеста.
func pathAlreadyUsed(m *Manifest, path string) bool {
	urls := []string{m.Probe.URL}
	if m.Status != nil {
		urls = append(urls, m.Status.URL)
	}
	if m.Stats != nil {
		urls = append(urls, m.Stats.URL)
	}
	for _, u := range urls {
		u = strings.TrimSuffix(u, "/")
		if u == path || strings.HasSuffix(u, path) {
			return true
		}
	}
	return false
}

// flattenJSON — рекурсивное расплющивание: скаляры становятся путями,
// объекты раскрываются, массивы дают count-путь и первые элементы с индексами.
func flattenJSON(prefix string, v interface{}, depth int, out *pathCollector) {
	if out.total >= hardPathScanCap || depth > maxProbeDepth {
		return
	}
	switch t := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := k
			if prefix != "" {
				p = prefix + "." + k
			}
			child := t[k]
			if _, isMap := child.(map[string]interface{}); isMap {
				flattenJSON(p, child, depth+1, out)
				continue
			}
			if arr, isArr := child.([]interface{}); isArr {
				appendArrayPaths(p, arr, depth, out)
				continue
			}
			appendScalar(p, child, out)
		}
	case []interface{}:
		appendArrayPaths(prefix, t, depth, out)
	default:
		appendScalar(prefix, v, out)
	}
}

// hardPathScanCap — потолок обхода: у гигантских ответов (547 графиков
// Netdata) не сканируем тысячи узлов зря, счётчик total после него неточен.
const hardPathScanCap = 2000

func appendArrayPaths(path string, arr []interface{}, depth int, out *pathCollector) {
	if path == "" || out.total >= hardPathScanCap {
		return
	}
	out.add(ProbePath{Path: path, Preview: strconv.Itoa(len(arr)) + " эл.", Guess: "count"})
	n := len(arr)
	if n > probeArrayPeek {
		n = probeArrayPeek
	}
	for i := 0; i < n; i++ {
		if out.total >= hardPathScanCap {
			return
		}
		item := arr[i]
		p := path + "." + strconv.Itoa(i)
		if _, isMap := item.(map[string]interface{}); isMap {
			flattenJSON(p, item, depth+1, out)
			continue
		}
		if a, ok := item.([]interface{}); ok {
			appendArrayPaths(p, a, depth+1, out)
			continue
		}
		appendScalar(p, item, out)
	}
}

func appendScalar(path string, v interface{}, out *pathCollector) {
	if path == "" || out.total >= hardPathScanCap {
		return
	}
	switch t := v.(type) {
	case string:
		s := t
		if runes := []rune(s); len(runes) > previewMaxRunes {
			s = string(runes[:previewMaxRunes]) + "…"
		}
		out.add(ProbePath{Path: path, Preview: s})
	case float64:
		guess := guessNumType(path, t)
		out.add(ProbePath{Path: path, Preview: formatNumPreview(t), Guess: guess})
	case bool:
		preview := "нет"
		if t {
			preview = "да"
		}
		out.add(ProbePath{Path: path, Preview: preview, Guess: "bool"})
	case nil:
		out.add(ProbePath{Path: path, Preview: "null"})
	default:
		b, _ := json.Marshal(t)
		out.add(ProbePath{Path: path, Preview: truncateRunes(string(b), previewMaxRunes)})
	}
}

// guessNumType — грубая эвристика типа по имени ключа: bytes/ms/dur/num.
func guessNumType(path string, f float64) string {
	low := strings.ToLower(path)
	last := low
	if i := strings.LastIndex(low, "."); i >= 0 {
		last = low[i+1:]
	}
	switch {
	case containsAny(last, "byte", "bytes", "size", "_sz"):
		return "bytes"
	case containsAny(last, "time", "latency", "duration", "_ms", "rtt"):
		if f < 1000 { // доли секунды → мс
			return "ms"
		}
		return "dur"
	default:
		return "num"
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func formatNumPreview(f float64) string {
	if f == float64(int64(f)) {
		return groupInt(int64(f))
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
