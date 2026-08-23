// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Логин/логаут панели (login.cgi, logout.cgi, session.cgi).
//
// Вход в панель по паролю из auth_config.json (тот же, что для мутаций).
// Успешный вход создаёт файл-сессию (см. internal/auth/session.go) и отдаёт
// Set-Cookie: panel_session=...; HttpOnly; SameSite=Strict; Path=/.
// При enabled=false панель открыта (логин не требуется) — обратная совместимость.
package stats

import (
	"encoding/json"
	"entware-manager/internal/cgiutil"
	"fmt"
	"io"
	"os"
	"time"

	"entware-manager/internal/auth"
)

// HandleLogin обрабатывает login.cgi (POST: password, action=login).
func HandleLogin() {
	if os.Getenv("REQUEST_METHOD") != "POST" {
		writeAuthJSON(map[string]interface{}{"status": "error", "message": "Метод не поддерживается"})
		return
	}
	if auth.IsCrossSiteOrigin() {
		logAuthAction("WARN", "Запрос из недоверенного источника (CSRF)")
		writeAuthJSON(map[string]interface{}{"status": "error", "message": "Запрос из недоверенного источника (CSRF)"})
		return
	}
	body, _ := io.ReadAll(os.Stdin)
	params := cgiutil.ParseFormBody(string(body))
	password := params["password"]

	// IP клиента (за прокси может быть 127.0.0.1 — общий bucket, компромисс).
	ip := os.Getenv("REMOTE_ADDR")
	if ip == "" {
		ip = "0.0.0.0"
	}

	allow, reason := auth.EnabledReports()
	if !allow {
		logAuthAction("WARN", "Вход отклонён: "+reason)
		writeAuthJSON(map[string]interface{}{"status": "error", "message": reason})
		return
	}
	// Антибрутфорс: после 5 неудач подряд — отказ на 30 сек.
	if auth.RateLimited(ip) {
		time.Sleep(1 * time.Second)
		logAuthAction("WARN", "Вход заблокирован: исчерпаны попытки ("+ip+")")
		writeAuthJSON(map[string]interface{}{"status": "error", "message": "Слишком много неудачных попыток. Повторите через 30 секунд"})
		return
	}
	if !auth.CheckPassword(password) {
		auth.RecordFailure(ip)
		// антибрутфорс: задержка перед ответом
		time.Sleep(1 * time.Second)
		logAuthAction("WARN", "Неверный пароль при входе")
		writeAuthJSON(map[string]interface{}{"status": "error", "message": "Неверный пароль"})
		return
	}
	auth.ResetFailures(ip)

	migratePasswordHash(password)
	logAuthAction("INFO", "Успешный вход")
	token, err := auth.CreateSession()
	if err != nil {
		logAuthAction("WARN", "Не удалось создать сессию")
		writeAuthJSON(map[string]interface{}{"status": "error", "message": "Не удалось создать сессию"})
		return
	}
	fmt.Print("Content-type: application/json; charset=utf-8\n")
	fmt.Printf("Set-Cookie: %s=%s; Path=/; HttpOnly; SameSite=Strict\n", auth.SessionCookieName, token)
	fmt.Print("\n")
	fmt.Println(`{"status":"ok","message":"Вход выполнен"}`)
}

// HandleLogout обрабатывает logout.cgi (POST).
func HandleLogout() {
	if os.Getenv("REQUEST_METHOD") != "POST" {
		writeAuthJSON(map[string]interface{}{"status": "error", "message": "Метод не поддерживается"})
		return
	}
	auth.DestroySession()
	fmt.Print("Content-type: application/json; charset=utf-8\n")
	fmt.Printf("Set-Cookie: %s=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0\n", auth.SessionCookieName)
	fmt.Print("\n")
	fmt.Println(`{"status":"ok","message":"Выход выполнен"}`)
}

// HandleSession обрабатывает session.cgi (GET) — статус авторизации для фронта.
func HandleSession() {
	// Панель открыта тогда и только тогда, когда открыт гейт (то же условие,
	// что в cgi.go): если защита включена — нужна валидная сессия.
	authenticated := !auth.Enabled() || auth.SessionValid()
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	fmt.Printf(`{"authenticated":%v}`+"\n", authenticated)
}

// migratePasswordHash прозрачно перехеширует legacy-хеш (голый sha256-hex)
// в PBKDF2+соль после успешного входа. Атомарно (уникальный temp+rename),
// поле enabled сохраняется as-is; мигрируем только активную защиту.
func migratePasswordHash(password string) {
	cfg, ok := auth.LoadConfig()
	if !ok || !cfg.Enabled || cfg.PasswordHash == "" || !auth.NeedsRehash(cfg.PasswordHash) {
		return
	}
	newHash := auth.HashPassword(password)
	if newHash == "" {
		return // crypto/rand недоступен — остаёмся на legacy
	}
	out, _ := json.MarshalIndent(struct {
		Enabled      bool   `json:"enabled"`
		PasswordHash string `json:"password_hash"`
	}{Enabled: cfg.Enabled, PasswordHash: newHash}, "", "    ")
	out = append(out, '\n')
	tmp := fmt.Sprintf("%s.%d.tmp", auth.ConfigPath, os.Getpid())
	if os.WriteFile(tmp, out, 0600) == nil {
		if os.Rename(tmp, auth.ConfigPath) != nil {
			os.Remove(tmp)
		}
	}
}

// writeAuthJSON выводит JSON с Content-Type.
func writeAuthJSON(v interface{}) {
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	json.NewEncoder(os.Stdout).Encode(v)
}

// logAuthAction пишет запись о попытке входа в суточный лог защищённых действий
// /tmp/entware/logs/<дата>.log. Формат строки единый для статистики:
// [время] [уровень] [IP клиента] [pid] [login.cgi] сообщение.
func logAuthAction(level, msg string) {
	logFile := fmt.Sprintf("/tmp/entware/logs/%s.log", time.Now().Format("2006-01-02"))
	ts := time.Now().Format("2006-01-02 15:04:05")
	ip := os.Getenv("REMOTE_ADDR")
	if ip == "" {
		ip = "0.0.0.0"
	}
	entry := fmt.Sprintf("[%s] [%s] [%s] [%d] [login.cgi] %s\n", ts, level, ip, os.Getpid(), msg)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(entry)
		f.Close()
	}
}
