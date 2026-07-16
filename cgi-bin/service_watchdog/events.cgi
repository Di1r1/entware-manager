#!/bin/sh
# ==============================================
# Entware Manager - события демона мониторинга служб
# Версия: 2.4 (исправлен парсинг - tr -d '()' удаляет все скобки)
# Дата: 2026-04-07
# ==============================================

export PATH=/opt/bin:/bin:/usr/bin:/sbin:/usr/sbin:/opt/sbin:/usr/sbin

. /opt/web_entware/lib/common.sh

LOG_FILE="/tmp/entware/logs/service_events.log"
QUERY_STRING="${QUERY_STRING:-}"
LIMIT=$(echo "$QUERY_STRING" | sed -n 's/.*limit=\([0-9]*\).*/\1/p')
[ -z "$LIMIT" ] && LIMIT=20

if [ ! -f "$LOG_FILE" ]; then
    json_out '{"events":[]}'
fi

EVENTS=$(tail -n 1000 "$LOG_FILE" 2>/dev/null | grep '\[SERVICE\]' | tail -n "$LIMIT")

if [ -z "$EVENTS" ]; then
    json_out '{"events":[]}'
fi

FIRST=1
RESULT=""

while IFS= read -r line; do
    ts=$(echo "$line" | cut -c1-19)
    lvl=$(echo "$line" | sed -n 's/.*\[\(INFO\|WARN\|ERROR\)\].*/\1/p')
    [ -z "$lvl" ] && lvl="INFO"
    
    rest=$(echo "$line" | sed 's/.*\[SERVICE\] //')
    
    svc=$(echo "$rest" | awk '{print $1}')
    [ -z "$svc" ] && svc="unknown"
    
    rest_after_svc=$(echo "$rest" | sed "s/^$svc //")
    
    evt=$(echo "$rest_after_svc" | awk '{print $1}')
    [ -z "$evt" ] && evt="unknown"
    
    dtl=$(echo "$rest_after_svc" | sed "s/^$evt //")
    dtl=$(echo "$dtl" | tr -d '()')
    [ -z "$dtl" ] && dtl="-"
    
    if [ "$FIRST" -eq 1 ]; then
        FIRST=0
    else
        RESULT="${RESULT},"
    fi
    
    RESULT="${RESULT}{\"timestamp\":\"${ts}\",\"level\":\"${lvl}\",\"service\":\"${svc}\",\"event\":\"${evt}\",\"details\":\"${dtl}\"}"
done <<EOF
$EVENTS
EOF

json_out "{\"events\":[${RESULT}]}"
