#!/bin/sh
# ==============================================
# Entware Manager - демон защиты от зависших процессов
# Версия: 3.0 (единый интерфейс start/stop/restart/status/daemon)
# ==============================================

export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin

. /opt/web_entware/lib/common.sh

CONFIG="/opt/web_entware/monitor_config.json"
COUNTER_DIR="/tmp/entware/counters"
IGNORE_COUNTER_DIR="/tmp/entware/counters_ignore"
PIDFILE="/tmp/entware/pid/watchdog.pid"
LOG_FILE="/tmp/entware/logs/monitor.log"
LOG_INTERVAL=30

read_config() {
    if [ -f "$CONFIG" ] && command -v jq >/dev/null 2>&1; then
        ENABLED=$(jq -r '.enabled' "$CONFIG")
        INTERVAL=$(jq -r '.interval' "$CONFIG")
        INDIVIDUAL_ENABLED=$(jq -r '.individual.enabled' "$CONFIG")
        INDIVIDUAL_CPU=$(jq -r '.individual.threshold_cpu' "$CONFIG")
        INDIVIDUAL_TIME=$(jq -r '.individual.threshold_time' "$CONFIG")
        IGNORE_LIST=$(jq -r '.ignore[]' "$CONFIG" | tr '\n' '|')
        IGNORE_LIST="${IGNORE_LIST%|}"
        IGNORE_PS=$(jq -r '.ignore_ps' "$CONFIG")
        if [ "$IGNORE_PS" != "false" ]; then
            IGNORE_PS="true"
        fi
        MAX_PROCESSES=$(jq -r '.max_processes // 200' "$CONFIG")
        case "$MAX_PROCESSES" in
            ''|*[!0-9]*) MAX_PROCESSES=200 ;;
            *) [ "$MAX_PROCESSES" -lt 1 ] && MAX_PROCESSES=200 ;;
        esac
    else
        ENABLED=true
        INTERVAL=30
        INDIVIDUAL_ENABLED=true
        INDIVIDUAL_CPU=20
        INDIVIDUAL_TIME=120
        IGNORE_LIST="lighttpd|cron|ttyd|watchdog|ps|top"
        IGNORE_PS=true
        log_message "WARN" "[monitor] jq не найден, использую настройки по умолчанию"
    fi
}

kill_process() {
    pid=$1
    log_message "INFO" "[monitor] Убиваю процесс $pid"
    kill -9 "$pid" 2>/dev/null
}

daemon_loop() {
    read_config
    mkdir -p "$COUNTER_DIR" "$IGNORE_COUNTER_DIR" "$(dirname "$PIDFILE")" "$(dirname "$LOG_FILE")" 2>/dev/null

    log_message "INFO" "[monitor] Демон запущен (PID $$), ENABLED=$ENABLED, INTERVAL=$INTERVAL, CPU_THRESHOLD=$INDIVIDUAL_CPU, TIME_THRESHOLD=$INDIVIDUAL_TIME, IGNORE_PS=$IGNORE_PS, MAX_PROCESSES=$MAX_PROCESSES"

    rm -rf "$COUNTER_DIR"/* "$IGNORE_COUNTER_DIR"/*

    trap 'log_message "INFO" "[monitor] Демон остановлен (PID $$)"; rm -f "$PIDFILE"; rm -rf "$COUNTER_DIR" "$IGNORE_COUNTER_DIR"; exit 0' TERM
    trap 'read_config; log_message "INFO" "[monitor] Конфигурация перечитана (CPU=$INDIVIDUAL_CPU TIME=$INDIVIDUAL_TIME IGNORE_PS=$IGNORE_PS MAX_PROCESSES=$MAX_PROCESSES)"' HUP

    while true; do
        top -bn1 2>/dev/null | sed -n '/^  PID/,$ p' | sed '1d' | head -$MAX_PROCESSES | while read pid ppid user stat vsz cpuid cpu comm; do
            [ "$pid" = "$$" ] && continue
            if [ "$IGNORE_PS" = "true" ]; then
                _cmd_base="${comm##*/}"
                case "$_cmd_base" in
                    ps|top) continue ;;
                esac
            fi

            is_ignore=0
            echo "$comm" | grep -qE "$IGNORE_LIST" && is_ignore=1

            cpu=$(echo "$cpu" | sed 's/,/./')
            cpu_int=$(printf "%.0f" "$cpu" 2>/dev/null || echo "$cpu")

            if [ "$cpu_int" -ge "$INDIVIDUAL_CPU" ]; then
                if [ "$is_ignore" -eq 1 ]; then
                    dir="$IGNORE_COUNTER_DIR"
                    type="ИГНОРИРУЕМЫЙ"
                else
                    dir="$COUNTER_DIR"
                    type="ОБЫЧНЫЙ"
                fi
                counter_file="$dir/indiv_$pid"
                if [ -f "$counter_file" ]; then
                    count=$(cat "$counter_file")
                    count=$((count + INTERVAL))
                else
                    count=$INTERVAL
                fi
                echo "$count" > "$counter_file"

                if [ $((count % LOG_INTERVAL)) -eq 0 ] && [ "$count" -lt "$INDIVIDUAL_TIME" ]; then
                    log_message "WARN" "[monitor] $type процесс $pid ($comm) CPU $cpu% держится уже $count сек"
                fi

                if [ "$count" -ge "$INDIVIDUAL_TIME" ]; then
                    if [ "$is_ignore" -eq 1 ]; then
                        log_message "WARN" "[monitor] ИГНОРИРУЕМЫЙ процесс $pid ($comm) CPU $cpu% держится уже $INDIVIDUAL_TIME сек (не убит)"
                        rm -f "$counter_file"
                    else
                        log_message "ERROR" "[monitor] Превышен порог: процесс $pid ($comm) CPU $cpu% в течение $INDIVIDUAL_TIME сек"
                        kill_process "$pid"
                        rm -f "$counter_file"
                    fi
                fi
            else
                rm -f "$COUNTER_DIR/indiv_$pid" "$IGNORE_COUNTER_DIR/indiv_$pid" 2>/dev/null
            fi
        done

        if [ -f "$LOG_FILE" ]; then
            size=$(stat -c %s "$LOG_FILE" 2>/dev/null)
            if [ -n "$size" ] && [ "$size" -gt 1048576 ]; then
                mv "$LOG_FILE" "${LOG_FILE}.old"
                touch "$LOG_FILE"
                log_message "INFO" "[monitor] Лог ротирован"
            fi
        fi

        sleep "$INTERVAL"
    done
}

case "$1" in
    start)
        read_config
        [ "$ENABLED" = "true" ] || { echo "Disabled in config"; exit 1; }
        daemon_start "monitor" "$PIDFILE" "$LOG_FILE" "watchdog\.sh daemon"
        ;;
    stop)
        daemon_stop "$PIDFILE" "watchdog\.sh daemon" 'rm -rf "$COUNTER_DIR"/* "$IGNORE_COUNTER_DIR"/*'
        log_message "INFO" "[monitor] Демон защиты остановлен"
        echo "Stopped"
        ;;
    restart)
        "$0" stop
        sleep 1
        "$0" start
        ;;
    status)
        daemon_status "$PIDFILE" "watchdog\.sh daemon"
        ;;
    daemon)
        daemon_loop
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|daemon}"
        exit 1
        ;;
esac
