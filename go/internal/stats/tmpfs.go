package stats

import (
	_ "embed"
	"encoding/json"
	"entware-manager/internal/cgiutil"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

//go:embed tmpfs.html
var tmpfsTemplate string

func HandleTmpfs() {
	path := cgiutil.GetQueryParam("path")
	if path == "" {
		path = "/tmp"
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = filepath.Clean(path)

	// Листинг только из /tmp и /dev/shm (как delete_file/view_file/tmpfs_clean).
	if !cleanableRoot(path) {
		fmt.Print("Content-type: text/html; charset=utf-8\n\n")
		fmt.Printf("<p class='error'>Доступ запрещен: %s</p>", html.EscapeString(path))
		return
	}

	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		fmt.Print("Content-type: text/html; charset=utf-8\n\n")
		fmt.Printf("<p class='error'>Директория не существует: %s</p>", html.EscapeString(path))
		return
	}

	authEnabled := "false"
	if data, err := os.ReadFile("/opt/web_entware/auth_config.json"); err == nil {
		var cfg struct {
			Enabled bool `json:"enabled"`
		}
		if json.Unmarshal(data, &cfg) == nil && cfg.Enabled {
			authEnabled = "true"
		}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Print("Content-type: text/html; charset=utf-8\n\n")
		fmt.Printf("<p class='error'>Ошибка чтения директории: %s</p>", html.EscapeString(err.Error()))
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	html := strings.ReplaceAll(tmpfsTemplate, "{TITLE_PATH}", html.EscapeString(path))
	html = strings.ReplaceAll(html, "{BREADCRUMB}", generateBreadcrumb(path))
	html = strings.ReplaceAll(html, "{UP_LINK}", generateUpLink(path))
	html = strings.ReplaceAll(html, "{FILE_ROWS}", generateFileRows(path, entries))
	html = strings.ReplaceAll(html, "{AUTH_ENABLED}", authEnabled)

	fmt.Print("Content-type: text/html; charset=utf-8\n\n")
	fmt.Print(html)
}

func generateBreadcrumb(path string) string {
	if path == "/" {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	var b strings.Builder
	tempPath := ""
	for _, seg := range parts {
		if seg == "" {
			continue
		}
		tempPath += "/" + seg
		b.WriteString(`<svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=3#icon-chevron-right"/></svg>`)
		if tempPath != path {
			b.WriteString(fmt.Sprintf(` <a href="?path=%s">%s</a> `,
				url.QueryEscape(tempPath), html.EscapeString(seg)))
		} else {
			b.WriteString(fmt.Sprintf(` <span>%s</span> `, html.EscapeString(seg)))
		}
	}
	return b.String()
}

func generateUpLink(path string) string {
	if path == "/" || path == "/tmp" || path == "/dev" || path == "/dev/shm" {
		return ""
	}
	parent := filepath.Dir(path)
	return fmt.Sprintf(`<tr class="row-nav"><td colspan="6"><a href="?path=%s"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=3#icon-arrow-left"/></svg> .. (наверх)</a></td></tr>`,
		url.QueryEscape(parent))
}

func generateFileRows(dir string, entries []os.DirEntry) string {
	var b strings.Builder
	for _, entry := range entries {
		name := entry.Name()
		fullPath := dir + "/" + name
		info, err := entry.Info()
		if err != nil {
			continue
		}

		escName := html.EscapeString(name)
		escPath := html.EscapeString(fullPath)
		qPath := url.QueryEscape(fullPath)

		isDir := entry.IsDir()
		dtype := "file"
		icon := "file"
		iconClass := "file"
		link := fmt.Sprintf(`<a href="#" class="file-link" data-path="%s">%s</a>`, escPath, escName)

		if isDir {
			icon = "folder"
			iconClass = "folder"
			link = fmt.Sprintf(`<a href="?path=%s">%s</a>`, qPath, escName)
			dtype = "dir"
		}

		action := fmt.Sprintf(
			`<button class="delete-file-btn" data-path="%s" data-name="%s" data-type="%s"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=3#icon-trash"/></svg></button>`,
			escPath, escName, dtype,
		)

		hsize := cgiutil.HumanSize(info.Size())
		modTime := info.ModTime().Format("Jan _2 15:04")
		perm := info.Mode().String()

		owner := "?"
		group := "?"
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			owner = strconv.FormatUint(uint64(stat.Uid), 10)
			group = strconv.FormatUint(uint64(stat.Gid), 10)
		}

		b.WriteString(fmt.Sprintf(
			`<tr><td><span class="file-icon %s"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=3#icon-%s"/></svg></span> %s</td><td>%s</td><td>%s</td><td>%s</td><td>%s:%s</td><td>%s</td></tr>`,
			iconClass, icon, link, hsize, modTime, perm, owner, group, action,
		))
	}
	return b.String()
}
