package monitor

import (
	"bufio"
	"entware-manager/internal/cgiutil"
	"fmt"
	"os"
	"strings"
	"time"
)

func HandleLog() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	// Единый дневной суточный лог: и факты демона ([monitor]), и действия
	// кнопок панели ([monitor]) пишутся сюда — вкладка «Защита» показывает всё.
	logFile := fmt.Sprintf("/tmp/entware/logs/%s.log", time.Now().Format("2006-01-02"))

	var lines []string
	f, err := os.Open(logFile)
	if err != nil {
		WriteText("Лог-файл не найден\n")
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		lower := strings.ToLower(line)
		if strings.Contains(lower, "[monitor]") {
			lines = append(lines, line)
		}
	}

	// Новые сверху
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	// Last 200 lines
	start := 0
	if len(lines) > 200 {
		start = len(lines) - 200
	}
	lines = lines[start:]

	// ?format=json для NOC (п. 2.4 запроса): [{timestamp, level, text}].
	// Без параметра — plain-text как раньше.
	if GetParam("format") == "json" {
		cgiutil.WriteJSON(collectLogJSON(lines))
		return
	}

	WriteText(strings.Join(lines, "\n") + "\n")
}

// logJSONEntry — одна строка лога для format=json.
type logJSONEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Text      string `json:"text"`
}

func collectLogJSON(lines []string) []logJSONEntry {
	entries := make([]logJSONEntry, 0, len(lines))
	for _, line := range lines {
		t := strings.TrimSpace(line)
		ts := ""
		if idx1 := strings.Index(t, "["); idx1 >= 0 {
			if idx2 := strings.Index(t[idx1:], "]"); idx2 >= 0 {
				ts = strings.TrimSpace(t[idx1+1 : idx1+idx2])
			}
		}
		level := "INFO"
		lower := strings.ToLower(t)
		for _, l := range []string{"error", "warn"} {
			if strings.Contains(lower, "["+l+"]") {
				level = strings.ToUpper(l)
				break
			}
		}
		entries = append(entries, logJSONEntry{Timestamp: ts, Level: level, Text: t})
	}
	return entries
}
