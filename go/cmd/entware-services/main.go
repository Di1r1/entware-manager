// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package main

import (
	_ "entware-manager/internal/buildinfo"
	"entware-manager/internal/cgiutil"
	"os"

	_ "entware-manager/internal/localtime"
	"entware-manager/internal/services"
)

func main() {
	ep := os.Getenv("ENDPOINT")
	if ep == "" {
		cgiutil.WriteError("ENDPOINT not set")
		return
	}

	switch ep {
	case "services":
		services.HandleServices()
	case "service_action":
		services.HandleServiceAction()
	case "ttyd_control":
		services.HandleTTYDControl()
	case "service_watchdog_status":
		services.HandleWatchdogStatus()
	case "service_watchdog_action":
		services.HandleWatchdogAction()
	case "service_watchdog_config":
		services.HandleWatchdogConfig()
	case "service_watchdog_events":
		services.HandleWatchdogEvents()
	case "check_syntax":
		services.HandleCheckSyntax()
	case "check_deps":
		services.HandleCheckDeps()
	default:
		cgiutil.WriteError("unknown endpoint: " + ep)
	}
}
