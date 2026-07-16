#!/bin/sh
# ==============================================
# Entware Manager - статус демона мониторинга служб
# Версия: 1.2 (добавлен exclude_list)
# Дата: 2026-04-06
# ==============================================

. /opt/web_entware/lib/common.sh

PIDFILE="/tmp/entware/pid/service_watchdog.pid"
CONFIG="/opt/web_entware/service_config.json"
PID_STATE="/tmp/entware/pid/service_watchdog_pids.json"

find_pid() {
    if [ -f "$PIDFILE" ]; then
        pid=$(cat "$PIDFILE" 2>/dev/null)
        if [ -n "$pid" ] && pid_is_alive "$pid"; then
            echo "$pid"
        fi
    fi
}

get_config() {
    if [ -f "$CONFIG" ] && command -v jq >/dev/null 2>&1; then
        jq -c '.' "$CONFIG" 2>/dev/null
    else
        echo '{"enabled":true,"interval":10,"mode":"initd","watch_list":[],"auto_restart":false,"exclude_list":["dropbear","kvas-ws","service_watchdog"]}'
    fi
}

get_pids() {
    if [ -f "$PID_STATE" ] && command -v jq >/dev/null 2>&1; then
        jq -c '.' "$PID_STATE" 2>/dev/null
    else
        echo '{}'
    fi
}

pid=$(find_pid)

if [ -n "$pid" ]; then
    config=$(get_config)
    pids=$(get_pids)
    json_out "{\"running\":true,\"pid\":$pid,\"config\":$config,\"pids\":$pids}"
else
    config=$(get_config)
    json_out "{\"running\":false,\"pid\":null,\"config\":$config,\"pids\":{}}"
fi
