#!/bin/sh
# ==============================================
# Возвращает температуру CPU в формате JSON
# Версия: 0.02 (использование common.sh)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

TEMP=""
if [ -f /sys/class/thermal/thermal_zone0/temp ]; then
    TEMP=$(cat /sys/class/thermal/thermal_zone0/temp | sed 's/...$//')
elif [ -f /sys/class/thermal/thermal_zone*/temp ]; then
    TEMP=$(cat /sys/class/thermal/thermal_zone*/temp | head -1 | sed 's/...$//')
fi

if [ -n "$TEMP" ]; then
    json_out "{\"temperature\": $TEMP}"
else
    json_out "{\"temperature\": null}"
fi
