#!/bin/sh
# ==============================================
# Entware Manager - сохранение ссылок
# Версия: 0.04 (полные пути /opt/bin для CGI контекста)
# Дата: 2026-03-31
# ==============================================

. /opt/web_entware/lib/common.sh

LINKS_FILE="/opt/web_entware/links.json"

if [ "$REQUEST_METHOD" != "POST" ]; then
    json_out '{"status":"error","message":"POST required"}'
fi

POST_DATA=$(cat)

if echo "$POST_DATA" | /opt/bin/jq empty 2>/dev/null; then
    echo "$POST_DATA" > "$LINKS_FILE"
    log_action "INFO" "Ссылки сохранены"
    json_out '{"status":"ok","message":"Links saved"}'
else
    json_out '{"status":"error","message":"Invalid JSON"}'
fi
