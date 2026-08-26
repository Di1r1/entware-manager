package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"entware-manager/internal/auth"
)

var webRoot = "/opt/web_entware"

// Лимиты размера для распаковки бэкапа (защита от gzip-bomb и DoS).
const (
	maxEntrySize   = 16 << 20 // 16 MiB на запись
	maxArchiveSize = 64 << 20 // 64 MiB на весь архив
)

type configFile struct {
	Path string // relative to webRoot
}

var configs = []configFile{
	{Path: "links.json"},
	{Path: "monitor_config.json"},
	{Path: "network_config.json"},
	{Path: "service_config.json"},
	{Path: "logger/config.json"},
}

// secretDest — файлы, которые при восстановлении обязаны получить 0600
// (хеш пароля панели, токен бота, пароли приложений моста, его настройки).
func secretDest(dest string) bool {
	return dest == "auth_config.json" || dest == "telegram_config.json" ||
		dest == "bridge/_prefs.json" || strings.HasSuffix(dest, ".auth.json")
}

// BuildArchive собирает архив бэкапа конфигурации в памяти
// (конфиги + список пакетов + info). Экспортировано для чат-бота.
func BuildArchive() ([]byte, error) {
	cleanupOldTemp("entware-backup-")
	tmpDir, err := os.MkdirTemp("", "entware-backup-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	for _, cf := range configs {
		src := filepath.Join(webRoot, cf.Path)
		dst := filepath.Join(tmpDir, strings.ReplaceAll(cf.Path, "/", "_"))
		if data, err := os.ReadFile(src); err == nil {
			os.WriteFile(dst, data, 0644)
		}
	}

	// Мост сервисов: пользовательские манифесты, секреты приложений
	// (*.auth.json) и настройки (галочки уведомлений/управления).
	if bridges, err := os.ReadDir(filepath.Join(webRoot, "bridge")); err == nil {
		for _, e := range bridges {
			if e.IsDir() || filepath.Ext(e.Name()) != ".json" || strings.HasSuffix(e.Name(), ".tmp") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(webRoot, "bridge", e.Name()))
			if err != nil {
				continue
			}
			os.WriteFile(filepath.Join(tmpDir, "bridge_"+e.Name()), data, 0600)
		}
	}

	// Секреты панели: хеш пароля и токен Telegram-бота.
	for _, name := range []string{"auth_config.json", "telegram_config.json"} {
		if data, err := os.ReadFile(filepath.Join(webRoot, name)); err == nil {
			os.WriteFile(filepath.Join(tmpDir, name), data, 0600)
		}
	}

	// Пользовательские источники системных логов логгера.
	if data, err := os.ReadFile(filepath.Join(webRoot, "logger", "system_sources.json")); err == nil {
		os.WriteFile(filepath.Join(tmpDir, "logger_system_sources.json"), data, 0644)
	}

	pkgList, _ := exec.Command("opkg", "list-installed").Output()
	if len(pkgList) > 0 {
		os.WriteFile(filepath.Join(tmpDir, "packages.txt"), pkgList, 0644)
	}

	info := backupInfo()
	infoJSON, _ := json.MarshalIndent(info, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "backup.json"), infoJSON, 0644)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	entries, _ := os.ReadDir(tmpDir)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(tmpDir, entry.Name()))
		if err != nil {
			continue
		}
		tw.WriteHeader(&tar.Header{
			Name: entry.Name(),
			Size: int64(len(data)),
			Mode: 0644,
		})
		tw.Write(data)
	}

	tw.Close()
	gw.Close()

	return buf.Bytes(), nil
}

