#!/bin/sh
# ==============================================
# Entware Manager - события сети
# Версия: 1.5 (исправлен парсинг)
# Дата: 2026-04-05
# ==============================================

export PATH=/opt/bin:/bin:/usr/bin:/sbin:/usr/sbin:/opt/sbin:/usr/sbin

LOG_FILE="/tmp/entware/logs/network_events.log"
QUERY_STRING="${QUERY_STRING:-}"
LIMIT=$(echo "$QUERY_STRING" | sed -n 's/.*limit=\([0-9]*\).*/\1/p')
[ -z "$LIMIT" ] && LIMIT=20

echo "Content-type: application/json"
echo ""

if [ ! -f "$LOG_FILE" ]; then
    echo '{"events":[]}'
    exit 0
fi

EVENTS=$(tail -n 1000 "$LOG_FILE" 2>/dev/null | grep '\[NETWORK\]' | tail -n "$LIMIT")

if [ -z "$EVENTS" ]; then
    echo '{"events":[]}'
    exit 0
fi

FIRST=1
RESULT=""

while IFS= read -r line; do
    ts=$(echo "$line" | cut -c1-19)
    lvl=$(echo "$line" | sed -n 's/.*\[\(INFO\|WARN\|ERROR\)\].*/\1/p')
    
    rest=$(echo "$line" | sed 's/.*\[NETWORK\] //')
    
    service=$(echo "$rest" | awk '{print $1}')
    
    rest_after_service=$(echo "$rest" | sed "s/^$service //")
    evt=$(echo "$rest_after_service" | sed 's/ .*//')
    
    dtl=$(echo "$rest_after_service" | sed "s/^$evt //" | tr -d '()')
    
    [ -z "$dtl" ] && dtl="-"
    [ -z "$evt" ] && evt="-"
    [ -z "$lvl" ] && lvl="INFO"
    [ -z "$service" ] && service="network"
    
    if [ "$FIRST" -eq 1 ]; then
        FIRST=0
    else
        RESULT="${RESULT},"
    fi
    
    RESULT="${RESULT}{\"timestamp\":\"${ts}\",\"level\":\"${lvl}\",\"service\":\"${service}\",\"event\":\"${evt}\",\"details\":\"${dtl}\"}"
done <<EOF
$EVENTS
EOF

echo "{\"events\":[${RESULT}]}"
