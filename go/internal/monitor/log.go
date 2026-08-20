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

	WriteText(strings.Join(lines[start:], "\n") + "\n")
}