func HandleCreate() {
	data, err := BuildArchive()
	if err != nil {
		fmt.Print("Content-type: text/plain; charset=utf-8\n\n")
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Content-type: application/gzip; charset=utf-8")
	fmt.Println("Content-Disposition: attachment; filename=entware-manager-backup.tar.gz")
	fmt.Println("Content-Length:", len(data))
	fmt.Println()
	os.Stdout.Write(data)
}

func HandleRestore() {
	if os.Getenv("REQUEST_METHOD") != "POST" {
		fmt.Print("Content-type: text/plain; charset=utf-8\n\n")
		fmt.Println("Error: POST required")
		return
	}

	if auth.IsCrossSiteOrigin() {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Printf(`{"status":"error","message":%q}`+"\n", auth.CrossSiteDeny)
		return
	}

	body, err := io.ReadAll(io.LimitReader(os.Stdin, maxArchiveSize+1))
	if err != nil || len(body) < 100 {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Empty or too small file"}`)
		return
	}

	cleanupOldTemp("entware-restore-")
	tmpDir, err := os.MkdirTemp("", "entware-restore-*")
	if err != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Cannot create temp dir"}`)
		return
	}
	defer os.RemoveAll(tmpDir)

	gr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Invalid gzip data"}`)
		return
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Print("Content-type: application/json; charset=utf-8\n\n")
			fmt.Println(`{"status":"error","message":"Invalid tar archive"}`)
			return
		}

		// Безопасность: только обычные файлы (не symlink/dir/hardlink/device),
		// с плоским именем внутри tmpDir. Проверки ДО чтения данных.
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue
		}
		name := header.Name
		if name != filepath.Base(name) || name == "." || name == ".." {
			continue
		}

		data, err := io.ReadAll(io.LimitReader(tr, maxEntrySize+1))
		if err != nil || len(data) > maxEntrySize {
			continue
		}

		os.WriteFile(filepath.Join(tmpDir, name), data, 0644)
	}

	// Восстановление: плоские имена архива → пути в webRoot.
	// Старые имена (links/monitor/...) совместимы; новые — мост (bridge_*),
	// секреты панели и источники логов. Секреты получают 0600.
	var restored []string
	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		var dest string
		switch name {
		case "links.json", "monitor_config.json", "network_config.json", "service_config.json",
			"auth_config.json", "telegram_config.json":
			dest = name
		case "logger_config.json":
			dest = "logger/config.json"
		case "logger_system_sources.json":
			dest = "logger/system_sources.json"
		default:
			if strings.HasPrefix(name, "bridge_") && strings.HasSuffix(name, ".json") &&
				len(name) > len("bridge_") {
				dest = "bridge/" + strings.TrimPrefix(name, "bridge_")
			} else {
				continue // неизвестная запись — пропускаем
			}
		}
		data, err := os.ReadFile(filepath.Join(tmpDir, name))
		if err != nil {
			continue
		}
		dst := filepath.Join(webRoot, dest)
		os.MkdirAll(filepath.Dir(dst), 0755)
		mode := os.FileMode(0644)
		if secretDest(dest) {
			mode = 0600
		}
		if err := os.WriteFile(dst, data, mode); err != nil {
			continue
		}
		restored = append(restored, dest)
	}

	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	if len(restored) > 0 {
		logRestoreAction("INFO", "Восстановление конфигов: "+strings.Join(restored, ", "))
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":   "ok",
			"message":  fmt.Sprintf("Restored: %s", strings.Join(restored, ", ")),
			"restored": restored,
		})
	} else {
		logRestoreAction("WARN", "Восстановление: не найдены конфиги в архиве")
		fmt.Println(`{"status":"error","message":"No config files found in backup"}`)
	}
}

// cleanupOldTemp удаляет временные папки с указанным префиксом старше 24 часов.
// Защита: папки, изменённые за последние 24 часа, не трогаем — операция может быть активной.
func cleanupOldTemp(prefix string) {
	dirs, _ := filepath.Glob(filepath.Join(os.TempDir(), prefix+"*"))
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, d := range dirs {
		fi, err := os.Stat(d)
		if err != nil || !fi.IsDir() {
			continue
		}
		if fi.ModTime().Before(cutoff) {
			os.RemoveAll(d)
		}
	}
}

// backupInfo читает актуальную версию/дату из version.json.
func backupInfo() map[string]string {
	info := map[string]string{"version": "unknown", "date": ""}
	data, err := os.ReadFile(filepath.Join(webRoot, "version.json"))
	if err != nil {
		return info
	}
	var v struct {
		Version string `json:"version"`
		Date    string `json:"date"`
	}
	if json.Unmarshal(data, &v) != nil {
		return info
	}
	if v.Version != "" {
		info["version"] = v.Version
	}
	if v.Date != "" {
		info["date"] = v.Date
	}
	return info
}

// logRestoreAction пишет событие восстановления в дневной суточный лог
// с тегом [backup] — его читает Telegram-шлюз (source=system).
func logRestoreAction(level, msg string) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	ip := os.Getenv("REMOTE_ADDR")
	if ip == "" {
		ip = "localhost"
	}
	logDir := "/tmp/entware/logs"
	os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] [%s] [%s] [%d] [backup] %s\n", ts, level, ip, os.Getpid(), msg)
}
