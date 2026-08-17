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
