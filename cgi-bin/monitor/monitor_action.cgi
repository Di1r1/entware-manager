#!/bin/sh
# ==============================================
# Entware Manager - управление демоном защиты
# Версия: 2.1 (единый JSON с pid)
# ==============================================

. /opt/web_entware/lib/common.sh
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin

DEMON_SCRIPT="/opt/web_entware/watchdog.sh"
PIDFILE="/tmp/entware/pid/watchdog.pid"
LOG_FILE="/tmp/entware/logs/monitor.log"

log() {
    level="$1"
    message="$2"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [$level] [ACTION] $message" >> "$LOG_FILE"
}

method="$REQUEST_METHOD"
action=""

if [ "$method" = "POST" ]; then
    _POST_BODY=$(cat); export _POST_BODY
    action=$(post_param "action" "")
elif [ "$method" = "GET" ]; then
    action=$(get_param "action" "")
fi

action=$(echo "$action" | tr -d '\n\r')

case "$action" in
    start)
        log "INFO" "Запрос на ЗАПУСК демона"
        $DEMON_SCRIPT start >/dev/null 2>&1
        exit_code=$?
        if [ $exit_code -eq 0 ]; then
            sleep 1
            pid=$(cat "$PIDFILE" 2>/dev/null)
            if [ -n "$pid" ] && pid_is_alive "$pid"; then
                log "INFO" "Демон запущен (PID: $pid)"
                json_out "{\"status\":\"ok\",\"message\":\"Демон запущен\",\"pid\":$pid}"
                log_action "INFO" "Демон защиты запущен (PID: $pid)"
            else
                log "ERROR" "Демон не запустился"
                json_out '{"status":"error","message":"Демон не запустился"}'
            fi
        else
            log "ERROR" "Не удалось запустить демон"
            json_out '{"status":"error","message":"Не удалось запустить демон"}'
        fi
        ;;
    stop)
        log "INFO" "Запрос на ОСТАНОВКУ демона"
        $DEMON_SCRIPT stop >/dev/null 2>&1
        log "INFO" "Демон остановлен"
        json_out '{"status":"ok","message":"Демон остановлен"}'
        log_action "INFO" "Демон защиты остановлен"
        ;;
    restart)
        log "INFO" "Запрос на ПЕРЕЗАПУСК демона"
        $DEMON_SCRIPT restart >/dev/null 2>&1
        exit_code=$?
        if [ $exit_code -eq 0 ]; then
            sleep 1
            pid=$(cat "$PIDFILE" 2>/dev/null)
            if [ -n "$pid" ] && pid_is_alive "$pid"; then
                log "INFO" "Демон перезапущен (PID: $pid)"
                json_out "{\"status\":\"ok\",\"message\":\"Демон перезапущен\",\"pid\":$pid}"
                log_action "INFO" "Демон защиты перезапущен (PID: $pid)"
            else
                log "ERROR" "Демон не перезапустился"
                json_out '{"status":"error","message":"Демон не перезапустился"}'
            fi
        else
            log "ERROR" "Не удалось перезапустить демон"
            json_out '{"status":"error","message":"Не удалось перезапустить демон"}'
        fi
        ;;
    kill)
        pid=$(echo "$POST_DATA" | sed -n 's/.*pid=\([^&]*\).*/\1/p')
        if [ -n "$pid" ] && pid_is_alive "$pid"; then
            kill -9 "$pid" 2>/dev/null
            log "INFO" "Убит процесс $pid по запросу пользователя"
            json_out '{"status":"ok","message":"Процесс убит"}'
        else
            log "WARN" "Попытка убить несуществующий процесс $pid"
            json_out '{"status":"error","message":"Процесс не найден"}'
        fi
        ;;
    clearlog)
        LOG_FILE_PATH=$(grep -o '"log_file"[[:space:]]*:[[:space:]]*"[^"]*"' "/opt/web_entware/monitor_config.json" | sed 's/.*"log_file"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
        [ -z "$LOG_FILE_PATH" ] && LOG_FILE_PATH="/tmp/entware/logs/monitor.log"
        > "$LOG_FILE_PATH"
        log "INFO" "Лог очищен"
        json_out '{"status":"ok","message":"Лог очищен"}'
        ;;
    *)
        log "ERROR" "Неизвестное действие: $action"
        json_out '{"status":"error","message":"Неизвестное действие"}'
        ;;
esac
