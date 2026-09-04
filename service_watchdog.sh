#!/bin/sh
# shellcheck disable=SC3043,SC3037,SC3057,SC1090,SC1091,SC2155,SC2046,SC2010,SC2086,SC2034
# Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
# ==============================================
# Entware Manager - демон мониторинга служб
# Версия: 1.13 (добавлена проверка cmdline для избежания ложных срабатываний)
# Дата: 2026-04-07
# ==============================================

export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
export HOME=/tmp

CONFIG="/opt/web_entware/service_config.json"
PIDFILE="/tmp/entware/pid/service_watchdog.pid"
PID_STATE="/tmp/entware/pid/service_watchdog_pids.json"
LOG_FILE="/tmp/entware/logs/service_events.log"
LOG_MAX_SIZE=1048576
SERVICES_DIR="/opt/etc/init.d"

. /opt/web_entware/lib/common.sh

load_config() {
    if [ -f "$CONFIG" ] && command -v jq >/dev/null 2>&1; then
        i=0
        while IFS= read -r _v; do
            i=$((i + 1))
            case "$i" in
                1) ENABLED=$_v ;;
                2) INTERVAL=$_v ;;
                3) MODE=$_v ;;
                4) WATCH_LIST=$_v ;;
                5) AUTO_RESTART=$_v ;;
                6) EXCLUDE_LIST=$_v ;;
                7) LOG_TO_MONITOR=$_v ;;
                8) HISTORY_DAYS=$_v ;;
                9) AUTOSTART=$_v ;;
            esac
        done << EOF
$(jq -r '.enabled // true, (.interval // 10), (.mode // "initd"), (if (.watch_list|type)=="array" then (.watch_list|join(" ")) else "lighttpd cron ttyd" end), (.auto_restart // false), (if (.exclude_list|type)=="array" then (.exclude_list|join(" ")) else "dropbear kvas-ws service_watchdog" end), (.log_to_monitor // true), (.pid_history_days // 7), (.autostart // false)' "$CONFIG")
EOF
    else
        ENABLED=true
        INTERVAL=10
        MODE="initd"
        WATCH_LIST="lighttpd cron ttyd"
        AUTO_RESTART=false
        EXCLUDE_LIST=""
        LOG_TO_MONITOR=true
        HISTORY_DAYS=7
        AUTOSTART=false
    fi

    [ -z "$INTERVAL" ] || [ "$INTERVAL" = "null" ] && INTERVAL=10
    [ -z "$MODE" ] || [ "$MODE" = "null" ] && MODE="initd"
    [ -z "$AUTO_RESTART" ] || [ "$AUTO_RESTART" = "null" ] && AUTO_RESTART=false
    [ -z "$EXCLUDE_LIST" ] || [ "$EXCLUDE_LIST" = "null" ] && EXCLUDE_LIST="dropbear kvas-ws service_watchdog"
    [ -z "$HISTORY_DAYS" ] || [ "$HISTORY_DAYS" = "null" ] && HISTORY_DAYS=7
    [ -z "$LOG_TO_MONITOR" ] || [ "$LOG_TO_MONITOR" = "null" ] && LOG_TO_MONITOR=true
    [ -z "$AUTOSTART" ] || [ "$AUTOSTART" = "null" ] && AUTOSTART=false
    [ "$ENABLED" = "null" ] && ENABLED=true
}

clean_old_pids() {
    if [ -f "$PID_STATE" ] && command -v jq >/dev/null 2>&1; then
        # Порог — полночь N дней назад (date_days_ago из lib/common.sh,
        # чистый POSIX без GNU date -d, которого может не быть в BusyBox)
        cutoff_day=$(date_days_ago "$HISTORY_DAYS" 2>/dev/null)
        if [ -z "$cutoff_day" ]; then
            # Fallback: не удалять старые записи
            return
        fi
        cutoff="$cutoff_day 00:00:00"

        temp_file="${PID_STATE}.tmp"
        jq --arg cutoff "$cutoff" 'to_entries | map(select(.value.last_seen > $cutoff)) | from_entries' "$PID_STATE" > "$temp_file" 2>/dev/null
        mv "$temp_file" "$PID_STATE" 2>/dev/null
    fi
}

