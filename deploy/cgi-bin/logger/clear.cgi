#!/bin/sh
# ==============================================
# Очистка старых логов (старше 30 дней)
# Версия: 1.1 (использование common.sh)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

TARGET_DIR="/opt/var/log/entware"
if [ "$REQUEST_METHOD" = "POST" ]; then
    find "$TARGET_DIR" -name "*.log" -mtime +30 -delete 2>/dev/null
    json_out '{"status":"ok","message":"Логи старше 30 дней удалены"}'
else
    json_out '{"status":"error","message":"POST required"}'
fi
