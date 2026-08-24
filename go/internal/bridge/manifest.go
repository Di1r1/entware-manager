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
	"os"
	"path/filepath"
	"regexp"
)

const (
	MaxManifestSize = 16 * 1024
	MaxManifests    = 20
	MaxActions      = 10
	maxIDLen        = 32
)

var idRe = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

type Endpoint struct {
	URL    string `json:"url"`
	Expect string `json:"expect,omitempty"` // "json" → ответ обязан парситься
	Method string `json:"method,omitempty"`
	Body   string `json:"body,omitempty"`
}

type Action struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Method  string `json:"method,omitempty"`
	URL     string `json:"url"`
	Body    string `json:"body,omitempty"`
	Confirm bool   `json:"confirm,omitempty"`
}

type Manifest struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Base    string    `json:"base,omitempty"` // база для относительных URL
	Probe   Endpoint  `json:"probe"`
	Status  *Endpoint `json:"status,omitempty"`
	Actions []Action  `json:"actions,omitempty"`
}

// AuthCreds — секретный файл <id>.auth.json (0600), НИКОГДА не попадает в ответы.
type AuthCreds struct {
	Type     string `json:"type"` // "basic" | ""
	Username string `json:"username"`
	Password string `json:"password"`
}

// ValidateManifest — структурная и семантическая проверка + гейт URL.
func ValidateManifest(m *Manifest) error {
	if !idRe.MatchString(m.ID) {
		return fmt.Errorf("id %q не подходит под [a-z0-9_-]{1,%d}", m.ID, maxIDLen)
	}
	if len(m.Name) == 0 || len(m.Name) > 64 {
		return fmt.Errorf("имя пустое или длиннее 64")
	}
	if _, err := ValidateBridgeURL(m.Probe.URL, m.Base); err != nil {
		return fmt.Errorf("probe: %w", err)
	}
	if m.Status != nil {
		if _, err := ValidateBridgeURL(m.Status.URL, m.Base); err != nil {
			return fmt.Errorf("status: %w", err)
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
