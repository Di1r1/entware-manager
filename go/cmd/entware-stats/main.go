package main

import (
	"fmt"
	"os"

	"entware-manager/internal/stats"
)

func main() {
	ep := os.Getenv("ENDPOINT")
	if ep == "" {
		fmt.Println("Content-type: text/html; charset=utf-8\n")
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
	default:
		fmt.Println("Content-type: text/html; charset=utf-8\n")
		fmt.Printf("<p class='error'>unknown endpoint: %s</p>", ep)
	}
}
