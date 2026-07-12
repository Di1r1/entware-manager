#!/bin/sh
# ==============================================
# Entware Manager - обновление crontab
# Версия: 2.6 (использует log_action из common.sh)
# Дата: 2026-03-28
# ==============================================

export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
export HOME=/tmp

. /opt/web_entware/lib/common.sh

echo "Content-type: application/json"
echo ""

if [ "$REQUEST_METHOD" != "POST" ]; then
    echo '{"error":"POST required"}'
    exit 0
fi

POST_DATA=$(cat)

# Извлечение параметров
type=$(echo "$POST_DATA" | sed -n 's/.*type=\([^&]*\).*/\1/p' | tr -d '\r')
crontab_raw=$(echo "$POST_DATA" | sed -n 's/.*crontab=\([^&]*\).*/\1/p' | tr -d '\r')

# Декодирование URL
type=$(printf '%b' "$(echo "$type" | sed 's/+/ /g; s/%/\\x/g')")
crontab=$(printf '%b' "$(echo "$crontab_raw" | sed 's/+/ /g; s/%/\\x/g')")

case "$type" in
    system)
        tmpfile=$(mktemp)
        echo "$crontab" > "$tmpfile"
        if crontab "$tmpfile" 2>/dev/null; then
            rm -f "$tmpfile"
            log_action "INFO" "Сохранён crontab (system)"
            echo '{"status":"ok"}'
        else
            rm -f "$tmpfile"
            echo '{"status":"error","message":"Invalid crontab"}'
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
            echo '{"status":"ok"}'
        else
            echo '{"status":"error","message":"Failed to write file"}'
        fi
        ;;
    *)
        echo '{"status":"error","message":"Invalid type"}'
        exit 0
        ;;
esac
