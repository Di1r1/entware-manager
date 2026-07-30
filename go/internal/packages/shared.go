package packages

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var reSanitize = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func sanitizePkg(name string) string {
	return reSanitize.ReplaceAllString(name, "")
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func writeHTML(html string) {
	fmt.Print("Content-type: text/html; charset=utf-8\n\n")
	fmt.Print(html)
}

func writeJSON(v any) {
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	json.NewEncoder(os.Stdout).Encode(v)
}

func WriteError(msg string) {
	writeJSON(map[string]string{"error": msg})
}

func methodNotAllowed() {
	WriteError("Method not allowed")
}

func isGET() bool {
	return os.Getenv("REQUEST_METHOD") == "GET"
}

func isPOST() bool {
	return os.Getenv("REQUEST_METHOD") == "POST"
}

func logPackageChange(action, pkg, status string) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	os.MkdirAll("/opt/var/log", 0755)
	f, err := os.OpenFile("/opt/var/log/package_changes.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s | %s | %s | %s\n", ts, action, pkg, status)
}

func readBody() string {
	b, _ := io.ReadAll(os.Stdin)
	return string(b)
}

func parsePostParam(body, key string) string {
	for _, part := range strings.Split(body, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == key {
			return urlDecode(strings.ReplaceAll(kv[1], "+", " "))
		}
	}
	return ""
}

func urlDecode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			high := unhex(s[i+1])
			low := unhex(s[i+2])
			if high >= 0 && low >= 0 {
				b.WriteByte(byte(high<<4 | low))
				i += 2
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
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

func runOpkg(args ...string) (string, int) {
	return runCmd("/opt/bin/opkg", args...)
}

func runCmd(cmd string, args ...string) (string, int) {
	c := exec.Command(cmd, args...)
	out, err := c.CombinedOutput()
	if err != nil {
		return string(out), 1
	}
	return string(out), 0
}
