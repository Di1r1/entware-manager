// Di1r1
//
// Файл-сессия панели: единый токен на роутер в /opt/var/run/panel_session.
//
// CGI-архитектура не имеет общего демона-состояния (каждый запрос — отдельный
// процесс в обоих режимах веб-сервера), поэтому сессия хранится файлом на
// диске: токен из /dev/urandom, TTL по mtime файла, constant-time сравнение.
// Защищает все GET-эндпоинты панели (страницы без логина больше не открыты).
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"os"
	"strings"
	"time"
)

// SessionFile — файл сессии (переменная для тестов).
var SessionFile = "/opt/var/run/panel_session"

// SessionTTL — время жизни сессии.
const SessionTTL = 24 * time.Hour

// SessionCookieName — имя cookie.
const SessionCookieName = "panel_session"

// CreateSession генерирует токен и пишет файл атомарно (temp + mv), 0600 root.
// Возвращает токен (для Set-Cookie).
func CreateSession() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	tmp := SessionFile + ".tmp"
	if err := os.WriteFile(tmp, []byte(token+"\n"), 0600); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, SessionFile); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return token, nil
}

// DestroySession удаляет файл сессии.
func DestroySession() {
	_ = os.Remove(SessionFile)
}

// SessionTokenFromCookie извлекает токен из HTTP_COOKIE (env CGI).
func SessionTokenFromCookie() string {
	raw := os.Getenv("HTTP_COOKIE")
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, SessionCookieName+"=") {
			return strings.TrimPrefix(part, SessionCookieName+"=")
		}
	}
	return ""
}

// SessionValid проверяет, что cookie соответствует файлу сессии и не истёк.
func SessionValid() bool {
	return SessionValidCookie(SessionTokenFromCookie())
}

// SessionValidCookie проверяет переданный токен против файла сессии.
func SessionValidCookie(token string) bool {
	data, err := os.ReadFile(SessionFile)
	if err != nil {
		return false
	}
	stored := strings.TrimSpace(string(data))
	if token == "" || len(token) != len(stored) {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(stored)) != 1 {
		return false
	}
	fi, err := os.Stat(SessionFile)
	if err != nil {
		return false
	}
	if time.Since(fi.ModTime()) > SessionTTL {
		DestroySession()
		return false
	}
	return true
}

// TokenFromHeader извлекает panel_session из заголовка Cookie (для go-режима,
// где cookie приходит в HTTP-запросе, а не в CGI-окружении).
func TokenFromHeader(cookieHeader string) string {
	for _, part := range strings.Split(cookieHeader, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, SessionCookieName+"=") {
			return strings.TrimPrefix(part, SessionCookieName+"=")
		}
	}
	return ""
}
