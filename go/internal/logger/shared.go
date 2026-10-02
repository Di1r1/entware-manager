package logger

import (
	"os"
	"strings"
)

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func hasQuery(key string) bool {
	q := os.Getenv("QUERY_STRING")
	return strings.Contains(q, key)
}

// staticBaseHTML — <base> для статики.
// base=/entware-manager/ работает везде: классика и NOC на едином origin
// (:8087 — entware-server в go-режиме, общий lighttpd в lighttpd-режиме).
// Все asset-ссылки ниже должны быть ОТНОСИТЕЛЬНЫМИ (без ведущего /),
// чтобы резолвиться через <base>, а не через URL страницы
// (/entware-cgi/logger/ — там статики нет).
const staticBaseHTML = `<base href="/entware-manager/">`
