package packages

import (
	"fmt"
)

func Remove() {
	if !isPOST() {
		writeHTML(`<p class="error">Ошибка: требуется POST-запрос</p>`)
		return
	}

	body := readBody()
	pkgRaw := parsePostParam(body, "package")
	pkgClean := sanitizePkg(pkgRaw)

	if pkgClean == "" {
		writeHTML(`<p class="error">Недопустимое имя пакета</p>`)
		return
	}

	html := fmt.Sprintf("<h2>Удаление пакета: %s</h2>\n<pre>\n", htmlEscape(pkgClean))

	out, code := runOpkg("remove", pkgClean)
	html += out + "</pre>\n"

	if code == 0 {
		html += `<p class="success">Пакет успешно удалён.</p>`
		logPackageChange("remove", pkgClean, "success")
	} else {
		html += `<p class="error">Ошибка при удалении. Проверьте логи opkg.</p>`
		logPackageChange("remove", pkgClean, "error")
	}

	writeHTML(html)
}
