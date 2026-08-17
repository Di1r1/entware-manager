package logger

import (
	"bufio"
	"encoding/json"
	"entware-manager/internal/cgiutil"
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
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	sourceName := cgiutil.GetQueryParam("source")
	filePath := cgiutil.GetQueryParam("file")
	search := cgiutil.GetQueryParam("search")

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
	<link rel="stylesheet" href="/entware-manager/logger/style.css?v=2">
	<script src="/entware-manager/theme.js?v=2"></script>
	<script>
		if (window.Theme) Theme.init();
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

	// Новые сверху
	for i, j := start, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
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
