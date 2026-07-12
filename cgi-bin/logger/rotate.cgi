#!/bin/sh
# ==============================================
# Ручная ротация логов
# Версия: 1.1 (использование common.sh)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

if [ "$REQUEST_METHOD" != "POST" ]; then
    json_out '{"status":"error","message":"POST required"}'
fi

/opt/web_entware/logger/scripts/rotate.sh > /dev/null 2>&1
if [ $? -eq 0 ]; then
    json_out '{"status":"ok","message":"Ротация выполнена"}'
else
    json_out '{"status":"error","message":"Ошибка при ротации"}'
fi
