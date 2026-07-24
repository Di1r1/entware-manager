package stats

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func HandleLinksSave() {
	if os.Getenv("REQUEST_METHOD") != "POST" {
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		fmt.Println(`{"status":"error","message":"POST required"}`)
		return
	}

	body, _ := io.ReadAll(os.Stdin)
	data := strings.TrimSpace(string(body))

	if !json.Valid([]byte(data)) {
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		fmt.Println(`{"status":"error","message":"Invalid JSON"}`)
		return
	}

	if err := os.WriteFile("/opt/web_entware/links.json", []byte(data), 0644); err != nil {
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		fmt.Println(`{"status":"error","message":"Failed to write links file"}`)
		return
	}

	logAction("INFO", fmt.Sprintf("Ссылки сохранены: %s", truncateJSON(data)))
	fmt.Println("Content-type: application/json; charset=utf-8\n")
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
