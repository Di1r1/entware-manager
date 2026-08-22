// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
// entware-telegram — настройка Telegram-уведомлений.
// Эндпоинты: telegram_config, telegram_test. Режим бота: -bot.
package main

import (
	_ "entware-manager/internal/buildinfo"
	_ "entware-manager/internal/localtime"
	"entware-manager/internal/telegram"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-bot" {
		if err := telegram.BotRun(); err != nil {
			fmt.Fprintln(os.Stderr, "[bot]", err)
			os.Exit(1)
		}
		return
	}
	telegram.Handle()
}
