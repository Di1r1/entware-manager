// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Манифесты коннекторов моста: <id>.json (публичное описание, 0644) +
// опционально <id>.auth.json (секреты, 0600). Валидация строгая fail-closed:
// неизвестные поля, oversize и битые id отвергаются.
package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

const (
	MaxManifestSize = 16 * 1024
	MaxManifests    = 20
	MaxActions      = 10
	MaxExtra        = 8
	MaxFields       = 24
	maxIDLen        = 32
)

var idRe = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

const (
	maxProcessNames = 24
	procNamePattern = `[a-zA-Z0-9._-]{1,64}`
)

var procNameRe = regexp.MustCompile(`^` + procNamePattern + `$`)

var sourceNameRe = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

type Endpoint struct {
	URL    string `json:"url"`
	Expect string `json:"expect,omitempty"` // "json" → ответ обязан парситься
	Method string `json:"method,omitempty"`
	Body   string `json:"body,omitempty"`
}

// MethodOrGET — метод с дефолтом GET.
func (e *Endpoint) MethodOrGET() string {
	if e != nil && e.Method != "" {
		return e.Method
	}
	return http.MethodGet
}

type Action struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Method  string `json:"method,omitempty"`
	URL     string `json:"url"`
	Body    string `json:"body,omitempty"`
	Confirm bool   `json:"confirm,omitempty"`
}

// FieldDef — универсальное описание поля карточки сканера:
// значение берётся из ответа эндпоинта по точечному пути (a.b.c).
type FieldDef struct {
	Path  string `json:"path"`           // точечный путь в JSON-ответе
	Label string `json:"label"`          // подпись на карточке
	From  string `json:"from,omitempty"` // источник: status | stats | <имя extra>; пусто = перебор
	Type  string `json:"type,omitempty"` // "" текст | bool | bytes | count | num | ms | dur | top
	Tile  bool   `json:"tile,omitempty"` // показать крупной плиткой
	Color string `json:"color,omitempty"`
	On    string `json:"on,omitempty"`  // текст для true (по умолчанию «да»)
	Off   string `json:"off,omitempty"` // текст для false (по умолчанию «нет»)
}

type Manifest struct {
	ID      string                    `json:"id"`
	Name    string                    `json:"name"`
	Base    string                    `json:"base,omitempty"` // база для относительных URL
	Probe   Endpoint                  `json:"probe"`
	Status  *Endpoint                 `json:"status,omitempty"`
	Stats   *Endpoint                 `json:"stats,omitempty"`  // блок статистики для карточки
	Extra   map[string]*SliceEndpoint `json:"extra,omitempty"`  // именованные GET-эндпоинты (slice_last — обрезка массива)
	Ports   []int                     `json:"ports,omitempty"`  // кандидаты портов base (native/entware установки)
	Fields  []FieldDef                `json:"fields,omitempty"` // универсальные поля карточки сканера
	Actions []Action                  `json:"actions,omitempty"`
	// Process — имена процессов для детекта без веб-порта (демоны init.d).
	// Если задано, discovery и watch определяют живость ТОЛЬКО по процессу,
	// HTTP-probe игнорируется полностью (кворум v1.15.7: процесс = источник
	// истины). Поля карточки status/stats/extra работают как раньше.
	Process []string `json:"process,omitempty"`
	// Init — имя сервиса в /opt/etc/init.d (S??имя/K??имя) для кнопок
	// старт/стоп/рестарт. Выполняется ТОЛЬКО при включённой галочке
	// «управление» в настройках модуля (prefs.control, fail-closed).
	Init string `json:"init,omitempty"`
}

// AuthCreds — секретный файл <id>.auth.json (0600), НИКОГДА не попадает в ответы.
// Типы: "basic" | "cookie_login" | "" (без авторизации).
type AuthCreds struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Password string `json:"password"`
	LoginURL string `json:"login_url,omitempty"` // для cookie_login, относительно Base
}

// SliceEndpoint — GET-эндпоинт с опциональной обрезкой массива-ответа
// до последних N элементов (для тяжёлых историй).
type SliceEndpoint struct {
	Endpoint
	SliceLast int `json:"slice_last,omitempty"`
}

