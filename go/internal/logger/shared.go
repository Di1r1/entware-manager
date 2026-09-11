package logger

import (
	"os"
	"strings"
)

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func hasQuery(key string) bool {
	q := os.Getenv("QUERY_STRING")
	return strings.Contains(q, key)
}

// staticBaseHTML — <base> + NOC-фикс для статики.
// Классика (8087): base=/entware-manager/ → "logger/style.css" = /entware-manager/logger/style.css.
// NOC (8086, iframe): скрипт до загрузки CSS/JS переключает base на
// /entware-commander/entware-manager/ (там симлинки на /opt/web_entware/).
// Все asset-ссылки ниже должны быть ОТНОСИТЕЛЬНЫМИ (без ведущего /),
// чтобы резолвиться через <base>, а не через URL страницы
// (/entware-cgi/logger/ — там статики нет).
const staticBaseHTML = `<base id="static-base" href="/entware-manager/">` + "\n" +
	`<script>(function(){try{var p=location.port||'';` +
	`if(p==='8086'||location.pathname.indexOf('/entware-commander/')===0){` +
	`var b=document.getElementById('static-base');` +
	`if(b)b.setAttribute('href','/entware-commander/entware-manager/');}}catch(e){}})();</script>`
