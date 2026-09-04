// Package cgiutil — общий CGI-пламбинг для бинарников Entware Manager
// (единый источник вместо дублей в пакетах rdp/logger/monitor/network/
// packages/services/smart/stats).
package cgiutil

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// WriteJSON выводит JSON-ответ CGI.
func WriteJSON(v any) {
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	json.NewEncoder(os.Stdout).Encode(v)
}

// WriteError выводит ошибку в формате {"error": msg}
// (контракт фронта entware.js / smart.js для большинства эндпоинтов).
func WriteError(msg string) {
	WriteJSON(map[string]string{"error": msg})
}

// WriteStatusError выводит ошибку в формате {"status":"error","message":msg}
// (контракт фронта rdp.js / tmpfs.html).
func WriteStatusError(msg string) {
	WriteJSON(map[string]string{"status": "error", "message": msg})
}

// NotAllowed выводит стандартный ответ «метод не поддерживается».
func NotAllowed() {
	WriteError("Method not allowed")
}

// IsGET возвращает true, если запрос пришёл методом GET.
func IsGET() bool {
	return os.Getenv("REQUEST_METHOD") == "GET"
}

// IsPOST возвращает true, если запрос пришёл методом POST.
func IsPOST() bool {
	return os.Getenv("REQUEST_METHOD") == "POST"
}

// ReadPOSTBody читает всё тело POST-запроса из stdin.
func ReadPOSTBody() string {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return ""
	}
	return string(data)
}

// ParseFormBody разбирает тело application/x-www-form-urlencoded
// в карту «ключ → значение» (пробелы '+' и %XX декодируются).
func ParseFormBody(body string) map[string]string {
	params := make(map[string]string)
	for _, part := range strings.Split(body, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			key := URLDecode(strings.ReplaceAll(kv[0], "+", " "))
			val := URLDecode(strings.ReplaceAll(kv[1], "+", " "))
			params[key] = val
		}
	}
	return params
}

// GetQueryParam извлекает параметр из QUERY_STRING (GET).
func GetQueryParam(key string) string {
	q := os.Getenv("QUERY_STRING")
	for _, part := range strings.Split(q, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == key {
			val := strings.ReplaceAll(kv[1], "+", " ")
			return URLDecode(val)
		}
	}
	return ""
}

// GetParam извлекает параметр сначала из QUERY_STRING, затем из тела POST
// (комбо для эндпоинтов, принимающих оба способа).
func GetParam(key string) string {
	if v := GetQueryParam(key); v != "" {
		return v
	}
	if IsPOST() {
		if v, ok := ParseFormBody(ReadPOSTBody())[key]; ok {
			return v
		}
	}
	return ""
}

// URLDecode декодирует %XX-последовательности (без преобразования '+').
func URLDecode(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			high := unhex(s[i+1])
			low := unhex(s[i+2])
			if high >= 0 && low >= 0 {
				sb.WriteByte(byte(high<<4 | low))
				i += 2
				continue
			}
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}

func unhex(c byte) int {
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c && c <= 'f':
		return int(c - 'a' + 10)
	case 'A' <= c && c <= 'F':
		return int(c - 'A' + 10)
	}
	return -1
}

// WriteFileAtomic атомарно записывает файл: данные пишутся во временный
// файл в том же каталоге, затем temp переименовывается поверх целевого.
// При ошибке переименования временный файл удаляется.
func WriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// HumanSize форматирует размер в человеко-читаемый вид (B/K/M/G), как в UI панели.
func HumanSize(size int64) string {
	switch {
	case size < 1024:
		return strconv.FormatInt(size, 10) + "B"
	case size < 1048576:
		return strconv.FormatInt(size/1024, 10) + "K"
	case size < 1073741824:
		return strconv.FormatInt(size/1048576, 10) + "M"
	default:
		return strconv.FormatInt(size/1073741824, 10) + "G"
	}
}
