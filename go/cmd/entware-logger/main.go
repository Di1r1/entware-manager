package main

import (
	"os"

	"entware-manager/internal/logger"
)

func main() {
	ep := os.Getenv("ENDPOINT")
	if ep == "" {
		logger.WriteError("ENDPOINT not set")
		return
	}

	switch ep {
	case "logger_config":
		logger.HandleConfig()
	case "logger_view":
		logger.HandleView()
	case "logger_system_logs":
		logger.HandleSystemLogs()
	case "logger_system_log":
		logger.HandleSystemLog()
	case "logger_find_by_name":
		logger.HandleFind()
	case "logger_rotate":
		logger.HandleRotate()
	case "logger_clear":
		logger.HandleClear()
	default:
		logger.WriteError("unknown endpoint: " + ep)
	}
}
