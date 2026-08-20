// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package main

import (
	"entware-manager/internal/cgiutil"
	"os"

	_ "entware-manager/internal/localtime"
	"entware-manager/internal/logger"
)

func main() {
	ep := os.Getenv("ENDPOINT")
	if ep == "" {
		cgiutil.WriteError("ENDPOINT not set")
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
	case "logger_debug":
		logger.HandleLoggerDebug()
	case "logger_debug_path":
		logger.HandleLoggerDebugPath()
	default:
		cgiutil.WriteError("unknown endpoint: " + ep)
	}
}
