// Di1r1
package main

import (
	"fmt"
	"os"

	"entware-manager/internal/backup"
	"entware-manager/internal/stats"
)

func main() {
	ep := os.Getenv("ENDPOINT")
	if ep == "" {
		fmt.Print("Content-type: text/html; charset=utf-8\n\n")
		fmt.Println("<p class='error'>ENDPOINT not set</p>")
		return
	}
	switch ep {
	case "stats":
		stats.Handle()
	case "version":
		stats.HandleVersion()
	case "help":
		stats.HandleHelp()
	case "links_load":
		stats.HandleLinksLoad()
	case "tmpfs":
		stats.HandleTmpfs()
	case "view_file":
		stats.HandleViewFile()
	case "delete_file":
		stats.HandleDeleteFile()
	case "links_save":
		stats.HandleLinksSave()
	case "auth_config":
		stats.HandleAuthConfig()
	case "crontab":
		stats.HandleCrontab()
	case "crontab_update":
		stats.HandleCrontabUpdate()
	case "backup":
		backup.HandleCreate()
	case "backup_restore":
		backup.HandleRestore()
	case "update_check":
		stats.HandleUpdateCheck()
	case "update_run":
		stats.HandleUpdateRun()
	case "update_status":
		stats.HandleUpdateStatus()
	case "update_worker":
		stats.HandleUpdateWorker()
	case "prepare_offline":
		stats.HandleOfflinePrepare()
	default:
		fmt.Print("Content-type: text/html; charset=utf-8\n\n")
		fmt.Printf("<p class='error'>unknown endpoint: %s</p>", ep)
	}
}
