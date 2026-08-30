// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Список слушающих TCP-портов из /proc/net/tcp(6) — для подсказки в сканере
// манифеста: пользователь видит, какие порты реально открыты на роутере,
// и может вставить порт в base одним кликом. Плюс база подписей известных
// портов (portHints) — помогает опознать сервис по номеру порта.
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
// 127.0.0.1 / 0.0.0.0 (tcp4) и ::1 / :: (tcp6).
// tcp6-строка — 4 u32-слова: на little-endian (arm64/mipsel) каждое слово
// побайтово перевёрнуто (::1 = …01000000), на big-endian (mips) — прямая
// запись (::1 = …00000001); допускаем оба представления.
func reachableFromLoopback(hexIP string) bool {
	switch len(hexIP) {
	case 8: // IPv4: 0100007F = 127.0.0.1 (little-endian), 00000000 = 0.0.0.0
		return hexIP == "0100007F" || hexIP == "00000000"
	case 32:
		return ipv6IsSpecialLE(hexIP) || ipv6IsSpecialBE(hexIP)
	default:
		return false
	}
}

// ipv6IsSpecialBE — прямое прочтение пар hex (big-endian ядро).
func ipv6IsSpecialBE(h string) bool {
	var b [16]byte
	for i := 0; i < 16; i++ {
		v, err := strconv.ParseUint(h[i*2:i*2+2], 16, 8)
		if err != nil {
			return false
		}
		b[i] = byte(v)
	}
	return v6AllZero(b) || v6Loopback(b)
}

// ipv6IsSpecialLE — слова по 8 hex как little-endian u32 (x86/arm64/mipsel).
func ipv6IsSpecialLE(h string) bool {
	var b [16]byte
	for w := 0; w < 4; w++ {
		v, err := strconv.ParseUint(h[w*8:(w+1)*8], 16, 64)
		if err != nil {
			return false
		}
		b[w*4+0] = byte(v)
		b[w*4+1] = byte(v >> 8)
		b[w*4+2] = byte(v >> 16)
		b[w*4+3] = byte(v >> 24)
	}
	return v6AllZero(b) || v6Loopback(b)
}

func v6AllZero(b [16]byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}

func v6Loopback(b [16]byte) bool {
	for i := 0; i < 15; i++ {
		if b[i] != 0 {
			return false
		}
	}
	return b[15] == 1
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

// portHints — база известных портов: порт → короткая подпись сервиса.
// Список намеренно консервативный — только порты с уверенной привязкой.
var portHints = map[int]string{
	22:    "SSH",
	53:    "DNS",
	80:    "HTTP (веб-интерфейс)",
	139:   "SMB (NetBIOS)",
	222:   "SSH (Entware)",
	443:   "HTTPS",
	445:   "SMB",
	1080:  "SOCKS-прокси",
	1081:  "SOCKS-прокси",
	1900:  "SSDP (UPnP)",
	3702:  "WS-Discovery",
	51413: "BitTorrent (Transmission)",
	6800:  "aria2 RPC",
	7681:  "ttyd (терминал)",
	8080:  "AdGuard / веб-сервис",
	8086:  "InfluxDB",
	8087:  "Панель Entware Manager",
	8090:  "Transmission Web UI",
	8123:  "Home Assistant",
	8384:  "Syncthing (веб)",
	8443:  "Панель Entware Manager (HTTPS)",
	9091:  "Transmission RPC",
	9097:  "Веб-панель (Koffe)",
	10051: "Zabbix-агент",
	19999: "Netdata",
	22000: "Syncthing (синхронизация)",
}

// PortLabel — короткая подпись известного порта ("", если не в базе).
func PortLabel(port int) string {
	return portHints[port]
}

// PortLabelsDict — копия всей базы подписей для предзаполнения в UI
// (отдаётся в bridge_discover и bridge_probe один раз; карта возвращается
// копией, чтобы вызывающий не мог мутировать исходник).
func PortLabelsDict() map[int]string {
	out := make(map[int]string, len(portHints))
	for p, l := range portHints {
		out[p] = l
	}
	return out
}
