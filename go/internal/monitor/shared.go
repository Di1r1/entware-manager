package monitor

import (
	"fmt"

	"entware-manager/internal/cgiutil"
)

// WriteText выводит текст (text/plain) — используется лог-эндпоинтом.
func WriteText(s string) {
	fmt.Print("Content-type: text/plain; charset=utf-8\n\n")
	fmt.Print(s)
}

// GetParam извлекает параметр из QUERY_STRING или тела POST.
func GetParam(key string) string {
	return cgiutil.GetParam(key)
}
