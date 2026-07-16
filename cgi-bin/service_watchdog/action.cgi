#!/bin/sh
# ==============================================
# Entware Manager - управление демоном мониторинга служб
# Версия: 1.4 (проверка пути к демону, log_action для всех действий)
# Дата: 2026-04-06
# ==============================================

. /opt/web_entware/lib/common.sh

DEMON="/opt/web_entware/service_watchdog.sh"
PIDFILE="/tmp/entware/pid/service_watchdog.pid"
LOG_FILE="/tmp/entware/logs/service_events.log"

action=$(get_param "action" "")

case "$action" in
    start)
        if [ -x "$DEMON" ]; then
            if [ -f "$PIDFILE" ]; then
                pid=$(cat "$PIDFILE" 2>/dev/null)
                if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                    json_out "{\"status\":\"error\",\"message\":\"Демон уже запущен\",\"pid\":$pid}"
                fi
                rm -f "$PIDFILE"
            fi

            $DEMON start >> "$LOG_FILE" 2>&1
            sleep 1

            if [ -f "$PIDFILE" ]; then
                new_pid=$(cat "$PIDFILE")
                log_action "INFO" "Демон watchdog запущен вручную (PID: $new_pid)"
                json_out "{\"status\":\"ok\",\"message\":\"Демон запущен\",\"pid\":$new_pid}"
            else
                log_action "ERROR" "Не удалось запустить демон watchdog"
                json_out "{\"status\":\"error\",\"message\":\"Не удалось запустить демон\"}"
            fi
        else
            json_out "{\"status\":\"error\",\"message\":\"Демон не найден: $DEMON\"}"
        fi
        ;;
    stop)
        $DEMON stop >> "$LOG_FILE" 2>&1
        log_action "INFO" "Демон watchdog остановлен вручную"
        json_out "{\"status\":\"ok\",\"message\":\"Демон остановлен\"}"
        ;;
    restart)
        $DEMON restart >> "$LOG_FILE" 2>&1
        sleep 1

        if [ -f "$PIDFILE" ]; then
            new_pid=$(cat "$PIDFILE")
            log_action "INFO" "Демон watchdog перезапущен вручную (PID: $new_pid)"
            json_out "{\"status\":\"ok\",\"message\":\"Демон перезапущен\",\"pid\":$new_pid}"
        else
            log_action "ERROR" "Не удалось перезапустить демон watchdog"
            json_out "{\"status\":\"error\",\"message\":\"Не удалось перезапустить демон\"}"
        fi
        ;;
    *)
        json_out "{\"status\":\"error\",\"message\":\"Неизвестное действие: $action\"}"
        ;;
esac