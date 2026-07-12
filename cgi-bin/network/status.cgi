#!/bin/sh
# ==============================================
# Entware Manager - статус демона сети
# Версия: 1.7 (с поддержкой state file, защита от \n)
# Дата: 2026-04-07
# ==============================================

export PATH=/opt/bin:/bin:/usr/bin:/sbin:/usr/sbin:/opt/sbin:/usr/sbin

. /opt/web_entware/lib/common.sh

PIDFILE="/tmp/entware/pid/network_watchdog.pid"
STATEFILE="/tmp/entware/pid/network_watchdog_state.json"

get_status() {
    running="false"
    pid=""
    uptime="0"
    last_check=""
    
    if [ -f "$PIDFILE" ]; then
        pid=$(cat "$PIDFILE" 2>/dev/null)
        if [ -n "$pid" ] && pid_is_alive "$pid"; then
            running="true"
            if [ -f "/proc/$pid/stat" ]; then
                start_time=$(cut -d' ' -f22 "/proc/$pid/stat" 2>/dev/null)
                if [ -n "$start_time" ] && [ "$start_time" != "0" ]; then
                    sys_time=$(cat /proc/uptime | awk '{print int($1)}')
                    uptime_sec=$((sys_time - start_time/100))
                    if [ "$uptime_sec" -gt 0 ]; then
                        minutes=$((uptime_sec / 60))
                        seconds=$((uptime_sec % 60))
                        uptime="${minutes}m${seconds}s"
                    fi
                fi
            fi
        fi
    fi
    
    if [ -f "$STATEFILE" ]; then
        # Читаем timestamp и удаляем возможные переводы строк
        last_check=$(/opt/bin/jq -r '.timestamp // ""' "$STATEFILE" 2>/dev/null | tr -d '\n\r')
        # Если timestamp пустой или содержит только пробелы
        if [ -z "$last_check" ] || [ "$last_check" = "null" ]; then
            last_check=""
        fi
    fi
    
    echo "Content-type: application/json"
    echo ""
    echo "{\"running\":$running,\"pid\":\"$pid\",\"uptime\":\"$uptime\",\"last_check\":\"$last_check\"}"
}

if [ "$REQUEST_METHOD" = "GET" ]; then
    get_status
    exit 0
fi

json_out '{"error":"Method not allowed"}'
