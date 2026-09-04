// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// auth_log.cgi — история попыток входа в панель для вкладки
// «Настройки → Защита панели». Читает сегодняшний и вчерашний суточный лог,
// фильтрует строки [login.cgi] (пишет их logAuthAction в login.go):
//
//	[2026-08-23 15:04:05] [WARN] [192.168.3.5] [1234] [login.cgi] Неверный пароль при входе
//
// GET-only, максимум maxAuthLogEntries записей (свежие сверху).
package stats

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"

	"entware-manager/internal/cgiutil"
)

// logsDir — каталог суточных логов (переопределяется в тестах).
var logsDir = "/tmp/entware/logs"

// maxAuthLogEntries — лимит выдачи.
const maxAuthLogEntries = 50

type authLogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	IP      string `json:"ip"`
	Message string `json:"message"`
}

// HandleAuthLog обрабатывает auth_log.cgi (GET).
func HandleAuthLog() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	now := time.Now()
	entries, failed24h := collectAuthEntries(now)

	cgiutil.WriteJSON(map[string]interface{}{
		"status":     "ok",
		"entries":    entries,
		"failed_24h": failed24h,
	})
}

// collectAuthEntries собирает записи из вчерашнего и сегодняшнего логов.
// Порядок файлов вчера→сегодня + общий разворот = свежие сверху.
func collectAuthEntries(now time.Time) ([]authLogEntry, int) {
	files := []string{
		filepath.Join(logsDir, now.AddDate(0, 0, -1).Format("2006-01-02")+".log"),
		filepath.Join(logsDir, now.Format("2006-01-02")+".log"),
	}

	var entries []authLogEntry
	failed24h := 0
	dayAgo := now.Add(-24 * time.Hour)

	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			continue // лога нет — норма
		}
		sc := bufio.NewScanner(fh)
		for sc.Scan() {
			e, ok := parseAuthLine(sc.Text())
			if !ok {
				continue
			}
			if e.Level == "WARN" && isFailedLogin(e.Message) && isWithin24h(e.Time, dayAgo, now) {
				failed24h++
			}
			entries = append(entries, e)
		}
		fh.Close()
	}

	// свежие сверху
	reverseEntries(entries)
	if len(entries) > maxAuthLogEntries {
		entries = entries[:maxAuthLogEntries]
	}
	if entries == nil {
		entries = []authLogEntry{}
	}
	return entries, failed24h
}

// parseAuthLine разбирает строку лога login.cgi.
func parseAuthLine(line string) (authLogEntry, bool) {
	var e authLogEntry
	if !strings.Contains(line, "[login.cgi]") {
		return e, false
	}
	rest := strings.TrimPrefix(line, "[")
	parts := strings.SplitN(rest, "] [", 5) // время ] уровень ] ip ] pid ] [login.cgi] сообщение
	if len(parts) != 5 {
		return e, false
	}
	e.Time = strings.TrimSpace(parts[0])
	e.Level = strings.TrimSpace(parts[1])
	e.IP = strings.TrimSpace(parts[2])
	// parts[4] после SplitN по "] [" содержит "login.cgi] сообщение"
	// (открывающая скобка тега съедена разделителем).
	msg := strings.TrimPrefix(parts[4], "[login.cgi]")
	msg = strings.TrimPrefix(msg, "login.cgi]")
	e.Message = strings.TrimSpace(msg)
	return e, e.IP != "" && e.Message != ""
}

// isFailedLogin — только фактически неверный пароль; прочие WARN («Вход
// отклонён», блокировки) в счётчик неудач за 24ч не попадают.
func isFailedLogin(msg string) bool {
	return strings.Contains(msg, "Неверный пароль")
}

// isWithin24h — запись новее dayAgo (время в формате лога).
func isWithin24h(ts string, dayAgo, now time.Time) bool {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", ts, time.Local)
	if err != nil {
		return false
	}
	return t.After(dayAgo) && !t.After(now)
}

func reverseEntries(e []authLogEntry) {
	for i, j := 0, len(e)-1; i < j; i, j = i+1, j-1 {
		e[i], e[j] = e[j], e[i]
	}
}
