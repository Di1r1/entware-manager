// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Универсальный сканер карточки: опрашивает все JSON-эндпоинты манифеста
// (status/stats/extra) и собирает значения полей по путям из manifest.Fields.
// Карточка любого сервиса строится данными, без отдельного кода под приложение.
package bridge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const maxCardBody = 1024 * 1024

// CardTile / CardRow — формат, совместимый с рендером панели
// (renderBridgeDetails: tiles[{label,value,color}], rows[[label,value]]).
type CardTile struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Color string `json:"color,omitempty"`
}

type CardRow struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type CardData struct {
	Tiles []CardTile `json:"tiles,omitempty"`
	Rows  []CardRow  `json:"rows,omitempty"`
	State string     `json:"state,omitempty"` // текущее состояние из discovery
	Name  string     `json:"name,omitempty"`
	Procs []ProcStat `json:"procs,omitempty"` // живые числа process-модуля
}

// fetchJSON — GET/POST эндпоинта с authedDo; возврат распарсенного тела.
func fetchJSON(client *http.Client, m *Manifest, ep *Endpoint) (json.RawMessage, int, error) {
	u, err := ValidateBridgeURL(ep.URL, m.Base)
	if err != nil {
		return nil, 0, err
	}
	resp, err := authedDo(client, bridgeDirVar, m.ID, ep.MethodOrGET(), u.String(), ep.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("сервис не отвечает")
	}
	defer resp.Body.Close()
	body, _ := ioReadLimit(resp.Body, maxCardBody)
	if code := resp.StatusCode; code >= 400 {
		return nil, code, fmt.Errorf("HTTP %d", code)
	}
	var v json.RawMessage
	if json.Unmarshal(body, &v) != nil {
		return nil, resp.StatusCode, fmt.Errorf("ответ не является JSON")
	}
	return v, resp.StatusCode, nil
}

// BuildCard — универсальный сканер для карточки сервиса.
func BuildCard(dir, id string) (*CardData, error) {
	m, err := LoadManifest(dir, id)
	if err != nil {
		return nil, err
	}

	card := &CardData{Name: m.Name}
	client := clientBridge()

	// Источники данных: status/stats/extra (заполняются по мере опроса).
	sources := map[string]map[string]interface{}{}
	getSource := func(name string, ep *Endpoint) map[string]interface{} {
		if name == "" || ep == nil {
			return nil
		}
		if _, done := sources[name]; done {
			return sources[name]
		}
		raw, code, err := fetchJSON(client, m, ep)
		var parsed map[string]interface{}
		if err == nil && code < 400 {
			json.Unmarshal(raw, &parsed) // невалидный → пустая карта
		}
		if parsed == nil {
			parsed = map[string]interface{}{}
		}
		sources[name] = parsed
		return parsed
	}

	for _, f := range m.Fields {
		from := f.From
		if from == "" {
			from = "status" // умолчание: основной статус-эндпоинт
		}
		src := getSource(from, pickEndpoint(m, from))
		val, ok := lookupPath(src, strings.Split(f.Path, "."))
		if !ok {
			continue // поля нет в ответе — пропускаем строку
		}
		value := formatValue(&f, val, f.Type)
		if value == "" {
			continue
		}
		if f.Tile {
			card.Tiles = append(card.Tiles, CardTile{Label: f.Label, Value: value, Color: f.Color})
		} else {
			card.Rows = append(card.Rows, CardRow{Label: f.Label, Value: value})
		}
	}

	// process-модуль: живые числа (PID/аптайм/CPU/память) клиент берёт из
	// card.Procs и рисует сам (плитки + живой блок) — ProcessDetails больше
	// не дублируется в card.Rows (это был второй источник «PID · память»,
	// пересекающийся с карточкой; см. renderBridgeCardsOnStats).
	if len(m.Process) > 0 {
		card.Procs = ProcessStats(m.Process)
	}

	return card, nil
}

// pickEndpoint — эндпоинт-источник по имени поля.
// Пустой Probe (process-модуль без адреса) — не источник: вернём nil,
// иначе fetchJSON отдавал бы ошибку валидации вместо тихого пропуска.
func pickEndpoint(m *Manifest, from string) *Endpoint {
	switch from {
	case "", "status":
		if m.Status != nil {
			return m.Status
		}
		if m.Probe.URL != "" {
			return &m.Probe
		}
		return nil
	case "stats":
		return m.Stats
	}
	if ep, ok := m.Extra[from]; ok && ep != nil {
		return &ep.Endpoint
	}
	if m.Status != nil {
		return m.Status
	}
	if m.Probe.URL != "" {
		return &m.Probe
	}
	return nil
}

