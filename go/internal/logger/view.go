package logger

import (
	"bufio"
	"entware-manager/internal/cgiutil"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	tmpLogDir  = "/tmp/entware/logs"
	permLogDir = "/opt/var/log/entware"
)

func HandleView() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	dateFilter := cgiutil.GetQueryParam("date")
	levelFilter := cgiutil.GetQueryParam("level")
	search := cgiutil.GetQueryParam("search")

	if dateFilter == "" {
		dateFilter = time.Now().Format("2006-01-02")
	}

	logFile := findLogFile(dateFilter)

	// embed=noc — iframe внутри NOC-панели: отдельный style-noc.css +
	// ThemeBridge (data-console). Без параметра — классика: style.css +
	// Theme.init(), как было (независимость видов).
	embedNoc := cgiutil.GetQueryParam("embed") == "noc"

	fmt.Print("Content-type: text/html; charset=utf-8\n\n")

	fmt.Print(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Логи Entware Manager</title>
	` + staticBaseHTML + "\n")
	if embedNoc {
		fmt.Print(`	<link rel="stylesheet" href="logger/style-noc.css?v=1">
	<script src="theme.js?v=2"></script>
	<script src="noc/theme-bridge.js?v=2"></script>
	<script>
		if (window.ThemeBridge) ThemeBridge.init();
		else if (window.Theme) Theme.init();
	</script>`)
	} else {
		fmt.Print(`	<link rel="stylesheet" href="logger/style.css?v=3">
	<script src="theme.js?v=2"></script>
	<script>
		if (window.Theme) Theme.init();
	</script>`)
	}
	fmt.Print("\n</head>\n<body>")

	// Filter form
	fmt.Print(`<form class="log-filter" method="get">
	<div><label>Дата (YYYY-MM-DD)</label><input type="date" name="date" value="` + htmlEscape(dateFilter) + `"></div>
	<div><label>Уровень</label><select name="level">
		<option value="">Все</option>`)

	for _, l := range []string{"INFO", "WARN", "ERROR"} {
		sel := ""
		if l == levelFilter {
			sel = " selected"
		}
		fmt.Printf(`<option value="%s"%s>%s</option>`, l, sel, l)
	}

	fmt.Print(`</select></div>
	<div><label>Поиск</label><input type="text" name="search" value="` + htmlEscape(search) + `" placeholder="текст для поиска"></div>
	<div><button type="submit"><svg class="icon" width="14" height="14" style="fill: none; stroke: currentColor;"><use href="icons.svg?v=9#icon-search"/></svg> Фильтровать</button></div>
</form>
<div class="logs-container">`)

	if logFile == "" {
		fmt.Printf(`<div class="no-logs">Лог-файл не найден для даты: %s</div>`, htmlEscape(dateFilter))
	} else {
		f, err := os.Open(logFile)
		if err != nil {
			fmt.Printf(`<div class="no-logs">Ошибка открытия: %s</div>`, htmlEscape(err.Error()))
		} else {
			defer f.Close()
			var lines []string
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if levelFilter != "" {
					needle := "[" + levelFilter + "]"
					if !strings.Contains(line, needle) {
						continue
					}
				}
				if search != "" && !strings.Contains(line, search) {
					continue
				}
				lines = append(lines, line)
			}
			// Новые сверху
			for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
				lines[i], lines[j] = lines[j], lines[i]
			}
			for _, line := range lines {
				fmt.Println(`<div class="log-line">` + htmlEscape(line) + `</div>`)
			}
		}
	}

	fmt.Println(`</div></body></html>`)
}

func findLogFile(dateStr string) string {
	paths := []string{
		tmpLogDir + "/" + dateStr + ".log",
		permLogDir + "/" + dateStr + ".log",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
