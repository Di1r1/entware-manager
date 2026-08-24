// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Сессии приложений для моста: авто-логин при 401 и персистентный кэш cookie.
//
// CGI-процессы одноразовые — jar в памяти не живёт между запросами. Поэтому
// cookie приложения хранится файлом /tmp/entware/bridge/<id>.session (0600,
// tmpfs). Поток: запрос с сохранённой cookie → если 401 и есть creds типа
// cookie_login → логин (пароль из <id>.auth.json) → сохранить новую cookie →
// повторить запрос один раз.
package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Extend AuthCreds (manifest.go): Type="cookie_login", Password = пароль
// приложения, LoginURL = относительный адрес логина.
func sessionPath(dir, id string) string {
	return filepath.Join("/tmp/entware/bridge", sanitize(id)+".session")
}

// BridgeDir — каталог манифестов (для cmd-обработчиков).
func BridgeDir() string { return bridgeDirVar }

// SetBridgeDir — переопределение для тестов.
func SetBridgeDir(dir string) { bridgeDirVar = dir }

// SaveAuth атомарно пишет секреты коннектора (0600).
func SaveAuth(dir, id string, creds AuthCreds) error {
	if !ValidID(id) {
		return fmt.Errorf("плохой id")
	}
	os.MkdirAll(dir, 0755)
	out, err := json.MarshalIndent(creds, "", "    ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	tmp := filepath.Join(dir, id+".auth.json.tmp")
	if err := os.WriteFile(tmp, out, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(dir, id+".auth.json")); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// ClearSession удаляет сохранённую cookie приложения.
func ClearSession(dir, id string) {
	os.Remove(sessionPath(dir, id))
}

func loadSessionCookie(dir, id string) string {
	data, err := os.ReadFile(sessionPath(dir, id))
	if err != nil {
		return ""
	}
	parts := strings.SplitN(strings.TrimSpace(string(data)), "|", 2)
	name := "session"
	if len(parts) == 2 {
		name = parts[0]
	}
	value := parts[len(parts)-1]
	if name == "" || value == "" {
		return ""
	}
	return name + "=" + value
}

func saveSessionCookie(dir, id string, resp *http.Response) {
	for _, c := range resp.Cookies() {
		if c.Value == "" {
			continue
		}
		content := c.Name + "|" + c.Value
		os.MkdirAll(filepath.Dir(sessionPath(dir, id)), 0700)
		tmp := sessionPath(dir, id) + ".tmp"
		if os.WriteFile(tmp, []byte(content), 0600) == nil {
			os.Rename(tmp, sessionPath(dir, id))
		}
		return
	}
}

// authedDo выполняет запрос с авто-логином: cookie из файла → при 401 логин
// по creds → retry один раз. Возвращает финальный ответ (resp.Close на вызывающем).
func authedDo(client *http.Client, dir, id string, method, url, body string) (*http.Response, error) {
	creds := LoadAuth(dir, id)
	// ВАЖНО: контекст НЕ должен действовать на чтение тела ответа — иначе
	// отмена после возврата do() обрывает большие тела (истории) посреди
	// потока. Таймауты на соединение/заголовки задаёт Transport (clientBridge);
	// объём тела ограничивает вызывающий через LimitReader.
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	do := func(extraHdr, cookie string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(context.Background(), method, url, bodyReader)
		if err != nil {
			return nil, err
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if cookie != "" {
			req.Header.Set("Cookie", cookie)
		}
		if extraHdr != "" {
			// формат "Имя|Значение" из файла сессии (напр. X-Transmission-Session-Id)
			if i := strings.Index(extraHdr, "|"); i > 0 {
				req.Header.Set(extraHdr[:i], extraHdr[i+1:])
			}
		}
		if creds != nil && creds.Type == "basic" {
			req.SetBasicAuth(creds.Username, creds.Password)
		}
		return client.Do(req)
	}

	resp, err := do("", loadStoredSession(dir, id))
	if err != nil {
		return resp, err
	}
	// Transmission-стиль: 409 + X-Transmission-Session-Id → сохранить и повторить.
	if resp.StatusCode == http.StatusConflict {
		tok := resp.Header.Get("X-Transmission-Session-Id")
		if tok != "" {
			saveNamedSession(dir, id, "X-Transmission-Session-Id", tok)
			io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			return do("X-Transmission-Session-Id|"+tok, loadStoredSession(dir, id))
		}
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	resp.Body.Close()

	if creds == nil || creds.Type != "cookie_login" || creds.Password == "" || creds.LoginURL == "" {
		return resp, nil // отдаём исходный 401
	}
	lu, err := ValidateBridgeURL(creds.LoginURL, baseOf(url))
	if err != nil {
		return resp, nil
	}
	loginResp, err := client.Post(lu.String(), "application/json",
		strings.NewReader(fmt.Sprintf(`{"password":%q}`, creds.Password)))
	if err != nil {
		return &http.Response{StatusCode: http.StatusUnauthorized}, nil
	}
	defer loginResp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(loginResp.Body, 4096))
	saveSessionCookie(dir, id, loginResp)

	// Повтор исходного запроса со свежей cookie
	return do("", loadSessionCookie(dir, id))
}

func baseOf(url string) string {
	if i := strings.Index(url, "?"); i > 0 {
		url = url[:i]
	}
	if i := strings.LastIndex(url, "/"); i > 0 {
		return url[:i+1]
	}
	return url
}

// loadStoredSession — содержимое файла сессии "Имя|Значение" (универсальный
// формат: cookie приложения или именованный заголовок вроде
// X-Transmission-Session-Id).
func loadStoredSession(dir, id string) string {
	data, err := os.ReadFile(sessionPath(dir, id))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// saveNamedSession — сохранение именованной сессии приложения (0600, атомарно).
func saveNamedSession(dir, id, name, value string) {
	os.MkdirAll(filepath.Dir(sessionPath(dir, id)), 0700)
	tmp := sessionPath(dir, id) + ".tmp"
	if os.WriteFile(tmp, []byte(name+"|"+value), 0600) == nil {
		os.Rename(tmp, sessionPath(dir, id))
	}
}
