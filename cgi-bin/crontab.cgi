#!/bin/sh
# ==============================================
# Entware Manager - чтение crontab
# Версия: 2.2 (оригинальная, рабочая)
# Дата: 2026-03-28
# ==============================================

export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
export HOME=/tmp

echo "Content-type: application/json"
echo ""

type=$(echo "$QUERY_STRING" | sed -n 's/.*type=\([^&]*\).*/\1/p')
type=$(printf '%b' "$(echo "$type" | sed 's/+/ /g; s/%/\\x/g')")

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
        echo "{\"error\":\"Invalid type\"}"
        exit 0
        ;;
esac

# Экранирование для JSON
content=$(echo "$content" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g; s/$/\\n/' | tr -d '\n')
echo "{\"crontab\":\"$content\"}"
