package network

import (
	"entware-manager/internal/cgiutil"
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

	events := parseLogFile(tag, limit)
	if events == nil {
		events = []LogEvent{}
	}
	cgiutil.WriteJSON(map[string][]LogEvent{"events": events})
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
	tagPattern := "[" + strings.ToLower(tag) + "]"
	matchAll := tag == "all"

	var matched []string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if matchAll || strings.Contains(lower, tagPattern) {
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

	tagBracket := "[" + tag + "]"
	tagIdx := strings.Index(strings.ToLower(line), strings.ToLower(tagBracket))
	if tagIdx < 0 {
		return LogEvent{Timestamp: ts, Level: "INFO"}
	}

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
	return LogEvent{Timestamp: ts, Level: level, Event: text}
}