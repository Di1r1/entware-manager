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
		if strings.Contains(lower, "[monitor]") || strings.Contains(lower, "[action]") {
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

	WriteText(strings.Join(lines[start:], "\n") + "\n")
}
