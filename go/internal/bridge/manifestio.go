// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Чтение/сохранение/удаление манифестов для редактора во вкладке «Модули».
package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ValidateManifestData — строгая проверка JSON-текста манифеста
// (DisallowUnknownFields + семантика + SSRF-гейт). Возвращает разобранный
// манифест или понятную ошибку для показа пользователю.
func ValidateManifestData(data []byte) (*Manifest, error) {
	if len(data) > MaxManifestSize {
		return nil, fmt.Errorf("манифест больше %d байт", MaxManifestSize)
	}
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("битый JSON: %v", err)
	}
	if err := ValidateManifest(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// GetManifestRaw — исходный текст файла манифеста (для редактора).
func GetManifestRaw(dir, id string) (string, bool, error) {
	if !idRe.MatchString(id) {
		return "", false, fmt.Errorf("плохой id")
	}
	path := filepath.Join(dir, id+".json")
	if filepath.Dir(path) != cleanDir(dir) {
		return "", false, fmt.Errorf("выход за каталог")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	return string(data), true, nil
}

// SaveManifestFile атомарно пишет манифест (0644) после валидации.
// id из параметра обязан совпадать с id внутри текста.
func SaveManifestFile(dir, id string, data []byte) error {
	m, err := ValidateManifestData(data)
	if err != nil {
		return err
	}
	if m.ID != id {
		return fmt.Errorf("id в тексте (%q) не совпадает с именем файла (%q)", m.ID, id)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, id+".json")
	if filepath.Dir(path) != cleanDir(dir) {
		return fmt.Errorf("выход за каталог")
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// DeleteManifestFile удаляет файл манифеста (не .auth.json).
func DeleteManifestFile(dir, id string) error {
	if !idRe.MatchString(id) {
		return fmt.Errorf("плохой id")
	}
	path := filepath.Join(dir, id+".json")
	if filepath.Dir(path) != cleanDir(dir) {
		return fmt.Errorf("выход за каталог")
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// HasManifestFile — есть ли файл манифеста у сервиса.
func HasManifestFile(dir, id string) bool {
	if !idRe.MatchString(id) {
		return false
	}
	path := filepath.Join(dir, id+".json")
	if filepath.Dir(path) != cleanDir(dir) {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
