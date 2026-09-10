package packages

import (
	"entware-manager/internal/cgiutil"
	"os"
	"strconv"
	"strings"
)

const packageChangesLog = "/opt/var/log/package_changes.log"

// PackageChangesLog — путь к журналу (var для переопределения в тестах).
var PackageChangesLog = packageChangesLog

// PkgLogEntry — одна запись журнала изменений пакетов.
// Формат строки: "2026-08-28 19:44:20 | remove | 7-zip | success"
type PkgLogEntry struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Package   string `json:"package"`
	Status    string `json:"status"`
}

func parsePackageLogLine(line string) PkgLogEntry {
	line = strings.TrimSpace(line)
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	e := PkgLogEntry{}
	if len(parts) > 0 {
		e.Timestamp = parts[0]
	}
	if len(parts) > 1 {
		e.Action = parts[1]
	}
	if len(parts) > 2 {
		e.Package = parts[2]
	}
	if len(parts) > 3 {
		e.Status = parts[3]
	}
	return e
}

// CollectPackageLog — сырые записи журнала (новые сверху), для tag=packages в events.
func CollectPackageLog(limit int) []PkgLogEntry {
	if limit <= 0 {
		limit = 50
	}
	data, err := os.ReadFile(PackageChangesLog)
	if err != nil {
		return []PkgLogEntry{}
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	// Новые сверху
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	entries := make([]PkgLogEntry, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		entries = append(entries, parsePackageLogLine(line))
	}
	return entries
}

func HandlePackagesLog() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}
	limit := 50
	if l := cgiutil.GetParam("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	data, err := os.ReadFile(PackageChangesLog)
	if err != nil {
		cgiutil.WriteJSON([]PkgLogEntry{})
		return
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	// Новые сверху
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	entries := make([]PkgLogEntry, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		entries = append(entries, parsePackageLogLine(line))
	}
	cgiutil.WriteJSON(entries)
}