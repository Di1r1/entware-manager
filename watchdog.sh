#!/bin/sh
# shellcheck disable=SC2034
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
        i=0
        while IFS= read -r _v; do
            i=$((i + 1))
            case "$i" in
                1) ENABLED=$_v ;;
                2) INTERVAL=$_v ;;
                3) INDIVIDUAL_ENABLED=$_v ;;
                4) INDIVIDUAL_CPU=$_v ;;
                5) INDIVIDUAL_TIME=$_v ;;
                6) IGNORE_PS=$_v ;;
                7) MAX_PROCESSES=$_v ;;
                8) IGNORE_LIST=$_v ;;
            esac
        done << EOF
$(jq -r '.enabled // true, (.interval // 30), (.individual.enabled // true), (.individual.threshold_cpu // 20), (.individual.threshold_time // 120), (.ignore_ps // true), (.max_processes // 200), (if (.ignore|type)=="array" then (.ignore|join("|")) else "lighttpd|cron|ttyd|watchdog|ps|top" end)' "$CONFIG")
EOF
        if [ "$IGNORE_PS" != "false" ]; then
            IGNORE_PS="true"
        fi
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

    [ -z "$ENABLED" ] || [ "$ENABLED" = "null" ] && ENABLED=true
    [ -z "$INTERVAL" ] || [ "$INTERVAL" = "null" ] && INTERVAL=30
    [ -z "$INDIVIDUAL_ENABLED" ] || [ "$INDIVIDUAL_ENABLED" = "null" ] && INDIVIDUAL_ENABLED=true
    [ -z "$INDIVIDUAL_CPU" ] || [ "$INDIVIDUAL_CPU" = "null" ] && INDIVIDUAL_CPU=20
    [ -z "$INDIVIDUAL_TIME" ] || [ "$INDIVIDUAL_TIME" = "null" ] && INDIVIDUAL_TIME=120
    [ -z "$IGNORE_LIST" ] || [ "$IGNORE_LIST" = "null" ] && IGNORE_LIST="lighttpd|cron|ttyd|watchdog|ps|top"
}

kill_process() {
    pid=$1
    log_message "INFO" "[monitor] Убиваю процесс $pid"
    kill -9 "$pid" 2>/dev/null
}

daemon_loop() {
    read_config
    CPU_DIR="/tmp/entware/cpu_times"
    mkdir -p "$COUNTER_DIR" "$IGNORE_COUNTER_DIR" "$CPU_DIR" "$(dirname "$PIDFILE")" "$(dirname "$LOG_FILE")" 2>/dev/null

    log_message "INFO" "[monitor] Демон запущен (PID $$), ENABLED=$ENABLED, INTERVAL=$INTERVAL, CPU_THRESHOLD=$INDIVIDUAL_CPU, TIME_THRESHOLD=$INDIVIDUAL_TIME, IGNORE_PS=$IGNORE_PS, MAX_PROCESSES=$MAX_PROCESSES"

    rm -rf "${COUNTER_DIR:?}"/* "${IGNORE_COUNTER_DIR:?}"/* "${CPU_DIR:?}"/*

    trap 'log_message "INFO" "[monitor] Демон остановлен (PID $$)"; rm -f "$PIDFILE"; rm -rf "$COUNTER_DIR" "$IGNORE_COUNTER_DIR" "$CPU_DIR"; exit 0' TERM
    trap 'read_config; log_message "INFO" "[monitor] Конфигурация перечитана (CPU=$INDIVIDUAL_CPU TIME=$INDIVIDUAL_TIME IGNORE_PS=$IGNORE_PS MAX_PROCESSES=$MAX_PROCESSES)"' HUP

    # Сохраняем начальные CPU тики (utime+stime из /proc/pid/stat)
    # Поля после ')' в stat: state ppid pgrp session tty_nr tpgid flags minflt cminflt majflt cmajflt utime stime
    for pid_dir in /proc/[0-9]*/; do
        pid="${pid_dir%/}"; pid="${pid##*/}"
        read -r line 2>/dev/null < "/proc/$pid/stat" || continue
        after="${line#*)}"
        set -- $after
        [ $# -lt 13 ] && continue
        shift 11
        echo $(($1 + $2)) > "$CPU_DIR/$pid" 2>/dev/null
    done

    while true; do
        total_now=$(awk '/^cpu / {s=0; for(i=2;i<=9;i++) s+=$i; print s}' /proc/stat 2>/dev/null)
        [ -z "$total_now" ] && total_now=0

        for pid_dir in /proc/[0-9]*/; do
            pid="${pid_dir%/}"; pid="${pid##*/}"
            [ "$pid" = "$$" ] && continue

            read -r line 2>/dev/null < "/proc/$pid/stat" || continue

            comm="${line#*(}"
            comm="${comm%)*}"

            after="${line#*)}"
            set -- $after
            [ $# -lt 13 ] && continue
            ppid=$2
            shift 11
            utime=$1
            stime=$2

            current=$((utime + stime))
            prev=$(cat "$CPU_DIR/$pid" 2>/dev/null || echo "$current")
            delta=$((current - prev))
            echo "$current" > "$CPU_DIR/$pid" 2>/dev/null

            if [ "$total_now" -gt 0 ]; then
                cpu=$((delta * 100 / total_now))
            else
                cpu=0
            fi

            if [ "$cpu" -lt "$INDIVIDUAL_CPU" ]; then
                rm -f "$COUNTER_DIR/indiv_$pid" "$IGNORE_COUNTER_DIR/indiv_$pid" 2>/dev/null
                continue
            fi

            if [ "$IGNORE_PS" = "true" ]; then
                _cmd_base="${comm##*/}"
                case "$_cmd_base" in
                    ps|top) continue ;;
                esac
            fi

            is_ignore=0
            echo "$comm" | grep -qE "$IGNORE_LIST" && is_ignore=1

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
        done

        if [ -f "$LOG_FILE" ]; then
            size=$(wc -c < "$LOG_FILE" 2>/dev/null)
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
