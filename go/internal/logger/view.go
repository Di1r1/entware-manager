package logger

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	tmpLogDir = "/tmp/entware/logs"
	permLogDir = "/opt/var/log/entware"
)

func HandleView() {
	if !IsGET() {
		NotAllowed()
		return
	}

	dateFilter := getQueryParam("date")
	levelFilter := getQueryParam("level")
	search := getQueryParam("search")

	if dateFilter == "" {
		dateFilter = time.Now().Format("2006-01-02")
	}

	logFile := findLogFile(dateFilter)

	fmt.Println("Content-type: text/html; charset=utf-8\n")

	fmt.Print(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Логи Entware Manager</title>
	<link rel="stylesheet" href="/entware-manager/logger/style.css">
	<style>
		html.night { background: #1a202c; }
		html:not(.night) { background: #f9fafb; }
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
<body>`)

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
	<div><button type="submit"><svg class="icon" width="14" height="14" style="fill: none; stroke: currentColor;"><use href="/entware-manager/icons.svg?v=2#icon-search"/></svg> Фильтровать</button></div>
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
