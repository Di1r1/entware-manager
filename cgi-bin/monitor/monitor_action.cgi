#!/bin/sh
# ==============================================
# Entware Manager - управление демоном защиты
# Версия: 2.0 (использует watchdog.sh start/stop)
# ==============================================

. /opt/web_entware/lib/common.sh
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin

DEMON_SCRIPT="/opt/web_entware/watchdog.sh"
LOG_FILE="/tmp/entware/logs/monitor.log"

log() {
    level="$1"
    message="$2"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [$level] [ACTION] $message" >> "$LOG_FILE"
}

method="$REQUEST_METHOD"
action=""

if [ "$method" = "POST" ]; then
    action=$(post_param "action" "")
elif [ "$method" = "GET" ]; then
    action=$(get_param "action" "")
fi

action=$(url_decode "$action")
action=$(echo "$action" | tr -d '\n\r')

case "$action" in
    start)
        log "INFO" "Запрос на ЗАПУСК демона"
        result=$($DEMON_SCRIPT start 2>&1)
        exit_code=$?
        if [ $exit_code -eq 0 ]; then
            sleep 1
            alive=$($DEMON_SCRIPT status 2>&1)
            if [ $? -eq 0 ]; then
                log "INFO" "Демон запущен: $alive"
                json_out '{"status":"ok","message":"'"$alive"'"}'
                log_action "INFO" "Демон защиты запущен: $alive"
            else
                log "ERROR" "Демон не запустился: $alive"
                json_out '{"status":"error","message":"Демон не запустился"}'
            fi
        else
            log "ERROR" "Не удалось запустить демон: $result"
            json_out '{"status":"error","message":"'"$result"'"}'
        fi
        ;;
    stop)
        log "INFO" "Запрос на ОСТАНОВКУ демона"
        $DEMON_SCRIPT stop >/dev/null 2>&1
        log "INFO" "Демон остановлен"
        json_out '{"status":"ok","message":"Защита остановлена"}'
        log_action "INFO" "Демон защиты остановлен"
        ;;
    restart)
        log "INFO" "Запрос на ПЕРЕЗАПУСК демона"
        result=$($DEMON_SCRIPT restart 2>&1)
        exit_code=$?
        if [ $exit_code -eq 0 ]; then
            log "INFO" "Демон перезапущен: $result"
            json_out '{"status":"ok","message":"Защита перезапущена"}'
            log_action "INFO" "Демон защиты перезапущен"
        else
            log "ERROR" "Не удалось перезапустить демон: $result"
            json_out '{"status":"error","message":"'"$result"'"}'
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
