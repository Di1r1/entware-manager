// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
// entware-telegram — настройка Telegram-уведомлений.
// Эндпоинты: telegram_config, telegram_test.
package main

import (
	_ "entware-manager/internal/localtime"
	"entware-manager/internal/telegram"
)

func main() {
	telegram.Handle()
}
