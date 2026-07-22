package stats

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func HandleDeleteFile() {
	if os.Getenv("REQUEST_METHOD") != "POST" {
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		fmt.Println(`{"status":"error","message":"Метод не поддерживается"}`)
		return
	}

	body, _ := io.ReadAll(os.Stdin)
	params := parsePostForm(string(body))
	path := params["path"]
	password := params["password"]

	if !isAllowedDeletePath(path) {
		logDeleteAction("WARN", fmt.Sprintf("Попытка удаления с недопустимым путём: %s", path))
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		fmt.Println(`{"status":"error","message":"Доступ запрещен"}`)
		return
	}

	if !checkFilemgrAuth(password) {
		logDeleteAction("WARN", fmt.Sprintf("Неверный пароль при удалении: %s", path))
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		fmt.Println(`{"status":"error","message":"Неверный пароль"}`)
		return
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		logDeleteAction("WARN", fmt.Sprintf("Попытка удаления несуществующего объекта: %s", path))
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		fmt.Println(`{"status":"error","message":"Файл/папка не существует"}`)
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		fmt.Println(`{"status":"error","message":"Ошибка доступа"}`)
		return
	}

	if info.IsDir() {
		if err := os.Remove(path); err != nil {
			logDeleteAction("WARN", fmt.Sprintf("Не удалось удалить папку (не пуста): %s", path))
			fmt.Println("Content-type: application/json; charset=utf-8\n")
			fmt.Println(`{"status":"error","message":"Папка не пуста, удаление отменено"}`)
			return
		}
		logDeleteAction("INFO", fmt.Sprintf("Удалена пустая папка: %s", path))
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		fmt.Println(`{"status":"ok"}`)
		return
	}

	if err := os.Remove(path); err != nil {
		logDeleteAction("WARN", fmt.Sprintf("Не удалось удалить файл: %s", path))
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		fmt.Println(`{"status":"error","message":"Не удалось удалить файл"}`)
		return
	}
	logDeleteAction("INFO", fmt.Sprintf("Удалён файл: %s", path))
	fmt.Println("Content-type: application/json; charset=utf-8\n")
	fmt.Println(`{"status":"ok"}`)
}

func isAllowedDeletePath(p string) bool {
	return p == "/tmp" || p == "/dev/shm" ||
		strings.HasPrefix(p, "/tmp/") || strings.HasPrefix(p, "/dev/shm/")
}

func checkFilemgrAuth(password string) bool {
	data, err := os.ReadFile("/opt/web_entware/auth_config.json")
	if err != nil {
		return true
	}
	var cfg struct {
		Enabled      bool   `json:"enabled"`
		PasswordHash string `json:"password_hash"`
		Password     string `json:"password"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return true
	}
	if !cfg.Enabled {
		return true
	}
	if cfg.PasswordHash != "" {
		return sha256Hex(password) == cfg.PasswordHash
	}
	if cfg.Password != "" {
		return password == cfg.Password
	}
	return true
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func parsePostForm(body string) map[string]string {
	params := make(map[string]string)
	for _, part := range strings.Split(body, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			key := urlDecode(kv[0])
			val := urlDecode(kv[1])
			params[key] = val
		}
	}
	return params
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
		if s[i] == '+' {
			b.WriteByte(' ')
			continue
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

func logDeleteAction(level, msg string) {
	logFile := fmt.Sprintf("/tmp/entware/logs/%s.log", time.Now().Format("2006-01-02"))
	ts := time.Now().Format("2006-01-02 15:04:05")
	ip := os.Getenv("REMOTE_ADDR")
	if ip == "" {
		ip = "0.0.0.0"
	}
	entry := fmt.Sprintf("[%s] [%s] [%s] [%d] [delete_file.cgi] %s\n", ts, level, ip, os.Getpid(), msg)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(entry)
		f.Close()
	}
}
