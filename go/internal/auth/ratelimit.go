// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Антибрутфорс логина: файл-счётчик неудачных попыток в tmpfs.
//
// CGI-архитектура без общего состояния — счётчик хранится файлом
// /tmp/entware/ratelimit/<ip> (строка 1: счётчик, строка 2: unix ts последней
// неудачи). Записи с TTL: старше окна → сброс. Инкремент атомарный
// (уникальный temp + rename), гонки недоучитывают попытки — это сознательный
// компромисс defense-in-depth. Fail-open при недоступности /tmp: rate limit —
// вспомогательный барьер, основная защита (проверка пароля, fail-closed)
// от него не зависит.
package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// RatelimitDir — каталог счётчиков (переменная для тестов).
var RatelimitDir = "/tmp/entware/ratelimit"

// RateLimitMaxFails — число неудач до блокировки.
const RateLimitMaxFails = 5

// RateLimitWindow — TTL записи (неудачи старше окна не считаются).
const RateLimitWindow = 15 * time.Minute

// RateLimitLockout — длительность отказа после исчерпания попыток.
const RateLimitLockout = 30 * time.Second

// sanitizeIP делает имя файла безопасным: IPv6 содержит ':' — заменяем всё,
// кроме букв/цифр, на '_'.
func sanitizeIP(ip string) string {
	return strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return r
		}
		return '_'
	}, ip)
}

func rateFile(ip string) string {
	return filepath.Join(RatelimitDir, sanitizeIP(ip))
}

// RateLimited — true если вход с этого IP временно заблокирован.
func RateLimited(ip string) bool {
	data, err := os.ReadFile(rateFile(ip))
	if err != nil {
		return false // нет записи / /tmp недоступен → разрешаем (fail-open)
	}
	lines := strings.Fields(string(data))
	if len(lines) < 2 {
		return false
	}
	fails, err1 := strconv.Atoi(lines[0])
	last, err2 := strconv.ParseInt(lines[1], 10, 64)
	if err1 != nil || err2 != nil {
		return false
	}
	now := time.Now()
	if now.Unix()-last > int64(RateLimitWindow.Seconds()) {
		return false // запись протухла
	}
	if fails >= RateLimitMaxFails && now.Unix()-last <= int64(RateLimitLockout.Seconds()) {
		return true
	}
	// Локлаут истёк: разрешаем одну попытку, счётчик НЕ сбрасываем.
	// Следующая же неудача (RecordFailure обновит timestamp) — мгновенный
	// ре-лок. Раньше здесь был полный сброс → атакующий получал свежие
	// 5 попыток каждые ~35 сек (~8/мин); теперь ~2/мин максимум,
	// а успешный вход по-прежнему очищает счётчик полностью (ResetFailures).
	return false
}

// RecordFailure фиксирует неудачную попытку (атомарно, temp+rename).
func RecordFailure(ip string) {
	path := rateFile(ip)
	// каталог может отсутствовать после перезагрузки роутера (/tmp очищается)
	if os.MkdirAll(RatelimitDir, 0700) != nil {
		return // fail-open: rate limit — defense-in-depth
	}
	var fails int
	if data, err := os.ReadFile(path); err == nil {
		lines := strings.Fields(string(data))
		if len(lines) >= 2 {
			last, _ := strconv.ParseInt(lines[1], 10, 64)
			fails, _ = strconv.Atoi(lines[0])
			// старое окно → счётчик заново
			if time.Now().Unix()-last > int64(RateLimitWindow.Seconds()) {
				fails = 0
			}
		}
	}
	content := fmt.Sprintf("%d %d\n", fails+1, time.Now().Unix())
	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	if os.WriteFile(tmp, []byte(content), 0600) == nil {
		if os.Rename(tmp, path) != nil {
			os.Remove(tmp)
		}
	}
}

// ResetFailures удаляет счётчик (после успешного входа).
func ResetFailures(ip string) {
	_ = os.Remove(rateFile(ip))
}
