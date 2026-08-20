// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
// entware-rdp — управление RDP-модулем (grdp-proxy).
// Эндпоинты: rdp_status, rdp_config, rdp_start, rdp_stop.
package main

import (
	_ "entware-manager/internal/localtime"
	"entware-manager/internal/rdp"
)

func main() {
	rdp.Handle()
}