// lookupPath идёт по точечному пути в карте/массиве; числовой сегмент
// трактуется как индекс массива (пути вида dns_addresses.2 от сканера).
func lookupPath(src map[string]interface{}, parts []string) (interface{}, bool) {
	if src == nil || len(parts) == 0 {
		return nil, false
	}
	var cur interface{} = src
	for _, p := range parts {
		switch c := cur.(type) {
		case map[string]interface{}:
			next, ok := c[p]
			if !ok {
				return nil, false
			}
			cur = next
		case []interface{}:
			idx, err := strconv.Atoi(p)
			if err != nil || idx < 0 || idx >= len(c) {
				return nil, false
			}
			cur = c[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

// formatValue приводит значение к читаемой строке согласно типу поля.
// f может быть nil (прямые тестовые вызовы) — on/off тогда игнорируются.
func formatValue(f *FieldDef, v interface{}, typ string) string {
	switch typ {
	case "bool":
		b, _ := v.(bool)
		if b {
			if f != nil && f.On != "" {
				return f.On
			}
			return "да"
		}
		if f != nil && f.Off != "" {
			return f.Off
		}
		return "нет"
	case "dur":
		if f, ok := toFloat(v); ok {
			return humanDur(f)
		}
		return ""
	case "ms":
		if f, ok := toFloat(v); ok {
			return strconv.Itoa(int(MathRound(f*1000))) + " мс"
		}
		return ""
	case "num":
		if f, ok := toFloat(v); ok {
			return groupInt(int64(f))
		}
		return ""
	case "top":
		// массив одно-ключевых объектов [{"домен": N}] → "имя (N), ..."
		arr, ok := v.([]interface{})
		if !ok {
			return ""
		}
		var parts []string
		for i, item := range arr {
			if i >= 5 {
				break
			}
			if m, ok := item.(map[string]interface{}); ok {
				for k, cnt := range m {
					parts = append(parts, k+" ("+groupInt(int64(toF(cnt)))+")")
				}
			}
		}
		return strings.Join(parts, ", ")
	case "count":
		switch t := v.(type) {
		case []interface{}:
			return strconv.Itoa(len(t))
		case map[string]interface{}:
			return strconv.Itoa(len(t))
		case float64:
			return strconv.Itoa(int(t))
		case string:
			if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
				return strconv.Itoa(int(f))
			}
		}
		return "0"
	case "bytes":
		f, ok := toFloat(v)
		if !ok {
			return ""
		}
		return humanBytesServer(f)
	case "kbs":
		// скорость: байты/с → КБ/с (Transmission и подобные отдают Б/с)
		f, ok := toFloat(v)
		if !ok {
			return ""
		}
		return groupInt(int64(f/1024)) + " КБ/с"
	}
	// текст/число как есть
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return groupInt(int64(t))
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "да"
		}
		return "нет"
	}
	return ""
}

func toF(v interface{}) float64 {
	f, _ := v.(float64)
	return f
}

func toFloat(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case string:
		// некоторые сервисы (Netdata) отдают числа строками
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	}
	return 0, false
}

// humanBytesServer — человекочитаемый размер для серверных плиток.
func humanBytesServer(b float64) string {
	const KB, MB, GB = 1024, 1048576, 1073741824
	switch {
	case b >= GB:
		return trimZero(b/GB) + " ГБ"
	case b >= MB:
		return trimZero(b/MB) + " МБ"
	case b >= KB:
		return trimZero(b/KB) + " КБ"
	default:
		return strconv.Itoa(int(b)) + " Б"
	}
}

func trimZero(f float64) string {
	s := strconv.FormatFloat(f, 'f', 1, 64)
	return strings.TrimSuffix(s, ".0")
}

// groupInt — разряды через пробел (96 461).
func groupInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ' ')
		}
		out = append(out, c)
	}
	return string(out)
}

// MathRound — округление без math импорта в горячих путях.
func MathRound(f float64) float64 {
	if f < 0 {
		return -float64(int64(f - 0.5))
	}
	return float64(int64(f + 0.5))
}

// humanDur — секунды → «1 дн 2 ч 3 мин».
func humanDur(sec float64) string {
	s := int64(sec)
	d := s / 86400
	h := (s % 86400) / 3600
	m := (s % 3600) / 60
	var parts []string
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%d дн", d))
	}
	if h > 0 || d > 0 {
		parts = append(parts, fmt.Sprintf("%d ч", h))
	}
	parts = append(parts, fmt.Sprintf("%d мин", m))
	return strings.Join(parts, " ")
}
