#!/bin/sh
# shellcheck disable=SC3043,SC3037,SC3057,SC1090,SC1091,SC2155,SC2046,SC2010,SC2086,SC2034
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
        ENABLED=$(jq -r '.enabled' "$CONFIG")
        INTERVAL=$(jq -r '.interval' "$CONFIG")
        MODE=$(jq -r '.mode' "$CONFIG")
        WATCH_LIST=$(jq -r '.watch_list[]' "$CONFIG" 2>/dev/null | tr '\n' ' ')
        AUTO_RESTART=$(jq -r '.auto_restart' "$CONFIG")
        EXCLUDE_LIST=$(jq -r '.exclude_list[]' "$CONFIG" 2>/dev/null | tr '\n' ' ')
        LOG_TO_MONITOR=$(jq -r '.log_to_monitor' "$CONFIG")
        HISTORY_DAYS=$(jq -r '.pid_history_days // 7' "$CONFIG")
    else
        ENABLED=true
        INTERVAL=10
        MODE="initd"
        WATCH_LIST="lighttpd cron ttyd"
        AUTO_RESTART=false
        EXCLUDE_LIST=""
        LOG_TO_MONITOR=true
        HISTORY_DAYS=7
    fi

    [ -z "$INTERVAL" ] && INTERVAL=10
    [ -z "$MODE" ] && MODE="initd"
    [ -z "$AUTO_RESTART" ] && AUTO_RESTART=false
    [ -z "$EXCLUDE_LIST" ] && EXCLUDE_LIST="dropbear kvas-ws service_watchdog"
    [ -z "$HISTORY_DAYS" ] && HISTORY_DAYS=7
    [ "$ENABLED" = "null" ] && ENABLED=true
}

clean_old_pids() {
    if [ -f "$PID_STATE" ] && command -v jq >/dev/null 2>&1; then
        # Используем совместимый способ расчёта даты
        if date -v -${HISTORY_DAYS}d +"%Y-%m-%d %H:%M:%S" >/dev/null 2>&1; then
            # BSD date (macOS)
            cutoff=$(date -v -${HISTORY_DAYS}d +"%Y-%m-%d %H:%M:%S" 2>/dev/null)
        elif date -d "-${HISTORY_DAYS} days" +"%Y-%m-%d %H:%M:%S" >/dev/null 2>&1; then
            # GNU date
            cutoff=$(date -d "-${HISTORY_DAYS} days" +"%Y-%m-%d %H:%M:%S" 2>/dev/null)
        else
            # Fallback: не удалять старые записи
            return
        fi
        
        if [ -n "$cutoff" ]; then
            temp_file="${PID_STATE}.tmp"
            jq --arg cutoff "$cutoff" 'to_entries | map(select(.value.last_seen > cutoff)) | from_entries' "$PID_STATE" > "$temp_file" 2>/dev/null
            mv "$temp_file" "$PID_STATE" 2>/dev/null
        fi
    fi
}

get_service_pids() {
    service="$1"
    
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
                    init_script=$(ls /opt/etc/init.d/S* 2>/dev/null | grep -iE "/S[0-9]+${service}" | head -1)
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

    log_message "INFO" "[service] watchdog started (interval=${INTERVAL}s, mode=$MODE, auto_restart=$AUTO_RESTART, exclude=$EXCLUDE_LIST, pid=$$)"

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
        daemon_start "service" "$PIDFILE" "$LOG_FILE" "service_watchdog\.sh daemon"
        ;;
    stop)
        daemon_stop "$PIDFILE" "service_watchdog\.sh daemon" 'rm -f "$PID_STATE"'
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
        daemon_status "$PIDFILE" "service_watchdog\.sh daemon"
        ;;
    daemon)
        daemon_loop
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|daemon}"
        exit 1
        ;;
esac