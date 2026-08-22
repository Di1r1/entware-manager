// Package telegram — HTTP-обработчики настроек Telegram-шлюза.
package telegram

import (
	"encoding/json"
	"entware-manager/internal/auth"
	"entware-manager/internal/cgiutil"
	"os"
	"strings"
)

// Handle диспетчеризует подэндпоинты модуля Telegram.
func Handle() {
	ep := os.Getenv("ENDPOINT")
	switch ep {
	case "telegram_config":
		HandleConfig()
	case "telegram_test":
		HandleTest()
	default:
		cgiutil.WriteStatusError("unknown endpoint: " + ep)
	}
}

// HandleConfig: GET — конфиг без токена; POST — сохранение (Origin-чек).
func HandleConfig() {
	if cgiutil.IsPOST() {
		handleConfigPost()
		return
	}
	cfg := LoadConfig()
	cgiutil.WriteJSON(map[string]interface{}{
		"status":      "ok",
		"enabled":     cfg.Enabled,
		"configured":  cfg.Configured,
		"level":       cfg.Level,
		"sources":     cfg.Sources,
		"bot_enabled": cfg.BotEnabled,
		"autostart":   cfg.Autostart,
		"proxy_url":   cfg.ProxyURL,
		"chat_id":     cfg.ChatID,
		"thresholds":  cfg.Thresholds,
	})
}

func handleConfigPost() {
	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteStatusError(auth.CrossSiteDeny)
		return
	}
	body := cgiutil.ReadPOSTBody()
	params := cgiutil.ParseFormBody(body)
	if body == "" && len(params) == 0 {
		cgiutil.WriteStatusError("Empty request")
		return
	}

	cfg := LoadConfig()

	if v, ok := params["enabled"]; ok {
		cfg.Enabled = v == "true"
	}
	if v, ok := params["bot_enabled"]; ok {
		cfg.BotEnabled = v == "true"
	}
	if v, ok := params["autostart"]; ok {
		cfg.Autostart = v == "true"
	}
	if v, ok := params["level"]; ok && v != "" {
		if !IsValidLevel(v) {
			cgiutil.WriteStatusError("Некорректный уровень: " + v)
			return
		}
		cfg.Level = v
	}
	if v, ok := params["sources"]; ok {
		if v == "" {
			cfg.Sources = []string{"system", "monitor", "packages"}
		} else {
			cfg.Sources = splitComma(v)
		}
	}
	// chat_id: пустое значение сохраняет прежний (GET не отдавал его ранее,
	// и пересохранение настроек стирало chat_id — фикс потери).
	if v, ok := params["chat_id"]; ok && v != "" {
		if !IsValidChatID(v) {
			cgiutil.WriteStatusError("Некорректный chat_id (только цифры)")
			return
		}
		cfg.ChatID = v
	}
	// bot_token: пустое значение сохраняет прежний токен (поле скрыто на фронте).
	if v, ok := params["bot_token"]; ok && v != "" {
		cfg.BotToken = v
	}
	// proxy_url: http:// или socks5:// (пустое — прямое соединение).
	if v, ok := params["proxy_url"]; ok {
		if !IsValidProxyURL(v) {
			cgiutil.WriteStatusError("Некорректный прокси (допустимо http:// или socks5://)")
			return
		}
		cfg.ProxyURL = v
	}
	// thresholds: JSON-блок критических порогов ({"cpu_temp":{"enabled":true,"value":90},...}).
	if v, ok := params["thresholds"]; ok && v != "" {
		var th Thresholds
		if err := json.Unmarshal([]byte(v), &th); err != nil {
			cgiutil.WriteStatusError("Некорректный JSON порогов: " + err.Error())
			return
		}
		if !ValidateThresholds(th) {
			cgiutil.WriteStatusError("Значения порогов вне допустимых пределов")
			return
		}
		cfg.Thresholds = th
	}
	cfg.Configured = cfg.BotToken != ""

	if err := SaveConfig(cfg); err != nil {
		cgiutil.WriteStatusError("Не удалось сохранить настройки: " + err.Error())
		return
	}
	cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "message": "Настройки сохранены"})
}

// HandleTest: POST — отправка тестового сообщения (Origin-чек).
func HandleTest() {
	if !cgiutil.IsPOST() {
		cgiutil.WriteStatusError("Метод не поддерживается (только POST)")
		return
	}
	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteStatusError(auth.CrossSiteDeny)
		return
	}
	cfg := LoadConfig()
	if !cfg.Configured || cfg.ChatID == "" {
		cgiutil.WriteStatusError("Telegram не настроен: укажите токен и chat_id")
		return
	}
	if !SendMessage(cfg, "Тестовое сообщение от Entware Manager") {
		cgiutil.WriteStatusError("Не удалось отправить сообщение. Проверьте токен/chat_id")
		return
	}
	cgiutil.WriteJSON(map[string]interface{}{"status": "ok", "message": "Тестовое сообщение отправлено"})
}

// splitComma разбивает "a,b,c" на список без пустых элементов.
func splitComma(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
