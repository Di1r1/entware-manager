package stats

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"
)

func HandleLinksSave() {
	if os.Getenv("REQUEST_METHOD") != "POST" {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"POST required"}`)
		return
	}

	body, _ := io.ReadAll(os.Stdin)
	data := strings.TrimSpace(string(body))

	if !json.Valid([]byte(data)) {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Invalid JSON"}`)
		return
	}

	var links []Link
	if err := json.Unmarshal([]byte(data), &links); err != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Invalid links array"}`)
		return
	}
	if len(links) > 50 {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Too many links (max 50)"}`)
		return
	}
	for _, l := range links {
		if strings.TrimSpace(l.Name) == "" || !isSafeLinkURL(l.URL) || !isSafeLinkIcon(l.Icon) {
			fmt.Print("Content-type: application/json; charset=utf-8\n\n")
			fmt.Println(`{"status":"error","message":"Invalid link (name, URL scheme or icon)"}`)
			return
		}
	}

	if err := os.WriteFile("/opt/web_entware/links.json", []byte(data), 0644); err != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Failed to write links file"}`)
		return
	}

	logAction("INFO", fmt.Sprintf("Ссылки сохранены: %s", truncateJSON(data)))
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "ok", "message": "Links saved"})
}

func logAction(level, msg string) {
	logFile := fmt.Sprintf("/tmp/entware/logs/%s.log", time.Now().Format("2006-01-02"))
	ts := time.Now().Format("2006-01-02 15:04:05")
	ip := os.Getenv("REMOTE_ADDR")
	if ip == "" {
		ip = "0.0.0.0"
	}
	entry := fmt.Sprintf("[%s] [%s] [%s] [%d] [links_save.cgi] %s\n", ts, level, ip, os.Getpid(), msg)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(entry)
		f.Close()
	}
}

func truncateJSON(s string) string {
	if len(s) > 100 {
		return s[:100] + "..."
	}
	return s
}

// isSafeLinkURL допускает только http/https и относительные пути (/...).
// Блокирует javascript:, data:, vbscript: и протокольно-относительные //host.
func isSafeLinkURL(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" || len(u) > 2048 {
		return false
	}
	if strings.HasPrefix(u, "/") {
		return !strings.HasPrefix(u, "//")
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	switch parsed.Scheme {
	case "http", "https":
		return parsed.Host != ""
	default:
		return false
	}
}

// isSafeLinkIcon допускает только латиницу, цифры, "-" и "_".
func isSafeLinkIcon(s string) bool {
	if s == "" {
		return true
	}
	if len(s) > 32 {
		return false
	}
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			return false
		}
	}
	return true
}
