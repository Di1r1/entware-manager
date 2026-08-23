package stats

import (
	"crypto/sha256"
	"encoding/json"
	"entware-manager/internal/cgiutil"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"entware-manager/internal/auth"
)

func HandleDeleteFile() {
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

	if !isAllowedDeletePath(path) {
		logDeleteAction("WARN", fmt.Sprintf("Попытка удаления с недопустимым путём: %s", path))
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Доступ запрещен"}`)
		return
	}

	if !checkFilemgrAuth(password) {
		logDeleteAction("WARN", fmt.Sprintf("Неверный пароль при удалении: %s", path))
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Неверный пароль"}`)
		return
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		logDeleteAction("WARN", fmt.Sprintf("Попытка удаления несуществующего объекта: %s", path))
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Файл/папка не существует"}`)
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Ошибка доступа"}`)
		return
	}

	if info.IsDir() {
		if err := os.Remove(path); err != nil {
			logDeleteAction("WARN", fmt.Sprintf("Не удалось удалить папку (не пуста): %s", path))
			fmt.Print("Content-type: application/json; charset=utf-8\n\n")
			fmt.Println(`{"status":"error","message":"Папка не пуста, удаление отменено"}`)
			return
		}
		logDeleteAction("INFO", fmt.Sprintf("Удалена пустая папка: %s", path))
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"ok"}`)
		return
	}

	if err := os.Remove(path); err != nil {
		logDeleteAction("WARN", fmt.Sprintf("Не удалось удалить файл: %s", path))
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Не удалось удалить файл"}`)
		return
	}
	logDeleteAction("INFO", fmt.Sprintf("Удалён файл: %s", path))
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	fmt.Println(`{"status":"ok"}`)
}

func isAllowedDeletePath(p string) bool {
	if p == "" {
		return false
	}
	if strings.Contains(p, "..") {
		return false
	}
	if filepath.Clean(p) != p {
		return false
	}
	return p == "/tmp" || p == "/dev/shm" ||
		strings.HasPrefix(p, "/tmp/") || strings.HasPrefix(p, "/dev/shm/")
}

var authConfigFile = authConfigPath
var authMarkerFile = authMarkerPath

func checkFilemgrAuth(password string) bool {
	data, err := os.ReadFile(authConfigFile)
	if err != nil {
		// Конфига нет: если панель уже защищалась (marker) — fail-closed
		// (битый/удалённый конфиг не снимает защиту). Первичная установка
		// без пароля — панель открыта по умолчанию, разрешаем.
		if _, serr := os.Stat(authMarkerFile); serr == nil {
			return false
		}
		return true
	}
	var cfg struct {
		Enabled      bool   `json:"enabled"`
		PasswordHash string `json:"password_hash"`
		Password     string `json:"password"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		// Битый конфиг — fail-closed, не открываем удаление без пароля.
		return false
	}
	if !cfg.Enabled {
		return true
	}
	if cfg.PasswordHash != "" {
		// VerifyPassword понимает legacy sha256-hex и новый PBKDF2+соль.
		return auth.VerifyPassword(password, cfg.PasswordHash)
	}
	if cfg.Password != "" {
		return password == cfg.Password
	}
	// enabled=true, но пароль не задан ни в виде hash, ни plain — панель
	// настраивали, доступа без пароля быть не должно.
	return false
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
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
