// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// SSRF-гейт моста сервисов: единая проверка ВСЕХ URL манифеста
// (probe/status/actions). Разрешены только http-адреса на литеральном
// loopback; userinfo, фрагменты и редиректы запрещены. Относительные URL
// резолвятся от базового адреса манифеста и повторно прогоняются через гейт.
package bridge

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateBridgeURL проверяет URL цели проксирования.
// base — базовый URL манифеста для резолва относительных путей (может быть пустым).
func ValidateBridgeURL(raw, base string) (*url.URL, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("пустой URL")
	}
	if len(raw) > 2048 {
		return nil, fmt.Errorf("слишком длинный URL")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("не парсится: %w", err)
	}

	// Относительный путь → резолвим от базы и проверяем результат заново.
	if u.Host == "" {
		if base == "" {
			return nil, fmt.Errorf("относительный URL без базы запрещён")
		}
		b, err := url.Parse(base)
		if err != nil || b.Host == "" {
			return nil, fmt.Errorf("битая база манифеста")
		}
		resolved := b.ResolveReference(u)
		return ValidateBridgeURL(resolved.String(), "")
	}

	if u.Scheme != "http" {
		return nil, fmt.Errorf("схема %q запрещена — только http", u.Scheme)
	}
	if u.User != nil {
		return nil, fmt.Errorf("userinfo в URL запрещён")
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("фрагмент в URL запрещён")
	}

	host := u.Hostname() // без порта, скобки IPv6 сняты
	if strings.EqualFold(host, "localhost") {
		host = "127.0.0.1"
		u.Host = net.JoinHostPort(host, u.Port())
	}
	if host != "127.0.0.1" && host != "::1" {
		return nil, fmt.Errorf("хост %q разрешён только как 127.0.0.1", host)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return nil, fmt.Errorf("хост не loopback")
	}

	port := u.Port()
	if port == "" {
		return nil, fmt.Errorf("порт обязателен")
	}
	var pn int
	for _, ch := range port {
		if ch < '0' || ch > '9' {
			return nil, fmt.Errorf("некорректный порт")
		}
		pn = pn*10 + int(ch-'0')
		if pn > 65535 {
			return nil, fmt.Errorf("некорректный порт")
		}
	}
	if pn < 1 {
		return nil, fmt.Errorf("некорректный порт")
	}
	return u, nil
}

// noRedirects — транспорт моста не следует редиректам: 302 с loopback на
// произвольный хост пробил бы гейт.
const noRedirects = "redirects disabled"
