package main

import (
	"os"

	"entware-manager/internal/services"
)

func main() {
	ep := os.Getenv("ENDPOINT")
	if ep == "" {
		services.WriteError("ENDPOINT not set")
		return
	}

	switch ep {
	case "services":
		services.HandleServices()
	case "service_action":
		services.HandleServiceAction()
	case "ttyd_control":
		services.HandleTTYDControl()
	default:
		services.WriteError("unknown endpoint: " + ep)
	}
}
