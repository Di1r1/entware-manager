#!/bin/sh
# ==============================================
# Entware Manager - обновление списков пакетов
# Версия: 0.03 (использование common.sh)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

html_header

echo '<h2>Обновление списков пакетов</h2>'
echo '<pre>'
if /opt/bin/opkg update 2>&1; then
    echo '</pre><p class="success">Списки пакетов успешно обновлены.</p>'
    log_action "INFO" "Списки пакетов успешно обновлены"
else
    echo '</pre><p class="error">Ошибка обновления списков пакетов</p>'
    log_action "ERROR" "Ошибка обновления списков пакетов"
fi
