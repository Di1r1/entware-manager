// Package cache — лёгкий файловый кэш для CGI-процессов.
//
// Кэш живёт в /tmp (очищается перезагрузкой, не занимает flash).
// Запись атомарна (temp + rename) — безопасно при конкурентных CGI-процессах.
// Повреждённый или устаревший кэш игнорируется, вызывающий код пересчитывает.
package cache

import (
	"os"
	"path/filepath"
	"time"
)

var baseDir = "/tmp/entware/cache"

// SetBaseDir переопределяет каталог кэша (для тестов).
func SetBaseDir(dir string) {
	baseDir = dir
}

func path(key string) string {
	return filepath.Join(baseDir, key)
}

// Get возвращает содержимое кэша, если оно свежее (не старше ttl).
// ok=false, если кэша нет, он повреждён или истёк TTL.
func Get(key string, ttl time.Duration) ([]byte, bool) {
	p := path(key)
	st, err := os.Stat(p)
	if err != nil {
		return nil, false
	}
	if ttl > 0 && time.Since(st.ModTime()) > ttl {
		return nil, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Put пишет данные в кэш атомарно (temp + rename в том же каталоге).
// Ошибки каталога игнорируются — отсутствие кэша не критично.
func Put(key string, data []byte) error {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}
	p := path(key)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Invalidate удаляет ключи из кэша. Отсутствующие ключи не считаются ошибкой.
func Invalidate(keys ...string) {
	for _, k := range keys {
		os.Remove(path(k))
	}
}
