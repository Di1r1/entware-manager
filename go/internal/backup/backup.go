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

func HandleCreate() {
	cleanupOldTemp("entware-backup-")
	tmpDir, err := os.MkdirTemp("", "entware-backup-*")
	if err != nil {
		fmt.Print("Content-type: text/plain; charset=utf-8\n\n")
		fmt.Println("Error: cannot create temp dir:", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	for _, cf := range configs {
		src := filepath.Join(webRoot, cf.Path)
		dst := filepath.Join(tmpDir, strings.ReplaceAll(cf.Path, "/", "_"))
		if data, err := os.ReadFile(src); err == nil {
			os.WriteFile(dst, data, 0644)
		}
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

	fmt.Println("Content-type: application/gzip; charset=utf-8")
	fmt.Println("Content-Disposition: attachment; filename=entware-manager-backup.tar.gz")
	fmt.Println("Content-Length:", buf.Len())
	fmt.Println()
	os.Stdout.Write(buf.Bytes())
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

	restoreFile := func(tmpName, destRel string) error {
		src := filepath.Join(tmpDir, tmpName)
		if _, err := os.Stat(src); err != nil {
			return err
		}
		dst := filepath.Join(webRoot, destRel)
		os.MkdirAll(filepath.Dir(dst), 0755)
		data, _ := os.ReadFile(src)
		return os.WriteFile(dst, data, 0644)
	}

	type restoreMap struct {
		tmpName string
		dest    string
	}

	restoreList := []restoreMap{
		{"links.json", "links.json"},
		{"monitor_config.json", "monitor_config.json"},
		{"network_config.json", "network_config.json"},
		{"service_config.json", "service_config.json"},
		{"logger_config.json", "logger/config.json"},
	}

	var restored []string
	for _, rm := range restoreList {
		if err := restoreFile(rm.tmpName, rm.dest); err == nil {
			restored = append(restored, rm.dest)
		}
	}

	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	if len(restored) > 0 {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":   "ok",
			"message":  fmt.Sprintf("Restored: %s", strings.Join(restored, ", ")),
			"restored": restored,
		})
	} else {
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
