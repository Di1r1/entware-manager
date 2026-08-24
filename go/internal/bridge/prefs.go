// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Настройки модулей моста: /opt/web_entware/bridge/_prefs.json (0600).
// enabled=false → сервис исключается из discovery и карточки гаснут.
package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type ModulePrefs struct {
	Enabled       bool `json:"enabled"`
	Notifications bool `json:"notifications"`
}

type prefsFile struct {
	Modules map[string]ModulePrefs `json:"modules"`
}

var prefsMu sync.Mutex

func prefsPath() string { return filepath.Join(bridgeDirVar, "_prefs.json") }

// LoadPrefs читает настройки (файла нет → все включены по умолчанию).
func LoadPrefs() prefsFile {
	pf := prefsFile{Modules: map[string]ModulePrefs{}}
	data, err := os.ReadFile(prefsPath())
	if err != nil {
		return pf
	}
	var p prefsFile
	if json.Unmarshal(data, &p) != nil || p.Modules == nil {
		return pf
	}
	return p
}

// IsEnabled — участвует ли модуль в мосте (нет записи = включён).
func IsEnabled(id string) bool {
	prefsMu.Lock()
	defer prefsMu.Unlock()
	pf := LoadPrefs()
	m, ok := pf.Modules[id]
	if !ok {
		return true
	}
	return m.Enabled
}

// SavePrefs атомарно перезаписывает настройки.
func SavePrefs(pf prefsFile) error {
	prefsMu.Lock()
	defer prefsMu.Unlock()
	out, err := json.MarshalIndent(pf, "", "    ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	tmp := prefsPath() + ".tmp"
	if err := os.WriteFile(tmp, out, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, prefsPath()); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
