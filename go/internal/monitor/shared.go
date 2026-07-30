package monitor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func WriteJSON(v any) {
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	json.NewEncoder(os.Stdout).Encode(v)
}

func WriteText(s string) {
	fmt.Print("Content-type: text/plain; charset=utf-8\n\n")
	fmt.Print(s)
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

func GetParam(key string) string {
	qs := os.Getenv("QUERY_STRING")
	params := parseFormBody(qs)
	if v, ok := params[key]; ok {
		return v
	}
	if IsPOST() {
		body := readPOSTBody()
		params = parseFormBody(body)
		if v, ok := params[key]; ok {
			return v
		}
	}
	return ""
}

func readPOSTBody() string {
	data, err := io.ReadAll(os.Stdin)
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
