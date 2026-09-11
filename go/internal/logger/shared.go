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
// base=/entware-manager/ работает везде: классика (8087) напрямую,
// NOC (8086, iframe) — через прокси /entware-manager/ → 8087.
// Все asset-ссылки ниже должны быть ОТНОСИТЕЛЬНЫМИ (без ведущего /),
// чтобы резолвиться через <base>, а не через URL страницы
// (/entware-cgi/logger/ — там статики нет).
const staticBaseHTML = `<base id="static-base" href="/entware-manager/">`
