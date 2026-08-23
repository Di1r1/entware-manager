// Package telegram — отправка сообщений через Telegram Bot API.
package telegram

import (
	"fmt"
	"io"
	"mime/multipart"
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
	form := url.Values{}
	form.Set("chat_id", cfg.ChatID)
	form.Set("text", text)
	form.Set("disable_web_page_preview", "true")

	code, _, err := postTelegram(cfg, "sendMessage", form)
	if err != nil {
		logErr("sendMessage: %s", err.Error())
		return false
	}
	if code != http.StatusOK {
		// Не логируем тело — там может быть часть токена/чувствительные данные.
		logErr("sendMessage HTTP %d (bot token redacted)", code)
		return false
	}
	return true
}

// SendMessageMarkup отправляет текст с произвольной reply_markup (JSON-строка),
// например inline-клавиатурой для подтверждения действий бота.
func SendMessageMarkup(cfg Config, text, replyMarkupJSON string) bool {
	if !cfg.Configured || cfg.ChatID == "" || text == "" {
		return false
	}
	form := url.Values{}
	form.Set("chat_id", cfg.ChatID)
	form.Set("text", text)
	form.Set("disable_web_page_preview", "true")
	if replyMarkupJSON != "" {
		form.Set("reply_markup", replyMarkupJSON)
	}
	code, _, err := postTelegram(cfg, "sendMessage", form)
	if err != nil {
		logErr("sendMessageMarkup: %s", err.Error())
		return false
	}
	return code == http.StatusOK
}

// AnswerCallbackQuery закрывает «часики» на нажатой inline-кнопке.
func AnswerCallbackQuery(cfg Config, cbID, text string) bool {
	form := url.Values{}
	form.Set("callback_query_id", cbID)
	if text != "" {
		form.Set("text", text)
	}
	code, _, err := postTelegram(cfg, "answerCallbackQuery", form)
	if err != nil {
		logErr("answerCallbackQuery: %s", err.Error())
		return false
	}
	return code == http.StatusOK
}

// postTelegram — POST формы на метод Bot API. Возвращает код и тело.
// Ошибки сети уже маскированы (токен вырезан).
func postTelegram(cfg Config, method string, form url.Values) (int, []byte, error) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/%s", cfg.BotToken, method)
	client := httpClient(cfg)
	resp, err := client.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return 0, nil, fmt.Errorf("%s", redactURL(err.Error(), cfg.BotToken))
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, body, nil
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

// SendDocumentBytes отправляет файл документом в chat_id (лимит 50 МБ).
func SendDocumentBytes(cfg Config, filename string, data []byte, caption string) bool {
	if !cfg.Configured || cfg.ChatID == "" || len(data) == 0 {
		return false
	}
	if len(data) > 50*1024*1024 {
		logErr("sendDocument: файл %s слишком большой (%d байт)", filename, len(data))
		return false
	}
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		defer pw.Close()
		mw.WriteField("chat_id", cfg.ChatID)
		if caption != "" {
			mw.WriteField("caption", caption)
		}
		part, err := mw.CreateFormFile("document", filename)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		part.Write(data)
		mw.Close()
	}()

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", cfg.BotToken)
	client := httpClient(cfg)
	resp, err := client.Post(apiURL, mw.FormDataContentType(), pr)
	if err != nil {
		logErr("sendDocument: %s", redactURL(err.Error(), cfg.BotToken))
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))
	if resp.StatusCode != http.StatusOK {
		logErr("sendDocument HTTP %d", resp.StatusCode)
		return false
	}
	return true
}
