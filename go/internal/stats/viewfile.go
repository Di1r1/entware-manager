package stats

import (
	_ "embed"
	"encoding/json"
	"entware-manager/internal/cgiutil"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"entware-manager/internal/auth"
)

//go:embed viewfile.html
var viewFileTemplate string

type ViewFileResponse struct {
	Status  string `json:"status"`
	Name    string `json:"name,omitempty"`
	Content string `json:"content,omitempty"`
	Message string `json:"message,omitempty"`
}

func HandleViewFile() {
	if os.Getenv("REQUEST_METHOD") != "POST" {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Метод не поддерживается"}`)
		return
	}

	if auth.IsCrossSiteOrigin() {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"` + auth.CrossSiteDeny + `"}`)
		return
	}

	body, _ := io.ReadAll(os.Stdin)
	params := cgiutil.ParseFormBody(string(body))
	path := params["path"]
	password := params["password"]
	isXHR := os.Getenv("HTTP_X_REQUESTED_WITH") != ""

	if path == "" || !isAllowedPath(path) {
		viewFileError("Доступ запрещен", isXHR)
		return
	}

	if !checkFilemgrAuth(password) {
		logViewFileAction("WARN", "Неверный пароль при просмотре файла: "+path)
		viewFileError("Неверный пароль", isXHR)
		return
	}

	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		viewFileError("Файл не найден", isXHR)
		return
	}

	if info.Size() > 1048576 {
		viewFileError("Файл слишком большой (макс. 1 MB)", isXHR)
		return
	}

	if info.Size() == 0 {
		viewFileResult(path, "", isXHR)
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		viewFileError("Ошибка чтения файла", isXHR)
		return
	}

	limit := 4096
	if len(data) < limit {
		limit = len(data)
	}
	if containsNullByte(data[:limit]) {
		viewFileError("Невозможно отобразить бинарный файл", isXHR)
		return
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	if len(lines) > 1000 {
		lines = lines[len(lines)-1000:]
	}
	content = strings.Join(lines, "\n")

	viewFileResult(path, content, isXHR)
}

func isAllowedPath(p string) bool {
	if p == "" {
		return false
	}
	if strings.Contains(p, "..") {
		return false
	}
	if filepath.Clean(p) != p {
		return false
	}
	return strings.HasPrefix(p, "/tmp/") || strings.HasPrefix(p, "/dev/shm/")
}

func containsNullByte(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}

func viewFileError(msg string, isXHR bool) {
	if isXHR {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		json.NewEncoder(os.Stdout).Encode(ViewFileResponse{Status: "error", Message: msg})
	} else {
		html := strings.NewReplacer(
			"{TITLE}", "Ошибка",
			"{CONTENT}", fmt.Sprintf(`<p class="error">%s</p>`, html.EscapeString(msg)),
			"{BACK_BUTTON}", backButton(),
		).Replace(viewFileTemplate)
		fmt.Print("Content-type: text/html; charset=utf-8\n\n")
		fmt.Print(html)
	}
}

func viewFileResult(path, content string, isXHR bool) {
	name := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		name = path[idx+1:]
	}

	if isXHR {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		json.NewEncoder(os.Stdout).Encode(ViewFileResponse{
			Status:  "ok",
			Name:    name,
			Content: content,
		})
		return
	}

	escaped := html.EscapeString(content)
	html := strings.NewReplacer(
		"{TITLE}", html.EscapeString(name),
		"{CONTENT}", `<pre class="file-viewer-content">`+escaped+`</pre>`,
		"{BACK_BUTTON}", backButton(),
	).Replace(viewFileTemplate)
	fmt.Print("Content-type: text/html; charset=utf-8\n\n")
	fmt.Print(html)
}

func backButton() string {
	return `<p style="margin-top:1rem;"><a href="javascript:history.back()" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=3#icon-arrow-left"/></svg> Назад</a></p>`
}

// logViewFileAction пишет событие (например, неверный пароль) в суточный лог
// с тегом [view_file.cgi] — его читает Telegram-шлюз (source=system).
func logViewFileAction(level, msg string) {
	logFile := fmt.Sprintf("/tmp/entware/logs/%s.log", time.Now().Format("2006-01-02"))
	ts := time.Now().Format("2006-01-02 15:04:05")
	ip := os.Getenv("REMOTE_ADDR")
	if ip == "" {
		ip = "0.0.0.0"
	}
	entry := fmt.Sprintf("[%s] [%s] [%s] [%d] [view_file.cgi] %s\n", ts, level, ip, os.Getpid(), msg)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(entry)
		f.Close()
	}
}
