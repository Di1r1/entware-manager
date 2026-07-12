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

echo "Content-type: application/json"
echo ""

action=$(get_param "action" "")

case "$action" in
    start)
        if [ -x "$DEMON" ]; then
            if [ -f "$PIDFILE" ]; then
                pid=$(cat "$PIDFILE" 2>/dev/null)
                if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                    echo "{\"status\":\"error\",\"message\":\"Демон уже запущен (PID: $pid)\"}"
                    exit 0
                fi
                rm -f "$PIDFILE"
            fi

            $DEMON start >> "$LOG_FILE" 2>&1
            sleep 1

            if [ -f "$PIDFILE" ]; then
                new_pid=$(cat "$PIDFILE")
                log_action "INFO" "Демон watchdog запущен вручную (PID: $new_pid)"
                echo "{\"status\":\"ok\",\"message\":\"Демон запущен (PID: $new_pid)\"}"
            else
                log_action "ERROR" "Не удалось запустить демон watchdog"
                echo "{\"status\":\"error\",\"message\":\"Не удалось запустить демон\"}"
            fi
        else
            echo "{\"status\":\"error\",\"message\":\"Демон не найден: $DEMON\"}"
        fi
        ;;
    stop)
        $DEMON stop >> "$LOG_FILE" 2>&1
        log_action "INFO" "Демон watchdog остановлен вручную"
        echo "{\"status\":\"ok\",\"message\":\"Демон остановлен\"}"
        ;;
    restart)
        $DEMON restart >> "$LOG_FILE" 2>&1
        sleep 1

        if [ -f "$PIDFILE" ]; then
            new_pid=$(cat "$PIDFILE")
            log_action "INFO" "Демон watchdog перезапущен вручную (PID: $new_pid)"
            echo "{\"status\":\"ok\",\"message\":\"Демон перезапущен (PID: $new_pid)\"}"
        else
            log_action "ERROR" "Не удалось перезапустить демон watchdog"
            echo "{\"status\":\"error\",\"message\":\"Не удалось перезапустить демон\"}"
        fi
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Неизвестное действие: $action\"}"
        ;;
esac