#!/bin/sh
# ==============================================
# Entware Manager - обновление crontab
# Версия: 2.6 (использует log_action из common.sh)
# Дата: 2026-03-28
# ==============================================

export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
export HOME=/tmp

. /opt/web_entware/lib/common.sh

if [ "$REQUEST_METHOD" != "POST" ]; then
    json_out '{"error":"POST required"}'
fi

_POST_BODY=$(cat); export _POST_BODY

type=$(post_param "type" "")
crontab=$(post_param "crontab" "")

case "$type" in
    system)
        tmpfile=$(mktemp)
        echo "$crontab" > "$tmpfile"
        if crontab "$tmpfile" 2>/dev/null; then
            rm -f "$tmpfile"
            log_action "INFO" "Сохранён crontab (system)"
            json_out '{"status":"ok"}'
        else
            rm -f "$tmpfile"
            json_out '{"status":"error","message":"Invalid crontab"}'
        fi
        ;;
    opt|"")
        CRONTAB_FILE="/opt/etc/crontab"
        mkdir -p "$(dirname "$CRONTAB_FILE")"
        echo "$crontab" > "$CRONTAB_FILE"
        if [ $? -eq 0 ]; then
            pid=$(pgrep -f "cron" | grep -v grep | head -1)
            [ -n "$pid" ] && kill -HUP "$pid" 2>/dev/null
            log_action "INFO" "Сохранён crontab (opt)"
            json_out '{"status":"ok"}'
        else
            json_out '{"status":"error","message":"Failed to write file"}'
        fi
        ;;
    *)
        json_out '{"status":"error","message":"Invalid type"}'
        ;;
esac
