package network

import (
	"entware-manager/internal/cgiutil"
	"entware-manager/internal/packages"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type LogEvent struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Service   string `json:"service"`
	Event     string `json:"event"`
	Details   string `json:"details"`
}

const maxFallbackDays = 30

func HandleEvents() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	limit := 20
	if l := cgiutil.GetParam("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	tag := cgiutil.GetParam("tag")
	if tag == "" {
		tag = "network"
	}

	events := CollectEvents(tag, limit)
	cgiutil.WriteJSON(map[string][]LogEvent{"events": events})
}

// CollectEvents — чистое значение для dashboard.cgi и network_events.cgi.
// tag=packages читает package_changes.log (п. 3.1/2.2 запроса NOC, вариант Б).
func CollectEvents(tag string, limit int) []LogEvent {
	if tag == "packages" {
		return collectPackageEvents(limit)
	}
	events := parseLogFile(tag, limit)
	if events == nil {
		events = []LogEvent{}
	}
	return events
}

// collectPackageEvents маппит записи журнала пакетов в общий формат событий.
// Level: success → INFO, прочее → WARN; Service всегда "packages".
func collectPackageEvents(limit int) []LogEvent {
	entries := packages.CollectPackageLog(limit)
	events := make([]LogEvent, 0, len(entries))
	for _, e := range entries {
		level := "INFO"
		if e.Status != "" && e.Status != "success" {
			level = "WARN"
		}
		events = append(events, LogEvent{
			Timestamp: e.Timestamp,
			Level:     level,
			Service:   "packages",
			Event:     e.Action,
			Details:   e.Package,
		})
	}
	return events
}

func parseLogFile(tag string, limit int) []LogEvent {
	// Пробуем текущий день
	logPath := filepath.Join(LogDir, time.Now().Format("2006-01-02")+".log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		// Фолбэк: ищем за последние maxFallbackDays
		for d := 1; d <= maxFallbackDays; d++ {
			date := time.Now().AddDate(0, 0, -d).Format("2006-01-02")
			logPath = filepath.Join(LogDir, date+".log")
			data, err = os.ReadFile(logPath)
			if err == nil {
				break
			}
		}
		if err != nil {
			return nil
		}
	}

	lines := strings.Split(string(data), "\n")
	normTag := normalizeServiceTag(tag)
	matchAll := normTag == "all"

	var matched []string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if matchAll || extractServiceTag(line) == normTag {
			matched = append(matched, line)
			if len(matched) >= limit {
				break
			}
		}
	}

	events := make([]LogEvent, 0, len(matched))
	for _, line := range matched {
		events = append(events, parseLogLine(line, tag))
	}
	return events
}

func parseLogLine(line, tag string) LogEvent {
	ts := ""
	if idx1 := strings.Index(line, "["); idx1 >= 0 {
		if idx2 := strings.Index(line[idx1:], "]"); idx2 >= 0 {
			ts = strings.TrimSpace(line[idx1+1 : idx1+idx2])
		}
	}

	if tag == "all" {
		return parseGenericLine(line, ts)
	}

	normTag := normalizeServiceTag(tag)
	if extractServiceTag(line) != normTag {
		return LogEvent{Timestamp: ts, Level: "INFO"}
	}

	// Отрезаем сообщение после тега [tag] (регистронезависимо, с учётом
	// суффикса .cgi: [login.cgi], [crontab_update.cgi]).
	lower := strings.ToLower(line)
	tagIdx := -1
	tagLen := 0
	for _, cand := range []string{"[" + strings.ToLower(tag) + "]", "[" + normTag + ".cgi]"} {
		if idx := strings.Index(lower, cand); idx >= 0 {
			tagIdx, tagLen = idx, len(cand)
			break
		}
	}
	if tagIdx < 0 {
		return LogEvent{Timestamp: ts, Level: "INFO"}
	}

	prefix := line[:tagIdx]
	rest := strings.TrimSpace(line[tagIdx+tagLen:])

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

	return LogEvent{
		Timestamp: ts,
		Level:     level,
		Service:   service,
		Event:     event,
		Details:   details,
	}
}

func parseGenericLine(line, ts string) LogEvent {
	level := "INFO"
	lower := strings.ToLower(line)
	for _, l := range []string{"error", "warn", "info"} {
		if strings.Contains(lower, "["+l+"]") {
			level = strings.ToUpper(l)
			break
		}
	}
	var b strings.Builder
	depth := 0
	for _, r := range line {
		switch {
		case r == '[':
			depth++
		case r == ']' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	text := strings.TrimSpace(b.String())
	text = strings.ReplaceAll(text, "(", "")
	text = strings.ReplaceAll(text, ")", "")
	return LogEvent{Timestamp: ts, Level: level, Service: extractServiceTag(line), Event: text}
}

// normalizeServiceTag приводит тег к каноническому виду для фильтров
// карточки «События»: нижний регистр + срез суффикса .cgi
// ([login.cgi] → login, [crontab_update.cgi] → crontab_update).
func normalizeServiceTag(tag string) string {
	t := strings.ToLower(strings.TrimSpace(tag))
	t = strings.TrimPrefix(t, "[")
	t = strings.TrimSuffix(t, "]")
	t = strings.TrimSuffix(t, ".cgi")
	return t
}

// extractServiceTag достаёт нормализованный тег источника из строки
// дневного лога. Формат: [ts] [LEVEL] [ip] [pid] [TAG] сообщение,
// TAG — последняя [...] группа перед сообщением.
func extractServiceTag(line string) string {
	var groups []string
	depth := 0
	start := -1
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '[':
			if depth == 0 {
				start = i
			}
			depth++
		case ']':
			if depth > 0 {
				depth--
				if depth == 0 && start >= 0 {
					groups = append(groups, line[start+1:i])
					start = -1
				}
			}
		}
	}
	// Пропускаем служебные группы: ts (дата+время), LEVEL, ip, pid.
	// Тег источника — последняя оставшаяся группа.
	for i := len(groups) - 1; i >= 0; i-- {
		g := strings.TrimSpace(groups[i])
		gl := strings.ToLower(g)
		if gl == "info" || gl == "warn" || gl == "error" || gl == "crit" {
			continue
		}
		if strings.Contains(g, "-") && strings.Contains(g, ":") {
			continue // timestamp YYYY-MM-DD HH:MM:SS
		}
		if gl == "localhost" {
			continue // REMOTE_ADDR без DNS
		}
		if isDigits(g) {
			continue // pid
		}
		if !hasLetter(g) {
			continue // ip-адрес: только цифры, точки и двоеточия
		}
		return normalizeServiceTag(g)
	}
	return ""
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func hasLetter(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return true
		}
	}
	return false
}