// ValidateManifest — структурная и семантическая проверка + гейт URL.
func ValidateManifest(m *Manifest) error {
	if !idRe.MatchString(m.ID) {
		return fmt.Errorf("id %q не подходит под [a-z0-9_-]{1,%d}", m.ID, maxIDLen)
	}
	if len(m.Name) == 0 || len(m.Name) > 64 {
		return fmt.Errorf("имя пустое или длиннее 64")
	}
	// probe обязателен только для HTTP-модулей: process-детекту адрес не нужен
	if m.Probe.URL == "" {
		if len(m.Process) == 0 {
			return fmt.Errorf("probe: пустой url (или задайте process для детекта по процессу)")
		}
	} else if _, err := ValidateBridgeURL(m.Probe.URL, m.Base); err != nil {
		return fmt.Errorf("probe: %w", err)
	}
	if m.Status != nil {
		if _, err := ValidateBridgeURL(m.Status.URL, m.Base); err != nil {
			return fmt.Errorf("status: %w", err)
		}
	}
	if m.Stats != nil {
		if _, err := ValidateBridgeURL(m.Stats.URL, m.Base); err != nil {
			return fmt.Errorf("stats: %w", err)
		}
	}
	if len(m.Fields) > MaxFields {
		return fmt.Errorf("полей больше %d", MaxFields)
	}
	for _, p := range m.Ports {
		if p < 1 || p > 65535 {
			return fmt.Errorf("порт %d вне диапазона 1–65535", p)
		}
	}
	if len(m.Process) > maxProcessNames {
		return fmt.Errorf("process: имён больше %d", maxProcessNames)
	}
	for i, p := range m.Process {
		if !procNameRe.MatchString(p) {
			return fmt.Errorf("process[%d]: имя %q не подходит под %s", i, p, procNamePattern)
		}
	}
	if m.Init != "" && !initNameRe.MatchString(m.Init) {
		return fmt.Errorf("init: имя %q не подходит под %s", m.Init, procNamePattern)
	}
	for i, f := range m.Fields {
		if f.Path == "" || len(f.Path) > 128 {
			return fmt.Errorf("field[%d]: пустой или длинный path", i)
		}
		if f.Label == "" || len(f.Label) > 64 {
			return fmt.Errorf("field[%d]: пустая или длинная label", i)
		}
		switch f.Type {
		case "", "bool", "bytes", "count", "num", "ms", "dur", "top", "kbs":
		default:
			return fmt.Errorf("field[%d]: неизвестный тип %q (допустимы bool, bytes, count, num, ms, dur, top, kbs)", i, f.Type)
		}
		if f.From != "" && !sourceNameRe.MatchString(f.From) {
			return fmt.Errorf("field[%d]: плохое имя источника %q", i, f.From)
		}
	}
	if len(m.Extra) > MaxExtra {
		return fmt.Errorf("extra-эндпоинтов больше %d", MaxExtra)
	}
	for name, ep := range m.Extra {
		if !idRe.MatchString(name) || len(name) > maxIDLen {
			return fmt.Errorf("extra %q: плохое имя", name)
		}
		if ep == nil {
			return fmt.Errorf("extra %q: пустой", name)
		}
		if _, err := ValidateBridgeURL(ep.URL, m.Base); err != nil {
			return fmt.Errorf("extra %q: %w", name, err)
		}
	}
	if len(m.Actions) > MaxActions {
		return fmt.Errorf("действий больше %d", MaxActions)
	}
	seen := map[string]bool{}
	for i, a := range m.Actions {
		if !idRe.MatchString(a.ID) {
			return fmt.Errorf("action[%d]: плохой id", i)
		}
		if seen[a.ID] {
			return fmt.Errorf("action[%d]: дубликат id", i)
		}
		seen[a.ID] = true
		if _, err := ValidateBridgeURL(a.URL, m.Base); err != nil {
			return fmt.Errorf("action %q: %w", a.ID, err)
		}
	}
	return nil
}

// LoadManifest читает и валидирует манифест по id (защита от path traversal:
// файл только внутри dir).
func LoadManifest(dir, id string) (*Manifest, error) {
	if !idRe.MatchString(id) {
		return nil, fmt.Errorf("плохой id")
	}
	path := filepath.Join(dir, id+".json")
	if filepath.Dir(path) != cleanDir(dir) {
		return nil, fmt.Errorf("выход за каталог bridge/")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("манифест не найден")
	}
	if len(data) > MaxManifestSize {
		return nil, fmt.Errorf("манифест больше %d байт", MaxManifestSize)
	}
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields() // fail-closed против сюрпризных расширений
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("битый манифест: %v", err)
	}
	if err := ValidateManifest(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// LoadAuth читает секреты коннектора (может отсутствовать — не ошибка).
func LoadAuth(dir, id string) *AuthCreds {
	if !idRe.MatchString(id) {
		return nil
	}
	path := filepath.Join(dir, id+".auth.json")
	if filepath.Dir(path) != cleanDir(dir) {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var a AuthCreds
	if json.Unmarshal(data, &a) != nil {
		return nil
	}
	return &a
}

// ListManifests — все валидные манифесты каталога (битые пропускаются).
func ListManifests(dir string) []*Manifest {
	out := make([]*Manifest, 0, 8)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" ||
			filepath.Ext(trimJSONExt(e.Name())) == ".auth" {
			continue
		}
		id := trimJSONExt(e.Name())
		if len(out) >= MaxManifests {
			break
		}
		if m, err := LoadManifest(dir, id); err == nil {
			m.ID = id // имя файла главнее поля
			out = append(out, m)
		}
	}
	return out
}

func trimJSONExt(name string) string {
	if len(name) > 5 && name[len(name)-5:] == ".json" {
		return name[:len(name)-5]
	}
	return name
}

func cleanDir(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
}
