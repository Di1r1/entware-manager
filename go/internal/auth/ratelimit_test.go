// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRatelimitFlow(t *testing.T) {
	dir := t.TempDir()
	RatelimitDir = dir
	ip := "192.168.1.50"

	if RateLimited(ip) {
		t.Fatal("чистый IP не должен быть заблокирован")
	}
	for i := 0; i < RateLimitMaxFails; i++ {
		RecordFailure(ip)
	}
	if !RateLimited(ip) {
		t.Fatal("после исчерпания попыток IP должен быть заблокирован")
	}

	// имитируем истечение локлаута: last = 31 сек назад (в пределах окна 15 мин)
	ageRecord(t, ip, RateLimitLockout+time.Second)

	if RateLimited(ip) {
		t.Error("после истечения локлаута попытка должна быть разрешена (счётчик НЕ сбрасывается)")
	}
	// счётчик сохранился (5) → следующая неудача = мгновенный ре-лок
	RecordFailure(ip)
	if !RateLimited(ip) {
		t.Error("неудача после локлаута должна давать мгновенный ре-лок")
	}

	// сброс только при успехе
	ResetFailures(ip)
	if RateLimited(ip) {
		t.Fatal("после сброса IP не должен быть заблокирован")
	}
}

// ageRecord сдвигает timestamp записи назад (имитация прошедшего времени).
func ageRecord(t *testing.T, ip string, d time.Duration) {
	t.Helper()
	path := filepath.Join(RatelimitDir, sanitizeIP(ip))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fails int
	var last int64
	if _, err := fmt.Sscanf(string(data), "%d %d", &fails, &last); err != nil {
		t.Fatal(err)
	}
	newer := fmt.Sprintf("%d %d\n", fails, last-int64(d.Seconds()))
	if err := os.WriteFile(path, []byte(newer), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRatelimitIPv6Sanitize(t *testing.T) {
	dir := t.TempDir()
	RatelimitDir = dir
	ip := "fe80::1234:5678"

	RecordFailure(ip)
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("файл счётчика не создан: %v", err)
	}
	name := entries[0].Name()
	for _, ch := range name {
		if ch == ':' || ch == '/' {
			t.Fatalf("опасный символ в имени файла: %q", name)
		}
	}
	_ = RateLimited(ip)
	ResetFailures(ip)
}

func TestRatelimitUnknownIPAllowed(t *testing.T) {
	RatelimitDir = filepath.Join(t.TempDir(), "missing")
	if RateLimited("10.0.0.9") {
		t.Error("недоступный каталог → fail-open (разрешаем)")
	}
}

func TestRatelimitWindowExpiry(t *testing.T) {
	dir := t.TempDir()
	RatelimitDir = dir
	ip := "10.1.1.1"
	// вручную пишем протухшую запись: много неудач, но старше окна
	old := fmt.Sprintf("%d %d\n", RateLimitMaxFails, time.Now().Add(-2*RateLimitWindow).Unix())
	if err := os.WriteFile(filepath.Join(dir, sanitizeIP(ip)), []byte(old), 0600); err != nil {
		t.Fatal(err)
	}
	if RateLimited(ip) {
		t.Error("протухшая запись не должна блокировать")
	}
}
