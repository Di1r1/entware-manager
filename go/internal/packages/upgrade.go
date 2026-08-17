package packages

import (
	"entware-manager/internal/cgiutil"
	"fmt"

	"entware-manager/internal/cache"
)

func Upgrade() {
	if !cgiutil.IsPOST() {
		writeHTML(`<p class="error">Ошибка: требуется POST-запрос</p>`)
		return
	}

	body := cgiutil.ReadPOSTBody()
	upgradeAll := cgiutil.ParseFormBody(body)["upgrade_all"]
	pkgRaw := cgiutil.ParseFormBody(body)["package"]

	if upgradeAll == "1" {
		html := "<h2>Обновление всех пакетов...</h2>\n<pre>\n"
		out, code := runOpkg("upgrade")
		html += htmlEscape(out) + "</pre>\n"
		if code == 0 {
			html += `<p class="success">Все пакеты обновлены.</p>`
			logPackageChange("upgrade-all", "all-packages", "success")
			cache.Invalidate("opkg_installed", "opkg_list")
		} else {
			html += `<p class="error">Ошибка при обновлении.</p>`
			logPackageChange("upgrade-all", "all-packages", "error")
		}
		writeHTML(html)
		return
	}

	pkgClean := sanitizePkg(pkgRaw)
	if pkgClean == "" {
		writeHTML(`<p class="error">Недопустимое имя пакета</p>`)
		return
	}

	html := fmt.Sprintf("<h2>Обновление пакета: %s</h2>\n<pre>\n", htmlEscape(pkgClean))
	out, code := runOpkg("upgrade", pkgClean)
	html += htmlEscape(out) + "</pre>\n"

	if code == 0 {
		html += `<p class="success">Пакет успешно обновлён.</p>`
		logPackageChange("upgrade", pkgClean, "success")
		cache.Invalidate("opkg_installed", "opkg_list")
	} else {
		html += `<p class="error">Ошибка при обновлении. Проверьте логи opkg.</p>`
		logPackageChange("upgrade", pkgClean, "error")
	}

	writeHTML(html)
}
