package monitor

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func HandleLog() {
	if !IsGET() {
		NotAllowed()
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
		if strings.Contains(strings.ToLower(line), "[monitor]") {
			lines = append(lines, line)
		}
	}

	// Take last 200 lines
	start := 0
	if len(lines) > 200 {
		start = len(lines) - 200
	}

	WriteText(strings.Join(lines[start:], "\n") + "\n")
}
