#!/bin/sh
# ==============================================
# Entware Manager - чтение crontab
# Версия: 2.2 (оригинальная, рабочая)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

type=$(get_param "type" "")
type=$(url_decode "$type")

case "$type" in
    system)
        content=$(crontab -l 2>/dev/null)
        ;;
    opt|"")
        CRONTAB_FILE="/opt/etc/crontab"
        if [ -f "$CRONTAB_FILE" ]; then
            content=$(cat "$CRONTAB_FILE")
        else
            content=""
        fi
        ;;
    *)
        json_out '{"error":"Invalid type"}'
        ;;
esac

# Экранирование для JSON
content=$(echo "$content" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g; s/$/\\n/' | tr -d '\n')
json_out "{\"crontab\":\"$content\"}"
