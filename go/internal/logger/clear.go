package logger

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const logDir = "/opt/var/log/entware"

func HandleClear() {
	if !IsPOST() {
		NotAllowed()
		return
	}

	now := time.Now()
	filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".log") {
			return nil
		}
		if now.Sub(info.ModTime()) > 30*24*time.Hour {
			os.Remove(path)
		}
		return nil
	})

	WriteJSON(map[string]string{"status": "ok", "message": "Логи старше 30 дней удалены"})
}
