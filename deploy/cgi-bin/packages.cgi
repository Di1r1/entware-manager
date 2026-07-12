#!/bin/sh
# ==============================================
# Entware Manager - список установленных пакетов
# Версия: 0.23 (исправлен парсинг версии через " - ")
# Дата: 2026-03-29
# ==============================================

. /opt/web_entware/lib/common.sh

html_header

pkg_list=$(/opt/bin/opkg list-installed 2>&1)
installed_count=$(echo "$pkg_list" | wc -l)

cat <<HTML
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Установленные пакеты</title>
<link rel="stylesheet" href="/entware-manager/style.css?v=2">
</head>
<body class="packages-body">
<div class="packages-container">
    <h2 style="display: flex; align-items: center; gap: 8px;">
        <span class="stat-icon" style="width: 28px; height: 28px;">
            <svg class="icon" width="28" height="28">
                <use href="/entware-manager/icons.svg?v=2#icon-package"/>
            </svg>
        </span>
        Установленные пакеты ($installed_count)
    </h2>
HTML

if [ -z "$pkg_list" ] || [ "$installed_count" -eq 0 ]; then
    echo '<div class="packages-no-data"> Пакеты не найдены или ошибка opkg</div>'
    echo '</div></body></html>'
    exit 0
fi

cat <<HTML
<div style="display: flex; gap: 8px; align-items: center; margin-bottom: 20px;">
    <div class="search-container" style="display: flex; gap: 8px; align-items: center; flex: 1; background: var(--input-bg); border: 2px solid var(--input-border); border-radius: 40px; padding: 0 12px; transition: border-color 0.3s ease, box-shadow 0.3s ease;">
        <svg class="icon" width="18" height="18" style="color: var(--text-muted);"><use href="/entware-manager/icons.svg?v=2#icon-search"/></svg>
        <input type="text" id="searchInput" placeholder="Поиск по названию..." style="flex: 1; background: transparent; border: none; outline: none; padding: 14px 0; font-size: 16px; color: var(--text-primary);">
    </div>
</div>
<div class="packages-table-wrapper">
    <table class="packages-table" id="packagesTable">
        <thead><th>Пакет</th><th>Версия</th><th>Действие</th></thead>
        <tbody id="tableBody">
HTML

echo "$pkg_list" | awk '
function escape(s) {
    gsub(/&/,"\\&amp;",s)
    gsub(/</,"\\&lt;",s)
    gsub(/>/,"\\&gt;",s)
    gsub(/"/,"\\&quot;",s)
    return s
}
{
    if (NF == 0) next
    n = split($0, parts, " - ")
    if (n < 2) next
    pkg = parts[1]
    ver = parts[2]
    if (ver == "") ver = "?"
    printf "            <tr>\n"
    printf "                <td>%s</td>\n", escape(pkg)
    printf "                <td>%s</td>\n", escape(ver)
    printf "                <td>\n"
    printf "                    <form method=\"post\" style=\"margin:0;\" onsubmit=\"opkgAction(event, '\''remove'\'', this.package.value); return false;\">\n"
    printf "                        <input type=\"hidden\" name=\"package\" value=\"%s\">\n", escape(pkg)
    printf "                        <input type=\"submit\" value=\"Удалить\" class=\"packages-delete-btn\">\n"
    printf "                    </form>\n"
    printf "                </td>\n"
    printf "            </tr>\n"
}
'

cat <<HTML
        </tbody>
    </table>
</div>
</div>
</body>
</html>
HTML
