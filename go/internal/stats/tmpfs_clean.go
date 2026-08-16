package stats

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"entware-manager/internal/auth"
)

// tmpfsProtected — каталоги демонов, которые нельзя предлагать к очистке,
// чтобы не загубить их случайно (koffe, nginx, логи менеджера в /tmp/entware).
var tmpfsProtected = map[string]bool{
	"koffe":   true,
	"nginx":   true,
	"entware": true,
}

// CleanDir — найденный объект (подпапка или файл), кандидат на удаление.
type CleanDir struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Files int    `json:"files"`
	Type  string `json:"type"` // "dir" | "file"
}

type cleanScanResponse struct {
	AuthRequired bool       `json:"auth_required"`
	Path         string     `json:"path"`
	Dirs         []CleanDir `json:"dirs"`
}

type cleanDeleteResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Deleted int    `json:"deleted"`
}

func HandleTmpClean() {
	switch os.Getenv("REQUEST_METHOD") {
	case "GET":
		scanTmpClean()
	case "POST":
		if auth.IsCrossSiteOrigin() {
			fmt.Print("Content-type: application/json; charset=utf-8\n\n")
			fmt.Println(`{"status":"error","message":"` + auth.CrossSiteDeny + `"}`)
			return
		}
		deleteTmpClean()
	default:
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Метод не поддерживается"}`)
	}
}

// scanTmpClean — GET: сканирует корень, возвращает JSON-список подпапок,
// чей рекурсивный размер >= min_bytes.
func scanTmpClean() {
	qs := os.Getenv("QUERY_STRING")
	path := getQueryParam(qs, "path")
	if path == "" {
		path = "/tmp"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = filepath.Clean(path)

	minBytes := int64(1 << 20) // 1 МиБ
	if mb := getQueryParam(qs, "min_bytes"); mb != "" {
		if v, err := strconv.ParseInt(mb, 10, 64); err == nil && v > 0 {
			minBytes = v
		}
	}

	rootInfo, err := os.Stat(path)
	if err != nil || !rootInfo.IsDir() {
		jsonClean("Директория не существует: " + path)
		return
	}
	rootDev := devOf(rootInfo)

	entries, err := os.ReadDir(path)
	if err != nil {
		jsonClean("Ошибка чтения директории: " + path)
		return
	}

	dirs := make([]CleanDir, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if tmpfsProtected[name] {
			continue
		}
		full := path + "/" + name
		info, err := os.Lstat(full)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if info.IsDir() {
			// не уходим в другие точки монтирования
			if devOf(info) != rootDev {
				continue
			}
			bytes, files := dirSizePath(full)
			if bytes < minBytes {
				continue
			}
			dirs = append(dirs, CleanDir{Name: name, Path: full, Bytes: bytes, Files: files, Type: "dir"})
		} else {
			// крупный файл — тоже кандидат на очистку
			if info.Size() < minBytes {
				continue
			}
			dirs = append(dirs, CleanDir{Name: name, Path: full, Bytes: info.Size(), Files: 1, Type: "file"})
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Bytes > dirs[j].Bytes })

	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	json.NewEncoder(os.Stdout).Encode(cleanScanResponse{
		AuthRequired: authConfigEnabled(),
		Path:         path,
		Dirs:         dirs,
	})
}

// deleteTmpClean — удаляет выбранные подпапки (POST paths=...&password=...).
func deleteTmpClean() {
	body, _ := io.ReadAll(os.Stdin)
	params := parsePostForm(string(body))

	pathStr := params["paths"]
	if pathStr == "" {
		pathStr = params["path"]
	}
	var paths []string
	if strings.Contains(pathStr, "\x00") {
		paths = strings.Split(pathStr, "\x00")
	} else {
		// frontend кодирует пути через '\n'
		paths = strings.Split(pathStr, "\n")
	}
	for i := range paths {
		paths[i] = filepath.Clean(paths[i])
	}

	// валидация до авторизации
	for _, p := range paths {
		if !isCleanablePath(p) {
			logDeleteAction("WARN", fmt.Sprintf("Попытка очистки недопустимого пути: %s", p))
			jsonClean("Доступ запрещен")
			return
		}
	}
	if len(paths) == 0 {
		jsonClean("Пути не указаны")
		return
	}
	if !checkFilemgrAuth(params["password"]) {
		logDeleteAction("WARN", "Неверный пароль при очистке tmpfs")
		jsonClean("Неверный пароль")
		return
	}

	deleted := 0
	for _, p := range paths {
		if p == "/" {
			continue
		}
		fi, err := os.Stat(p)
		if err != nil {
			logDeleteAction("WARN", fmt.Sprintf("Не удалось очистить: %s (%v)", p, err))
			continue
		}
		if fi.IsDir() {
			if err := os.RemoveAll(p); err != nil {
				logDeleteAction("WARN", fmt.Sprintf("Не удалось очистить: %s (%v)", p, err))
				continue
			}
		} else {
			if err := os.Remove(p); err != nil {
				logDeleteAction("WARN", fmt.Sprintf("Не удалось удалить файл: %s (%v)", p, err))
				continue
			}
		}
		logDeleteAction("INFO", fmt.Sprintf("Тmpfs-очистка: удалено %s", p))
		deleted++
	}

	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	fmt.Printf(`{"status":"ok","deleted":%d}`+"\n", deleted)
}

// isCleanablePath — допустимость пути для удаления: только подпапки
// /tmp и /dev/shm, без ".." и путей, меняющих смысл после Clean.
func isCleanablePath(p string) bool {
	if p == "" || p == "/" {
		return false
	}
	if strings.Contains(p, "..") {
		return false
	}
	if filepath.Clean(p) != p {
		return false
	}
	return strings.HasPrefix(p, "/tmp/") || strings.HasPrefix(p, "/dev/shm/")
}

// dirSizePath считает рекурсивный размер и число файлов, не пересекая
// симлинки (WalkDir их не проходит) и не выходя за границы ФС.
func dirSizePath(root string) (bytes int64, files int) {
	var rootDev uint64
	if fi, err := os.Stat(root); err == nil {
		rootDev = devOf(fi)
	}
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if p == root {
			return nil
		}
		if d.IsDir() {
			// не спускаемся в другую ФС
			if fi, err := os.Stat(p); err == nil && devOf(fi) != rootDev {
				return filepath.SkipDir
			}
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		files++
		if fi, err := d.Info(); err == nil {
			bytes += fi.Size()
		}
		return nil
	})
	return bytes, files
}

func devOf(fi os.FileInfo) uint64 {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Dev)
	}
	return 0
}

func authConfigEnabled() bool {
	data, err := os.ReadFile("/opt/web_entware/auth_config.json")
	if err != nil {
		return false
	}
	var cfg struct {
		Enabled bool `json:"enabled"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return false
	}
	return cfg.Enabled
}
func jsonClean(msg string) {
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	fmt.Printf(`{"status":"error","message":%s}`+"\n", jstr(msg))
}

func jstr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
