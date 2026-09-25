package logger

import (
	"entware-manager/internal/cgiutil"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"entware-manager/internal/auth"
)

// logDir — каталог журналов (var, а не const: подменяется в тестах).
var logDir = "/opt/var/log/entware"

func HandleClear() {
	if !cgiutil.IsPOST() {
		cgiutil.NotAllowed()
		return
	}

	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
		return
	}

	now := time.Now()
	var deleted, failed int
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
		if now.Sub(info.ModTime()) <= 30*24*time.Hour {
			return nil
		}
		if err := os.Remove(path); err != nil {
			failed++
			return nil
		}
		deleted++
		return nil
	})

	if failed > 0 {
		cgiutil.WriteJSON(map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Удалено %d, не удалось удалить %d (логи старше 30 дней)", deleted, failed),
			"deleted": deleted,
			"failed":  failed,
		})
		return
	}
	cgiutil.WriteJSON(map[string]interface{}{
		"status":  "ok",
		"message": "Логи старше 30 дней удалены",
		"deleted": deleted,
		"failed":  failed,
	})
}
