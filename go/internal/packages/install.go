package packages

import (
	"entware-manager/internal/cgiutil"
	"fmt"

	"entware-manager/internal/cache"
)

func invalidateOpkgCache() {
	cache.Invalidate("opkg_installed", "opkg_list")
}

func Install() {
	if !cgiutil.IsPOST() {
		writeHTML(`<p class="error">Ошибка: требуется POST-запрос</p>`)
		return
	}

	body := cgiutil.ReadPOSTBody()
	pkgRaw := cgiutil.ParseFormBody(body)["package"]
	pkgClean := sanitizePkg(pkgRaw)

	if pkgClean == "" {
		writeHTML(`<p class="error">Недопустимое имя пакета</p>`)
		return
	}

	html := fmt.Sprintf("<h2>Установка пакета: %s</h2>\n<pre>\n", htmlEscape(pkgClean))

	out, code := runOpkg("install", pkgClean)
	html += htmlEscape(out) + "</pre>\n"

	if code == 0 {
		html += `<p class="success">Пакет успешно установлен.</p>`
		logPackageChange("install", pkgClean, "success")
		invalidateOpkgCache()
	} else {
		html += `<p class="error">Ошибка при установке. Проверьте логи opkg.</p>`
		logPackageChange("install", pkgClean, "error")
	}

	writeHTML(html)
}
