package logger

import (
	"fmt"
	"os"
	"strings"
)

const systemLog = "/opt/var/log/entware/system.log"

func HandleSystemLog() {
	if !IsGET() {
		NotAllowed()
		return
	}

	fmt.Print("Content-type: text/html; charset=utf-8\n\n")

	fmt.Print(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { margin: 0; padding: 16px; background: transparent; color: #e0e0e0; font-family: monospace; font-size: 13px; }
		pre { margin: 0; white-space: pre-wrap; word-break: break-all; }
		.empty { color: #718096; text-align: center; padding: 20px; }
	</style>
</head>
<body>`)

	data, err := os.ReadFile(systemLog)
	if err != nil || len(data) == 0 {
		fmt.Println(`<div class="empty">Системный лог пуст</div>`)
	} else {
		fmt.Print("<pre>")
		fmt.Print(htmlEscape(strings.TrimSpace(string(data))))
		fmt.Println("</pre>")
	}

	fmt.Println(`</body></html>`)
}
