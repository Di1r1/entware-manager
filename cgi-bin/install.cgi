#!/bin/sh
# ==============================================
# Entware Manager - установка пакета
# Версия: 0.06 (использование common.sh, без ссылки close)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

html_header

if [ "$REQUEST_METHOD" != "POST" ]; then
    echo '<p class="error">Ошибка: требуется POST-запрос</p>'
    exit 0
fi

_POST_BODY=$(cat); export _POST_BODY
pkg_raw=$(post_param "package" "")
pkg_clean=$(sanitize_alnum "$pkg_raw")

if [ -z "$pkg_clean" ]; then
    echo '<p class="error">Недопустимое имя пакета</p>'
    exit 0
fi

echo "<h2>Установка пакета: $(html_escape "$pkg_clean")</h2>"
echo '<pre>'
if /opt/bin/opkg install "$pkg_clean" 2>&1; then
    echo '</pre><p class="success">Пакет успешно установлен.</p>'
    log_action "INFO" "Установлен пакет $pkg_clean"
    mkdir -p /tmp/entware/logs 2>/dev/null
    echo "$(date '+%Y-%m-%d %H:%M:%S') | install | $pkg_clean | success" >> /tmp/entware/logs/package_changes.log
else
    echo '</pre><p class="error">Ошибка при установке. Проверьте логи opkg.</p>'
    log_action "ERROR" "Ошибка установки пакета $pkg_clean"
    echo "$(date '+%Y-%m-%d %H:%M:%S') | install | $pkg_clean | error" >> /tmp/entware/logs/package_changes.log
fi
