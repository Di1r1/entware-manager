#!/bin/sh
# ==============================================
# Entware Manager - версия приложения
# Версия: 1.1 (чтение version.json)
# Дата: 2026-03-30
# ==============================================

. /opt/web_entware/lib/common.sh

if [ -f "/opt/web_entware/version.json" ]; then
    json_out "$(cat /opt/web_entware/version.json)"
else
    json_out '{"version":"unknown","date":"unknown"}'
fi
