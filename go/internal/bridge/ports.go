// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Список слушающих TCP-портов из /proc/net/tcp(6) — для подсказки в сканере
// манифеста: пользователь видит, какие порты реально открыты на роутере,
// и может вставить порт в base одним кликом.
package bridge

import (
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

// LoopbackListeningPorts — уникальные отсортированные LISTEN-порты,
// достижимые с 127.0.0.1: слушатели на loopback или wildcard (0.0.0.0/[::]).
// Порты, привязанные к конкретному внешнему адресу, не включаются —
// подключение к ним с loopback отклоняется. Ошибка чтения procfs → пусто.
func LoopbackListeningPorts() []int {
	seen := map[int]bool{}
	for _, f := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n")[1:] {
			fields := strings.Fields(line)
			if len(fields) < 4 || fields[3] != "0A" { // 0A = LISTEN
				continue
			}
			i := strings.LastIndexByte(fields[1], ':')
			if i < 0 || i+1 >= len(fields[1]) {
				continue
			}
			if !reachableFromLoopback(fields[1][:i]) {
				continue
			}
			port, err := strconv.ParseInt(fields[1][i+1:], 16, 32)
			if err != nil || port < 1 || port > 65535 {
				continue
			}
			seen[int(port)] = true
		}
	}
	out := make([]int, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Ints(out)
	return out
}

// reachableFromLoopback — hex-адрес из /proc/net/tcp(6): true для
// 127.0.0.1 / 0.0.0.0 (tcp4, little-endian) и ::1 / :: (tcp6).
func reachableFromLoopback(hexIP string) bool {
	switch len(hexIP) {
	case 8: // IPv4, little-endian: 0100007F = 127.0.0.1
		return hexIP == "0100007F" || hexIP == "00000000"
	case 32: // IPv6 big-endian
		allZero := strings.Trim(hexIP, "0") == ""
		return allZero || strings.HasSuffix(hexIP, "1") && strings.Trim(hexIP[:len(hexIP)-1], "0") == ""
	default:
		return false
	}
}

// describeDialError — человекочитаемая причина неудачного подключения.
func describeDialError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "connection refused"):
		return "соединение отклонено — порт закрыт или сервис не запущен"
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "Client.Timeout") || strings.Contains(msg, "deadline exceeded"):
		return "таймаут подключения"
	default:
		return "сервис не отвечает"
	}
}

// describeHTTPError — человекочитаемая причина ответа 4xx/5xx.
func describeHTTPError(code int) string {
	switch code {
	case http.StatusUnauthorized, http.StatusForbidden:
		return "HTTP " + strconv.Itoa(code) + " — сервис требует авторизацию (задайте логин/пароль через форму на карточке модуля)"
	case http.StatusNotFound:
		return "HTTP 404 — путь не найден, проверьте адрес в манифесте"
	case http.StatusMethodNotAllowed:
		return "HTTP 405 — неверный метод для этого пути"
	default:
		return "HTTP " + strconv.Itoa(code)
	}
}
