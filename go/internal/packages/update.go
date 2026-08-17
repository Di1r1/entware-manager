package packages

func Update() {
	if !isGET() {
		methodNotAllowed()
		return
	}

	html := "<h2>Обновление списков пакетов</h2>\n<pre>\n"

	out, code := runOpkg("update")
	html += htmlEscape(out) + "</pre>\n"

	if code == 0 {
		html += `<p class="success">Списки пакетов успешно обновлены.</p>`
	} else {
		html += `<p class="error">Ошибка обновления списков пакетов</p>`
	}

	writeHTML(html)
}
