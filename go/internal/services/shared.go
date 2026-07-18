package services

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func WriteJSON(v any) {
	fmt.Println("Content-type: application/json; charset=utf-8\n")
	json.NewEncoder(os.Stdout).Encode(v)
}

func WriteError(msg string) {
	WriteJSON(map[string]string{"error": msg})
}

func NotAllowed() {
	WriteJSON(map[string]string{"error": "Method not allowed"})
}

func IsGET() bool {
	return os.Getenv("REQUEST_METHOD") == "GET"
}

func IsPOST() bool {
	return os.Getenv("REQUEST_METHOD") == "POST"
}

func getQueryParam(key string) string {
	q := os.Getenv("QUERY_STRING")
	for _, part := range strings.Split(q, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == key {
			val := kv[1]
			val = strings.ReplaceAll(val, "+", " ")
			val = urlDecode(val)
			return val
		}
	}
	return ""
}

func urlDecode(s string) string {
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

func readPOSTBody() string {
	data, err := os.ReadFile("/dev/stdin")
	if err != nil {
		return ""
	}
	return string(data)
}

func parseFormBody(body string) map[string]string {
	params := make(map[string]string)
	for _, part := range strings.Split(body, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			key := urlDecode(strings.ReplaceAll(kv[0], "+", " "))
			val := urlDecode(strings.ReplaceAll(kv[1], "+", " "))
			params[key] = val
		}
	}
	return params
}
