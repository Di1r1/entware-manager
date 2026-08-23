// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Определение реального IP клиента за прокси/туннелем (KeenDNS, lighttpd).
package cgiutil

import (
	"net"
	"strings"
)

// ClientIP возвращает реальный IP посетителя.
//
// Прямой адрес (remoteAddr) используется как есть, кроме случая, когда
// запрос пришёл от доверенного источника (loopback / приватная подсеть /
// link-local) — тогда соединение терминирует локальный прокси или туннель
// (KeenDNS, mod_proxy), и реальный клиент берётся из X-Forwarded-For
// (первый элемент) либо X-Real-IP.
//
// Ограничение: устройство из LAN может подделать эти заголовки — для
// домашнего роутера компромисс осознанный; из интернета заголовки не
// действуют (публичный REMOTE_ADDR не является доверенным).
func ClientIP(remoteAddr, forwardedFor, realIP string) string {
	ra := stripPort(remoteAddr)
	if ra == "" {
		return ""
	}
	ip := net.ParseIP(ra)
	if ip == nil {
		return ra
	}
	if !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()) {
		return ra // прямой публичный адрес — заголовкам не верим
	}
	if f := firstValidIP(forwardedFor); f != "" {
		return f
	}
	if r := strings.TrimSpace(realIP); net.ParseIP(r) != nil {
		return r
	}
	return ra
}

// firstValidIP — первый валидный IP из списка X-Forwarded-For.
func firstValidIP(xff string) string {
	for _, part := range strings.Split(xff, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if net.ParseIP(part) != nil {
			return part
		}
	}
	return ""
}

// stripPort убирает порт из host:port / [v6]:port.
func stripPort(addr string) string {
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}
	return strings.TrimSpace(addr)
}