get_service_pids() {
    service="$1"

    # Для lighttpd — pid-файл менеджера является единственным источником
    # истины, чтобы не путать с чужим lighttpd (например, веб-панель zapret
    # на 8088). Если PID из файла мёртв — возвращаем пусто (сервис упал),
    # а НЕ фолбэчимся на pgrep, который вернул бы чужой процесс.
    if [ "$service" = "lighttpd" ] && [ -f "/opt/var/run/lighttpd.pid" ]; then
        pid=$(cat "/opt/var/run/lighttpd.pid" 2>/dev/null | tr -d ' ')
        if [ -n "$pid" ] && [ -d "/proc/$pid" ]; then
            cmdline=$(cat "/proc/$pid/cmdline" 2>/dev/null | tr '\0' ' ')
            if echo "$cmdline" | grep -qi "lighttpd"; then
                state=$(cat "/proc/$pid/status" 2>/dev/null | grep "State:" | awk '{print $2}')
                case "$state" in
                    Z) ;;
                    *) echo "$pid"; return ;;
                esac
            fi
        fi
        return
    fi

    # pid-файла нет. Если на роутере есть наш S80entware-lighttpd (т.е.
    # живёт чужой lighttpd), то любой найденный pgrep-ом процесс — чужой,
    # а наш сервис не запущен: возвращаем пусто.
    if [ "$service" = "lighttpd" ] && [ -x "/opt/etc/init.d/S80entware-lighttpd" ]; then
        return
    fi

    # Маппинг: имя файла службы -> возможные имена процессов
    case "$service" in
        cron|crond)
            patterns="cron crond /usr/sbin/cron /opt/sbin/cron" ;;
        lighttpd)
            patterns="lighttpd lighttpd-entr /opt/sbin/lighttpd /usr/sbin/lighttpd" ;;
        ttyd)
            patterns="ttyd" ;;
        AdGuardHome|AdGuard|adguardhome)
            patterns="AdGuardHome ./AdGuardHome /opt/bin/AdGuardHome" ;;
        koolproxy)
            patterns="koolproxy koolproxynginx koolproxy" ;;
        xray|XRay|xkeen)
            patterns="xray xray-core xray" ;;
        transmission|transmission-daemon)
            patterns="transmission transmission-daemon /opt/bin/transmission-daemon" ;;
        samba|smbd)
            patterns="smbd nmbd /opt/sbin/smbd /opt/sbin/nmbd" ;;
        vsftpd|ftp)
            patterns="vsftpd /opt/sbin/vsftpd" ;;
        sshd)
            patterns="sshd /usr/sbin/smbd /opt/sbin/sshd" ;;
        dropbear)
            patterns="dropbear /opt/sbin/dropbear /usr/sbin/dropbear" ;;
        kvas-ws|kvas)
            patterns="kvas-ws kvas ws" ;;
        nfqws2|nfqws)
            patterns="nfqws2 nfqws" ;;
        shadowsocks|ss)
            patterns="ss-redir ss-local ss-server" ;;
        *)
            patterns="$service" ;;
    esac
    
    for pattern in $patterns; do
        pids=$(find_pids "$pattern" | head -1)
        if [ -n "$pids" ]; then
            # Проверить cmdline чтобы убедиться что это реальный процесс
            cmdline=$(cat /proc/$pids/cmdline 2>/dev/null | tr '\0' ' ')
            if echo "$cmdline" | grep -qi "$pattern"; then
                if [ -d "/proc/$pids" ]; then
                    state=$(cat /proc/$pids/status 2>/dev/null | grep "State:" | awk '{print $2}')
                    case "$state" in
                        Z) continue ;;  # zombie - не считаем
                        *) echo "$pids"; return ;;
                    esac
                fi
            fi
        fi
    done
    
    # Fallback: поиск по частичному совпадению с проверкой cmdline
    pids=$(find_pids "$service" | head -1)
    
    if [ -n "$pids" ] && [ -d "/proc/$pids" ]; then
        cmdline=$(cat /proc/$pids/cmdline 2>/dev/null | tr '\0' ' ')
        if echo "$cmdline" | grep -qi "$service"; then
            state=$(cat /proc/$pids/status 2>/dev/null | grep "State:" | awk '{print $2}')
            case "$state" in
                Z) ;;  # zombie
                *) echo "$pids" ;;
            esac
        fi
    fi
}

get_initd_services() {
    [ -d "$SERVICES_DIR" ] || return
    for script in "$SERVICES_DIR"/S*; do
        [ -f "$script" ] && [ -x "$script" ] || continue
        basename=$(basename "$script")
        name=$(echo "$basename" | sed 's/^S[0-9]*//;s/^K[0-9]*//')
        echo "$name"
    done
}

save_pid_state() {
    local service="$1"
    local pid="$2"

    mkdir -p "$(dirname "$PID_STATE")" 2>/dev/null

    if [ ! -f "$PID_STATE" ]; then
        echo "{}" > "$PID_STATE"
    fi

    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local temp_file="${PID_STATE}.tmp"

    if command -v jq >/dev/null 2>&1; then
        jq --arg service "$service" --arg pid "$pid" --arg ts "$timestamp" \
           '.[$service] = {"pid": $pid, "last_seen": $ts, "first_seen": (.[$service].first_seen // $ts)}' \
           "$PID_STATE" > "$temp_file" 2>/dev/null
        mv "$temp_file" "$PID_STATE"
    fi
}

get_saved_pid() {
    local service="$1"
    if [ -f "$PID_STATE" ] && command -v jq >/dev/null 2>&1; then
        jq -r ".\"$service\".pid // \"\" " "$PID_STATE" 2>/dev/null
    fi
}

