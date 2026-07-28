// Di1r1
package main

import (
	"os"

	"entware-manager/internal/network"
)

func main() {
	ep := os.Getenv("ENDPOINT")
	if ep == "" {
		network.WriteError("ENDPOINT not set")
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
		network.WriteError("unknown endpoint: " + ep)
	}
}
