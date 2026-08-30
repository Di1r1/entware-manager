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
	Warnings        []string          `json:"warnings,omitempty"` // неблокирующие замечания (напр. init≠process)
	Sources         []ProbeSource     `json:"sources,omitempty"`
	ListenPorts     []int             `json:"listen_ports,omitempty"` // подсказка: открытые TCP-порты роутера
	PortLabels      map[int]string    `json:"port_labels,omitempty"`  // база подписей известных портов
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

// commonAPIPaths — распространённые API-пути произвольных сервисов для
// автоподбора, когда база отвечает, но ни один путь манифеста не дал JSON.
// Первым найденный даёт кнопку «применить» в сканере (как knownEndpoints).
var commonAPIPaths = []struct {
	Path    string
	Service string
}{
	{"/status", "статус"},
	{"/api/status", "статус"},
	{"/api", "API"},
	{"/api/v1/status", "статус"},
	{"/api/info", "инфо"},
	{"/index.json", "индекс"},
	{"/json", "JSON"},
	{"/v1/status", "статус"},
	{"/api/v1/info", "инфо"},
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
	res.Warnings = ManifestWarnings(&m)

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
	res.PortLabels = PortLabelsDict()
	// Автоопределение сервиса: порт отвечает (HTTP есть), но данные
	// ни по одному пути не получены — пробуем сигнатуры известных сервисов
	// и распространённые API-пути. Известные идут первыми (подписанные),
	// затем общие — так для произвольного сервиса сканер находит рабочий путь.
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
		candidates := append([]struct {
			Path    string
			Service string
		}{}, knownEndpoints...)
		candidates = append(candidates, commonAPIPaths...)
		seen := map[string]bool{}
		for _, ke := range candidates {
			if len(res.Suggestions) >= 6 {
				break
			}
			if seen[ke.Path] || pathAlreadyUsed(&m, ke.Path) {
				continue
			}
			seen[ke.Path] = true
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

// pathAlreadyUsed — путь уже ТОЧНО прописан в одном из источников манифеста.
// Только точное совпадение (после срезания хвостового "/"): суффикс-сравнение
// здесь вредно — манифест с неработающим /api/status не должен блокировать
// подсказку рабочего /status (автоподбор запускается лишь когда все пути
// манифеста не дали JSON).
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
		path = strings.TrimSuffix(path, "/")
		if u == path {
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
	guess, preview := "count", strconv.Itoa(len(arr))+" эл."
	if looksLikeTopArray(arr, probeArrayPeek) {
		guess, preview = "top", previewTopArray(arr)
	}
	out.add(ProbePath{Path: path, Preview: preview, Guess: guess})
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

// looksLikeTopArray — массив одно-ключевых объектов со числами
// (кандидат type=top; просматриваются первые limit элементов).
func looksLikeTopArray(arr []interface{}, limit int) bool {
	if len(arr) == 0 {
		return false
	}
	if len(arr) > limit {
		arr = arr[:limit]
	}
	for _, it := range arr {
		m, ok := it.(map[string]interface{})
		if !ok || len(m) != 1 {
			return false
		}
		for _, v := range m {
			if _, ok := v.(float64); !ok {
				return false
			}
		}
	}
	return true
}

// previewTopArray — пример содержимого top-массива: "ytlh (300), …".
func previewTopArray(arr []interface{}) string {
	var b strings.Builder
	count := 0
	for _, it := range arr {
		if count >= 2 {
			break
		}
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		for k, v := range m {
			if f, ok := v.(float64); ok {
				if b.Len() > 0 {
					b.WriteString(", ")
				}
				b.WriteString(k + " (" + groupInt(int64(f)) + ")")
			}
		}
		count++
	}
	if b.Len() == 0 {
		return strconv.Itoa(len(arr)) + " эл."
	}
	return b.String() + "…"
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

// guessNumType — грубая эвристика типа по имени ключа: bytes/ms/dur/kbs/num.
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
	case containsAny(last, "speed", "rate", "download", "upload", "kbs", "traffic", "throughput"):
		return "kbs"
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
