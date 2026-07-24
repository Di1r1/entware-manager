package services

import (
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
	if !IsGET() {
		NotAllowed()
		return
	}

	limit := 20
	if l := getQueryParam("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	events := parseWatchdogLog(limit)
	if events == nil {
		events = []WatchdogEvent{}
	}
	WriteJSON(map[string][]WatchdogEvent{"events": events})
}

func parseWatchdogLog(limit int) []WatchdogEvent {
	logPath := filepath.Join("/tmp/entware/logs", time.Now().Format("2006-01-02")+".log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	tagPattern := "[service]"

	var matched []string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.Contains(strings.ToLower(line), tagPattern) {
			matched = append(matched, line)
			if len(matched) >= limit {
				break
			}
		}
	}

	for i, j := 0, len(matched)-1; i < j; i, j = i+1, j-1 {
		matched[i], matched[j] = matched[j], matched[i]
	}

	events := make([]WatchdogEvent, 0, len(matched))
	for _, line := range matched {
		events = append(events, parseWatchdogLine(line))
	}
	return events
}

func parseWatchdogLine(line string) WatchdogEvent {
	ts := ""
	if idx1 := strings.Index(line, "["); idx1 >= 0 {
		if idx2 := strings.Index(line[idx1:], "]"); idx2 >= 0 {
			ts = strings.TrimSpace(line[idx1+1 : idx1+idx2])
		}
	}

	tagBracket := "[service]"
	tagIdx := strings.Index(strings.ToLower(line), tagBracket)
	if tagIdx < 0 {
		return WatchdogEvent{Timestamp: ts, Level: "INFO"}
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

	return WatchdogEvent{
		Timestamp: ts,
		Level:     level,
		Service:   service,
		Event:     event,
		Details:   details,
	}
}
