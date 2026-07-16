#!/bin/sh
# ==============================================
# Entware Manager - управление демоном сети
# Версия: 1.7 (упрощён, использует network_watchdog.sh)
# Дата: 2026-04-05
# ==============================================

export PATH=/opt/bin:/bin:/usr/bin:/sbin:/usr/sbin:/opt/sbin:/usr/sbin

. /opt/web_entware/lib/common.sh

WATCHDOG="/opt/web_entware/network_watchdog.sh"
PIDFILE="/tmp/entware/pid/network_watchdog.pid"
LOG_FILE="/tmp/entware/logs/network_events.log"

action=$(get_param "action" "")

case "$action" in
    start)
        if [ -f "$PIDFILE" ]; then
            pid=$(cat "$PIDFILE" 2>/dev/null)
            if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                json_out "{\"status\":\"error\",\"message\":\"Демон уже запущен\",\"pid\":$pid}"
            fi
            rm -f "$PIDFILE"
        fi

        $WATCHDOG start >> "$LOG_FILE" 2>&1
        sleep 1

        if [ -f "$PIDFILE" ]; then
            new_pid=$(cat "$PIDFILE")
            json_out "{\"status\":\"ok\",\"message\":\"Демон запущен\",\"pid\":$new_pid}"
        else
            json_out "{\"status\":\"error\",\"message\":\"Не удалось запустить демон\"}"
        fi
        ;;
    stop)
        $WATCHDOG stop >> "$LOG_FILE" 2>&1
        json_out '{"status":"ok","message":"Демон остановлен"}'
        ;;
    restart)
        $WATCHDOG restart >> "$LOG_FILE" 2>&1
        sleep 1

        if [ -f "$PIDFILE" ]; then
            new_pid=$(cat "$PIDFILE")
            json_out "{\"status\":\"ok\",\"message\":\"Демон перезапущен\",\"pid\":$new_pid}"
        else
            json_out "{\"status\":\"error\",\"message\":\"Не удалось перезапустить демон\"}"
        fi
        ;;
    *)
        json_out "{\"status\":\"error\",\"message\":\"Неизвестное действие: $action\"}"
        ;;
esac