check_service() {
    local service="$1"
    
    [ "$service" = "service_watchdog" ] && return
    [ "$service" = "entware-watchdogs" ] && return
    
    for excl in $EXCLUDE_LIST; do
        [ "$service" = "$excl" ] && return
    done
    
    local pids=$(get_service_pids "$service")
    local current_pid=$(echo "$pids" | awk '{print $1}')
    local saved_pid=$(get_saved_pid "$service")
    
    if [ -n "$current_pid" ]; then
        if [ -z "$saved_pid" ]; then
            log_message "INFO" "[service] $service: started (pid=$current_pid)"
            save_pid_state "$service" "$current_pid"
        elif [ "$current_pid" != "$saved_pid" ]; then
            if pid_is_alive "$current_pid"; then
                log_message "WARN" "[service] $service: restarted (old_pid=$saved_pid -> new_pid=$current_pid)"
                save_pid_state "$service" "$current_pid"
            fi
        else
            save_pid_state "$service" "$current_pid"
        fi
    else
        if [ -n "$saved_pid" ]; then
            log_message "ERROR" "[service] $service: stopped (old_pid=$saved_pid)"
            if [ "$AUTO_RESTART" = "true" ]; then
                local lock_file="/tmp/entware/pid/service_watchdog_${service}.lock"
                local now=$(date +%s)
                local cooldown=60
                local can_restart=true
                
                if [ -f "$lock_file" ]; then
                    local last_time
                    last_time=$(cat "$lock_file" 2>/dev/null)
                    if [ -n "$last_time" ] && [ $((now - last_time)) -lt $cooldown ]; then
                        can_restart=false
                    fi
                fi
                
                if [ "$can_restart" = "true" ]; then
                    local init_script
                    # Для lighttpd используем наш init-скрипт (pid-файл), чтобы не убивать
                    # чужой lighttpd (zapret на 8088) через killall штатного S80lighttpd.
                    if [ "$service" = "lighttpd" ] && [ -x "/opt/etc/init.d/S80entware-lighttpd" ]; then
                        init_script="/opt/etc/init.d/S80entware-lighttpd"
                    else
                        init_script=$(ls /opt/etc/init.d/S* 2>/dev/null | grep -iE "/S[0-9]+${service}" | head -1)
                    fi
                    if [ -n "$init_script" ] && [ -x "$init_script" ]; then
                        echo "$now" > "$lock_file"
                        log_message "INFO" "[service] $service: auto_restart attempting..."
                        $init_script restart >/dev/null 2>&1
                        sleep 2
                        local new_pid
                        new_pid=$(get_service_pids "$service" | awk '{print $1}')
                        if [ -n "$new_pid" ]; then
                            log_message "INFO" "[service] $service: auto_restart_ok (new_pid=$new_pid)"
                            save_pid_state "$service" "$new_pid"
                        else
                            log_message "ERROR" "[service] $service: auto_restart_failed"
                        fi
                    else
                        log_message "ERROR" "[service] $service: auto_restart_failed (script not found)"
                    fi
                else
                    log_message "INFO" "[service] $service: auto_restart skipped (cooldown 60s)"
                fi
            fi
            if command -v jq >/dev/null 2>&1 && [ -f "$PID_STATE" ]; then
                local temp_file="${PID_STATE}.tmp"
                jq "del(.\"$service\")" "$PID_STATE" > "$temp_file" 2>/dev/null
                mv "$temp_file" "$PID_STATE"
            fi
        fi
    fi
}

daemon_loop() {
    load_config
    clean_old_pids

    mkdir -p "$(dirname "$LOG_FILE")" "$(dirname "$PIDFILE")" 2>/dev/null

    log_message "INFO" "[service] watchdog started (interval=${INTERVAL}s, mode=$MODE, auto_restart=$AUTO_RESTART, exclude=$EXCLUDE_LIST)"

    trap 'log_message "INFO" "[service] watchdog stopped (pid=$$)"; rm -f "$PIDFILE"; exit 0' TERM
    trap 'log_message "INFO" "[service] watchdog config_reload_triggered (pid=$$)"; sleep 1; exec "$0" daemon' HUP

    while true; do
        case "$MODE" in
            initd)
                for service in $(get_initd_services); do
                    check_service "$service"
                done
                ;;
            custom)
                for service in $WATCH_LIST; do
                    [ -n "$service" ] && check_service "$service"
                done
                ;;
        esac

        sleep "$INTERVAL"
    done
}

case "$1" in
    start)
        load_config
        [ "$ENABLED" = "true" ] || { echo "Service watchdog disabled in config"; exit 0; }
        daemon_start "service" "$PIDFILE" "$LOG_FILE" "(^|[/ ])service_watchdog\.sh daemon"
        ;;
    stop)
        daemon_stop "$PIDFILE" "(^|[/ ])service_watchdog\.sh daemon" 'rm -f "$PID_STATE"'
        log_message "INFO" "[service] Демон мониторинга служб остановлен"
        ;;
    restart)
        "$0" stop
        sleep 1
        load_config
        if [ "$ENABLED" = "true" ]; then
            "$0" start
        else
            echo "Disabled"
        fi
        ;;
    status)
        daemon_status "$PIDFILE" "(^|[/ ])service_watchdog\.sh daemon"
        ;;
    daemon)
        daemon_loop
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|daemon}"
        exit 1
        ;;
esac