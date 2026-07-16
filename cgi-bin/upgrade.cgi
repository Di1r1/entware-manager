#!/bin/sh
# ==============================================
# Entware Manager - обновление пакета/всех пакетов
# Версия: 0.03 (добавлена поддержка upgrade all)
# Дата: 2026-04-07
# ==============================================

. /opt/web_entware/lib/common.sh

html_header

if [ "$REQUEST_METHOD" != "POST" ]; then
    echo '<p class="error">Ошибка: требуется POST-запрос</p>'
    exit 0
fi

upgrade_all=$(post_param "upgrade_all" "")
pkg_raw=$(post_param "package" "")

if [ "$upgrade_all" = "1" ]; then
    echo '<h2>Обновление всех пакетов...</h2>'
    echo '<pre>'
    if /opt/bin/opkg upgrade 2>&1; then
        echo '</pre><p class="success">Все пакеты обновлены.</p>'
        log_action "INFO" "Обновлены все пакеты"
    else
        echo '</pre><p class="error">Ошибка при обновлении.</p>'
        log_action "ERROR" "Ошибка обновления всех пакетов"
    fi
    exit 0
fi

pkg_clean=$(sanitize_alnum "$pkg_raw")

if [ -z "$pkg_clean" ]; then
    echo '<p class="error">Недопустимое имя пакета</p>'
    exit 0
fi

echo "<h2>Обновление пакета: $(html_escape "$pkg_clean")</h2>"
echo '<pre>'
if /opt/bin/opkg upgrade "$pkg_clean" 2>&1; then
    echo '</pre><p class="success">Пакет успешно обновлён.</p>'
    log_action "INFO" "Обновлён пакет $pkg_clean"
else
    echo '</pre><p class="error">Ошибка при обновлении. Проверьте логи opkg.</p>'
    log_action "ERROR" "Ошибка обновления пакета $pkg_clean"
fi
