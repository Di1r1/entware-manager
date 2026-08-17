package logger

import (
	"bytes"
	"entware-manager/internal/cgiutil"
	"os/exec"
	"strconv"
	"strings"

	"entware-manager/internal/auth"
)

const rotateScript = "/opt/web_entware/logger/scripts/rotate.sh"

// RotatedFile — ротированный файл (путь + размер в байтах).
type RotatedFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func HandleRotate() {
	if !cgiutil.IsPOST() {
		cgiutil.NotAllowed()
		return
	}

	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
		return
	}

	cmd := exec.Command(rotateScript)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Ошибка при ротации"})
		return
	}

	rotated := parseRotated(out.String())
	if rotated == nil {
		rotated = []RotatedFile{}
	}

	logSystemEvent("INFO", "Ротация логов: "+rotationSummary(rotated))

	message := "Ротация выполнена"
	if len(rotated) == 0 {
		message = "Ротация выполнена (файлов для ротации нет)"
	}
	cgiutil.WriteJSON(map[string]any{
		"status":  "ok",
		"message": message,
		"rotated": rotated,
	})
}

// rotationSummary формирует перечень ротированных файлов для системного лога:
// "путь (размер), путь (размер)". При пустом списке — "файлов для ротации нет".
func rotationSummary(files []RotatedFile) string {
	if len(files) == 0 {
		return "файлов для ротации нет"
	}
	var parts []string
	for _, f := range files {
		parts = append(parts, f.Path+" ("+sizeHuman(f.Size)+")")
	}
	return strings.Join(parts, ", ")
}

// sizeHuman — человеко-читаемый размер (B/K/M/G), как в UI панели.
func sizeHuman(size int64) string {
	switch {
	case size < 1024:
		return strconv.FormatInt(size, 10) + "B"
	case size < 1048576:
		return strconv.FormatInt(size/1024, 10) + "K"
	case size < 1073741824:
		return strconv.FormatInt(size/1048576, 10) + "M"
	default:
		return strconv.FormatInt(size/1073741824, 10) + "G"
	}
}

// parseRotated разбирает вывод rotate.sh: строки "ROTATED|путь|размер".
func parseRotated(output string) []RotatedFile {
	var files []RotatedFile
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ROTATED|") {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 || parts[1] == "" {
			continue
		}
		size, _ := strconv.ParseInt(parts[2], 10, 64)
		files = append(files, RotatedFile{Path: parts[1], Size: size})
	}
	return files
}
