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
	fmt.Println("Content-type: text/html; charset=utf-8\n")
	fmt.Print(html)
}

func writeJSON(v any) {
	fmt.Println("Content-type: application/json; charset=utf-8\n")
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
			return kv[1]
		}
	}
	return ""
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
