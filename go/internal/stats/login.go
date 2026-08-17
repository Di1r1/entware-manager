// Di1r1
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
		writeAuthJSON(map[string]interface{}{"status": "error", "message": "Запрос из недоверенного источника (CSRF)"})
		return
	}
	body, _ := io.ReadAll(os.Stdin)
	params := cgiutil.ParseFormBody(string(body))
	password := params["password"]

	allow, reason := auth.EnabledReports()
	if !allow {
		writeAuthJSON(map[string]interface{}{"status": "error", "message": reason})
		return
	}
	if !auth.CheckPassword(password) {
		// антибрутфорс: задержка перед ответом
		time.Sleep(1 * time.Second)
		writeAuthJSON(map[string]interface{}{"status": "error", "message": "Неверный пароль"})
		return
	}

	token, err := auth.CreateSession()
	if err != nil {
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

// writeAuthJSON выводит JSON с Content-Type.
func writeAuthJSON(v interface{}) {
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	json.NewEncoder(os.Stdout).Encode(v)
}
