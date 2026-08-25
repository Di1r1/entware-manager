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
	"sort"
	"strconv"
	"strings"
)

const (
	maxProbeDepth   = 4 // глубина рекурсии в JSON-ответе
	maxProbePaths   = 64
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
}

// ProbeResult — итог сканирования текста манифеста.
type ProbeResult struct {
	Valid           bool          `json:"valid"`
	ValidationError string        `json:"validation_error,omitempty"`
	Sources         []ProbeSource `json:"sources,omitempty"`
	ListenPorts     []int         `json:"listen_ports,omitempty"` // подсказка: открытые TCP-порты роутера
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
		paths := []ProbePath{}
		flattenJSON("", v, 0, &paths)
		src.Paths = paths
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
	return res
}

// flattenJSON — рекурсивное расплющивание: скаляры становятся путями,
// объекты раскрываются, массивы дают count-путь и первые элементы с индексами.
func flattenJSON(prefix string, v interface{}, depth int, out *[]ProbePath) {
	if len(*out) >= maxProbePaths || depth > maxProbeDepth {
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

func appendArrayPaths(path string, arr []interface{}, depth int, out *[]ProbePath) {
	if path == "" {
		return
	}
	if len(*out) >= maxProbePaths {
		return
	}
	*out = append(*out, ProbePath{Path: path, Preview: strconv.Itoa(len(arr)) + " эл.", Guess: "count"})
	n := len(arr)
	if n > probeArrayPeek {
		n = probeArrayPeek
	}
	for i := 0; i < n; i++ {
		if len(*out) >= maxProbePaths {
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

func appendScalar(path string, v interface{}, out *[]ProbePath) {
	if path == "" || len(*out) >= maxProbePaths {
		return
	}
	switch t := v.(type) {
	case string:
		s := t
		if runes := []rune(s); len(runes) > previewMaxRunes {
			s = string(runes[:previewMaxRunes]) + "…"
		}
		*out = append(*out, ProbePath{Path: path, Preview: s})
	case float64:
		guess := guessNumType(path, t)
		*out = append(*out, ProbePath{Path: path, Preview: formatNumPreview(t), Guess: guess})
	case bool:
		preview := "нет"
		if t {
			preview = "да"
		}
		*out = append(*out, ProbePath{Path: path, Preview: preview, Guess: "bool"})
	case nil:
		*out = append(*out, ProbePath{Path: path, Preview: "null"})
	default:
		b, _ := json.Marshal(t)
		*out = append(*out, ProbePath{Path: path, Preview: truncateRunes(string(b), previewMaxRunes)})
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
