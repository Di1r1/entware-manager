package packages

import (
	"fmt"
	"strings"
)

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

	html := "<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"UTF-8\">\n<title>Установленные пакеты</title>\n<link rel=\"stylesheet\" href=\"/entware-manager/style.css?v=23\">\n<script src=\"/entware-manager/theme.js?v=2\"></script>\n</head>\n<body class=\"packages-body\">\n<script>if (window.Theme) Theme.init();</script>\n<div class=\"packages-container\">\n"
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
        <thead><th>Пакет</th><th>Версия</th><th>Действие</th></thead>
        <tbody id="tableBody">
`

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
		html += fmt.Sprintf("            <tr>\n                <td>%s</td>\n                <td>%s</td>\n                <td>\n                    <form method=\"post\" style=\"margin:0;\" onsubmit=\"opkgAction(event, 'remove', this.package.value); return false;\">\n                        <input type=\"hidden\" name=\"package\" value=\"%s\">\n                        <input type=\"submit\" value=\"Удалить\" class=\"packages-delete-btn\">\n                    </form>\n                </td>\n            </tr>\n",
			htmlEscape(pkg), htmlEscape(ver), htmlEscape(pkg))
	}

	html += `        </tbody>
    </table>
</div>
</div>
</body>
</html>`

	writeHTML(html)
}
