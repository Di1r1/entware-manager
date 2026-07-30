package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const systemSourcesFile = "/opt/web_entware/logger/system_sources.json"

type Source struct {
	Name string `json:"name"`
	File string `json:"file"`
}

type Sources struct {
	Sources []Source `json:"sources"`
}

func HandleSystemLogs() {
	if !IsGET() {
		NotAllowed()
		return
	}

	sourceName := getQueryParam("source")
	filePath := getQueryParam("file")
	search := getQueryParam("search")

	logFile := filePath
	if logFile == "" && sourceName != "" {
		logFile = findSourceFile(sourceName)
	}

	fmt.Print("Content-type: text/html; charset=utf-8\n\n")

	fmt.Print(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Системные логи</title>
	<link rel="stylesheet" href="/entware-manager/logger/style.css">
	<style>
		.logs-container { padding: 1rem; }
		.log-line {
			font-family: monospace; font-size: 13px; padding: 4px 8px;
			border-bottom: 1px solid var(--border-color);
			white-space: pre-wrap; word-break: break-all;
			background: var(--input-bg); margin: 2px 0; border-radius: 6px;
			color: var(--text-primary);
		}
		.no-logs { text-align: center; padding: 2rem; color: var(--text-muted); }
	</style>
	<script>
		(function() {
			try {
				const isNight = localStorage.getItem('entware_theme') === 'night';
				if (isNight) document.documentElement.classList.add('night');
				else document.documentElement.classList.remove('night');
			} catch(e) {}
		})();
		window.addEventListener('storage', function(e) {
			if (e.key === 'entware_theme') {
				if (e.newValue === 'night') document.documentElement.classList.add('night');
				else document.documentElement.classList.remove('night');
			}
		});
	</script>
</head>
<body>
<div class="logs-container">`)

	if logFile == "" {
		fmt.Println(`<div class="no-logs">Лог-файл не найден</div>`)
		fmt.Println(`</div></body></html>`)
		return
	}

	f, err := os.Open(logFile)
	if err != nil {
		fmt.Printf(`<div class="no-logs">Лог-файл не найден: %s</div>`, htmlEscape(logFile))
		fmt.Println(`</div></body></html>`)
		return
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if search != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(search)) {
			continue
		}
		lines = append(lines, line)
		count++
	}

	start := 0
	if len(lines) > 500 {
		start = len(lines) - 500
	}

	for _, line := range lines[start:] {
		fmt.Println(`<div class="log-line">` + htmlEscape(line) + `</div>`)
	}

	fmt.Println(`</div></body></html>`)
}

func findSourceFile(name string) string {
	data, err := os.ReadFile(systemSourcesFile)
	if err != nil {
		return ""
	}
	var sources Sources
	if err := json.Unmarshal(data, &sources); err != nil {
		return ""
	}
	for _, s := range sources.Sources {
		if s.Name == name {
			return s.File
		}
	}
	return ""
}
