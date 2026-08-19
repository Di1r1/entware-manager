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

	// Дневной суточный лог (факты демонов [monitor]) + профильный лог действий
	// панели (monitor_actions.log: [ACTION]/[monitor_action]). Объединяем, чтобы
	// вкладка «Защита» показывала полную картину (и факты, и действия).
	var lines []string
	lines = append(lines, readLogLines(dailyLogFile(), "[monitor]", "[action]")...)
	lines = append(lines, readLogLines(monitorActionsLog, "[ACTION]", "[monitor_action]")...)

	// Новые сверху (по timestamp в начале строки [YYYY-MM-DD HH:MM:SS]).
	sortLogLines(lines)

	// Last 200 lines
	start := 0
	if len(lines) > 200 {
		start = len(lines) - 200
	}

	WriteText(strings.Join(lines[start:], "\n") + "\n")
}

// dailyLogFile возвращает путь к дневному суточному логу.
func dailyLogFile() string {
	return fmt.Sprintf("/tmp/entware/logs/%s.log", time.Now().Format("2006-01-02"))
}

// readLogLines читает файл и возвращает строки, содержащие любой из паттернов.
func readLogLines(file string, patterns ...string) []string {
	var out []string
	f, err := os.Open(file)
	if err != nil {
		return out
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		lower := strings.ToLower(line)
		for _, p := range patterns {
			if strings.Contains(lower, strings.ToLower(p)) {
				out = append(out, line)
				break
			}
		}
	}
	return out
}

// sortLogLines сортирует строки логов по timestamp (первые 20 символов
// "[YYYY-MM-DD HH:MM:SS]") — новые сверху. Стабильная сортировка по вставке.
func sortLogLines(lines []string) {
	// Сортировка вставками по убыванию timestamp (без внешних зависимостей).
	for i := 1; i < len(lines); i++ {
		key := lines[i]
		keyTs := tsPrefix(key)
		j := i - 1
		for j >= 0 && tsPrefix(lines[j]) < keyTs {
			lines[j+1] = lines[j]
			j--
		}
		lines[j+1] = key
	}
}

// tsPrefix извлекает префикс "[YYYY-MM-DD HH:MM:SS]" для сравнения.
func tsPrefix(line string) string {
	if len(line) < 20 {
		return line
	}
	return line[:20]
}
