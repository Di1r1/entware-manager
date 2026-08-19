package services

import (
	"entware-manager/internal/cgiutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type WatchdogEvent struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Service   string `json:"service"`
	Event     string `json:"event"`
	Details   string `json:"details"`
}

func HandleWatchdogEvents() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	limit := 20
	if l := cgiutil.GetQueryParam("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	events := parseWatchdogLog(limit)
	if events == nil {
		events = []WatchdogEvent{}
	}
	cgiutil.WriteJSON(map[string][]WatchdogEvent{"events": events})
}

func parseWatchdogLog(limit int) []WatchdogEvent {
	// Читаем дневной лог (факты демона service_watchdog: [service]) +
	// профильный лог действий панели (service_actions.log: [service_action]).
	// Объединяем и сортируем по времени — полная картина по службам.
	var matched []string
	matched = append(matched, readServiceLines(dailyLogFile(), "[service]", limit)...)
	matched = append(matched, readServiceLines(serviceActionsLog, "[service_action]", limit)...)

	// Сортировка по timestamp (первые 20 символов "[YYYY-MM-DD HH:MM:SS]"),
	// новые сверху. Сортировка вставками (без внешних зависимостей).
	for i := 1; i < len(matched); i++ {
		key := matched[i]
		keyTs := tsPrefix(key)
		j := i - 1
		for j >= 0 && tsPrefix(matched[j]) < keyTs {
			matched[j+1] = matched[j]
			j--
		}
		matched[j+1] = key
	}
	if len(matched) > limit {
		matched = matched[:limit]
	}

	events := make([]WatchdogEvent, 0, len(matched))
	for _, line := range matched {
		events = append(events, parseWatchdogLine(line))
	}
	return events
}

// dailyLogFile возвращает путь к дневному суточному логу.
func dailyLogFile() string {
	return filepath.Join("/tmp/entware/logs", time.Now().Format("2006-01-02")+".log")
}

// tsPrefix извлекает префикс "[YYYY-MM-DD HH:MM:SS]" для сортировки.
func tsPrefix(line string) string {
	if len(line) < 20 {
		return line
	}
	return line[:20]
}

// readServiceLines читает файл и возвращает последние строки (до limit),
// содержащие tagPattern, в порядке файла (снизу вверх по файлу не важно —
// сортировка по времени будет ниже).
func readServiceLines(file, tagPattern string, limit int) []string {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	var matched []string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.Contains(strings.ToLower(line), strings.ToLower(tagPattern)) {
			matched = append(matched, line)
			if len(matched) >= limit {
				break
			}
		}
	}
	return matched
}

func parseWatchdogLine(line string) WatchdogEvent {
	ts := ""
	if idx1 := strings.Index(line, "["); idx1 >= 0 {
		if idx2 := strings.Index(line[idx1:], "]"); idx2 >= 0 {
			ts = strings.TrimSpace(line[idx1+1 : idx1+idx2])
		}
	}

	// Тег может быть [service] (факт демона) или [service_action] (действие панели).
	var tagBracket string
	lower := strings.ToLower(line)
	if strings.Contains(lower, "[service_action]") {
		tagBracket = "[service_action]"
	} else if strings.Contains(lower, "[service]") {
		tagBracket = "[service]"
	}
	if tagBracket == "" {
		return WatchdogEvent{Timestamp: ts, Level: "INFO"}
	}

	tagIdx := strings.Index(lower, tagBracket)
	prefix := line[:tagIdx]
	rest := strings.TrimSpace(line[tagIdx+len(tagBracket):])

	level := "INFO"
	prefixLower := strings.ToLower(prefix)
	for _, l := range []string{"error", "warn", "info"} {
		if strings.Contains(prefixLower, "["+l+"]") {
			level = strings.ToUpper(l)
			break
		}
	}

	fields := strings.Fields(rest)
	var service, event, details string
	if len(fields) > 0 {
		service = strings.TrimRight(fields[0], ":")
	}
	if len(fields) > 1 {
		event = fields[1]
	}
	if len(fields) > 2 {
		details = strings.Join(fields[2:], " ")
		details = strings.ReplaceAll(details, "(", "")
		details = strings.ReplaceAll(details, ")", "")
	}

	return WatchdogEvent{
		Timestamp: ts,
		Level:     level,
		Service:   service,
		Event:     event,
		Details:   details,
	}
}
