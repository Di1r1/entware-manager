#!/bin/sh
# ==============================================
# Entware Manager - удаление пакета
# Версия: 0.05 (использование common.sh, убрана ссылка close)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

html_header

if [ "$REQUEST_METHOD" != "POST" ]; then
    echo '<p class="error">Ошибка: требуется POST-запрос</p>'
    exit 0
fi

pkg_raw=$(post_param "package" "")
pkg_clean=$(sanitize_alnum "$pkg_raw")

if [ -z "$pkg_clean" ]; then
    echo '<p class="error">Недопустимое имя пакета</p>'
    exit 0
fi

echo "<h2>Удаление пакета: $(echo "$pkg_clean" | sed 's/&/\&amp;/g')</h2>"
echo '<pre>'
if /opt/bin/opkg remove "$pkg_clean" 2>&1; then
    echo '</pre><p class="success">Пакет успешно удалён.</p>'
    log_action "INFO" "Удалён пакет $pkg_clean"
else
    echo '</pre><p class="error">Ошибка при удалении. Проверьте логи opkg.</p>'
    log_action "ERROR" "Ошибка удаления пакета $pkg_clean"
fi

# Ссылка "Закрыть окно" удалена – пользователь закрывает модальное окно крестиком
