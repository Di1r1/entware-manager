// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package main

import (
	"os"

	_ "entware-manager/internal/buildinfo"
	_ "entware-manager/internal/localtime"
	"entware-manager/internal/smart"
)

// main — диспетчер по ENDPOINT (G1). Неизвестный/пустой ENDPOINT раньше молча
// уходил в HandleSmart (список дисков): опечатка в маппинге выглядела бы как
// «всё работает», а отдавала не тот ответ.
func main() {
	switch os.Getenv("ENDPOINT") {
	case "smart_config":
		smart.HandleSmartConfig()
	default:
		// "" — обычный список дисков (спецслучай smart.cgi без ENDPOINT).
		smart.HandleSmart()
	}
}
