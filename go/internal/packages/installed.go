package packages

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const opkgStatusFile = "/opt/lib/opkg/status"

// readInstalledTimes парсит /opt/lib/opkg/status и возвращает
// map[имя пакета] -> unix-время установки (Installed-Time).
func readInstalledTimes() map[string]int64 {
	times := make(map[string]int64)
	data, err := os.ReadFile(opkgStatusFile)
	if err != nil {
		return times
	}
	var curPkg string
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(line, "Package: "):
			curPkg = strings.TrimSpace(strings.TrimPrefix(line, "Package: "))
		case strings.HasPrefix(line, "Installed-Time: ") && curPkg != "":
			ts, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "Installed-Time: ")), 10, 64)
			if err == nil && ts > 0 {
				times[curPkg] = ts
			}
			curPkg = ""
		}
	}
	return times
}

func Installed() {
	if !isGET() {
		methodNotAllowed()
		return
	}

	out, code := runOpkg("list-installed")
	pkgList := strings.TrimSpace(out)
	count := 0
	if pkgList != "" {
		count = len(strings.Split(pkgList, "\n"))
	}

	html := "<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"UTF-8\">\n<title>Установленные пакеты</title>\n<link rel=\"stylesheet\" href=\"/entware-manager/style.css?v=34\">\n<script src=\"/entware-manager/theme.js?v=2\"></script>\n</head>\n<body class=\"packages-body\">\n<script>if (window.Theme) Theme.init();</script>\n<div class=\"packages-container\">\n"
	html += fmt.Sprintf("<h2 style=\"display: flex; align-items: center; gap: 8px;\"><span class=\"stat-icon\" style=\"width: 28px; height: 28px;\"><svg class=\"icon\" width=\"28\" height=\"28\"><use href=\"/entware-manager/icons.svg?v=2#icon-package\"/></svg></span>Установленные пакеты (%d)</h2>\n", count)

	if code != 0 || pkgList == "" || count == 0 {
		html += "<div class=\"packages-no-data\"> Пакеты не найдены или ошибка opkg</div>\n</div></body></html>\n"
		writeHTML(html)
		return
	}

	html += `<div style="display: flex; gap: 8px; align-items: center; margin-bottom: 20px;">
    <div class="search-container" style="display: flex; gap: 8px; align-items: center; flex: 1; background: var(--input-bg); border: 2px solid var(--input-border); border-radius: 40px; padding: 0 12px; transition: border-color 0.3s ease, box-shadow 0.3s ease;">
        <svg class="icon" width="18" height="18" style="color: var(--text-muted);"><use href="/entware-manager/icons.svg?v=2#icon-search"/></svg>
        <input type="text" id="searchInput" placeholder="Поиск по названию..." style="flex: 1; background: transparent; border: none; outline: none; padding: 14px 0; font-size: 16px; color: var(--text-primary);">
    </div>
</div>
<div class="packages-table-wrapper">
    <table class="packages-table" id="packagesTable">
        <thead><th>Пакет</th><th>Версия</th><th>Установлен</th><th>Действие</th></thead>
        <tbody id="tableBody">
`

	installedTimes := readInstalledTimes()

	lines := strings.Split(pkgList, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) < 2 {
			continue
		}
		pkg := parts[0]
		ver := parts[1]
		if ver == "" {
			ver = "?"
		}
		instDate := "—"
		if ts, ok := installedTimes[pkg]; ok {
			instDate = time.Unix(ts, 0).Format("2006-01-02 15:04")
		}
		html += fmt.Sprintf("            <tr>\n                <td>%s</td>\n                <td>%s</td>\n                <td>%s</td>\n                <td>\n                    <form method=\"post\" style=\"margin:0;\" onsubmit=\"opkgAction(event, 'remove', this.package.value); return false;\">\n                        <input type=\"hidden\" name=\"package\" value=\"%s\">\n                        <input type=\"submit\" value=\"Удалить\" class=\"packages-delete-btn\">\n                    </form>\n                </td>\n            </tr>\n",
			htmlEscape(pkg), htmlEscape(ver), htmlEscape(instDate), htmlEscape(pkg))
	}

	html += `        </tbody>
    </table>
</div>
</div>
</body>
</html>`

	writeHTML(html)
}

// InstalledPkg — JSON-представление установленного пакета (для единой вкладки «Пакеты»).
type InstalledPkg struct {
	Package       string `json:"package"`
	Version       string `json:"version"`
	InstalledDate string `json:"installed_date,omitempty"`
}

// InstalledJSON отдаёт список установленных пакетов как JSON:
// [{package, version, installed_date}] — версии из opkg list-installed,
// дата установки из /opt/lib/opkg/status.
func InstalledJSON() {
	if !isGET() {
		methodNotAllowed()
		return
	}

	out, code := runOpkg("list-installed")
	if code != 0 {
		writeJSON([]InstalledPkg{})
		return
	}

	installedTimes := readInstalledTimes()

	lines := strings.Split(strings.TrimSpace(out), "\n")
	pkgs := make([]InstalledPkg, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) < 2 {
			continue
		}
		pkg := parts[0]
		ver := parts[1]
		if ver == "" {
			ver = "?"
		}
		instDate := ""
		if ts, ok := installedTimes[pkg]; ok {
			instDate = time.Unix(ts, 0).Format("2006-01-02 15:04")
		}
		pkgs = append(pkgs, InstalledPkg{Package: pkg, Version: ver, InstalledDate: instDate})
	}

	writeJSON(pkgs)
}
