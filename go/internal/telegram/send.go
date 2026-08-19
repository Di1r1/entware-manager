// Package telegram — отправка сообщений через Telegram Bot API.
package telegram

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// sendTimeout — таймаут HTTP-запроса к Telegram (не вешать роутер).
const sendTimeout = 10 * time.Second

// httpClient возвращает HTTP-клиент, маршрутизирующий трафик через прокси
// (если задан), иначе — прямое соединение. HTTP-прокси поддерживается
// нативно net/http; socks5:// — через ProxyURL с socks5-схемой (Go понимает
// только http/https, поэтому для socks5 используем прокси как http-конверт —
// большинство локальных прокси (xray) принимают оба).
func httpClient(cfg Config) *http.Client {
	transport := &http.Transport{}
	if cfg.ProxyURL != "" {
		if pu, err := url.Parse(cfg.ProxyURL); err == nil {
			transport.Proxy = http.ProxyURL(pu)
		}
	}
	return &http.Client{Timeout: sendTimeout, Transport: transport}
}

// SendMessage отправляет текст в chat_id через Bot API.
// Возвращает false при ошибке. Токен не логируется.
func SendMessage(cfg Config, text string) bool {
	if !cfg.Configured || cfg.ChatID == "" || text == "" {
		return false
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	form := url.Values{}
	form.Set("chat_id", cfg.ChatID)
	form.Set("text", text)
	form.Set("disable_web_page_preview", "true")

	client := httpClient(cfg)
	resp, err := client.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		// *url.Error включает полный URL с токеном — маскируем перед логированием.
		logErr("sendMessage network error: %s", redactURL(err.Error(), cfg.BotToken))
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode != http.StatusOK {
		// Не логируем тело — там может быть часть токена/чувствительные данные.
		logErr("sendMessage HTTP %d (bot token redacted)", resp.StatusCode)
		return false
	}
	_ = body
	return true
}

// logErr пишет в stderr с временной меткой (демон перенаправляет в лог).
func logErr(format string, args ...interface{}) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(os.Stderr, "[%s] [ERROR] "+format+"\n", append([]interface{}{ts}, args...)...)
}

// redactURL заменяет вхождение токена в строке (URL из ошибки) на маску.
func redactURL(s, token string) string {
	if token == "" {
		return s
	}
	return strings.ReplaceAll(s, token, RedactToken(token))
}
