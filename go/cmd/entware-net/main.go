// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package main

import (
	_ "entware-manager/internal/buildinfo"
	"entware-manager/internal/cgiutil"
	"os"

	_ "entware-manager/internal/localtime"
	"entware-manager/internal/network"
)

func main() {
	ep := os.Getenv("ENDPOINT")
	if ep == "" {
		cgiutil.WriteError("ENDPOINT not set")
		return
	}

	switch ep {
	case "network_interfaces":
		network.HandleInterfaces()
	case "network_routes":
		network.HandleRoutes()
	case "network_arp":
		network.HandleARP()
	case "network_status":
		network.HandleStatus()
	case "network_stats":
		network.HandleNetworkStats()
	case "network_events":
		network.HandleEvents()
	case "network_config":
		network.HandleConfig()
	case "network_action":
		network.HandleAction()
	default:
		cgiutil.WriteError("unknown endpoint: " + ep)
	}
}
