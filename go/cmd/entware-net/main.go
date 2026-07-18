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
	default:
		network.WriteError("unknown endpoint: " + ep)
	}
}
