package logger

import (
	"entware-manager/internal/cgiutil"
	"fmt"
	"os"
	"strings"
)

const systemLog = "/opt/var/log/entware/system.log"

func HandleSystemLog() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	fmt.Print("Content-type: text/html; charset=utf-8\n\n")

	fmt.Print(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		.syslog-wrap { padding: 16px; }
		.syslog { margin: 0; white-space: pre-wrap; word-break: break-all; font-family: monospace; font-size: 13px; }
		.syslog-empty { text-align: center; padding: 20px; opacity: .65; }
	</style>
</head>
<body><div class="syslog-wrap">`)

	data, err := os.ReadFile(systemLog)
	if err != nil || len(data) == 0 {
		fmt.Println(`<div class="syslog-empty">Системный лог пуст</div>`)
	} else {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		// Новые сверху
		for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
			lines[i], lines[j] = lines[j], lines[i]
		}
		fmt.Print(`<pre class="syslog">`)
		fmt.Print(htmlEscape(strings.Join(lines, "\n")))
		fmt.Println(`</pre>`)
	}

	fmt.Println(`</div></body></html>`)
}
