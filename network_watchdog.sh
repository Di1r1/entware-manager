#!/bin/sh
# shellcheck disable=SC3043,SC3037,SC3057,SC1090,SC1091,SC2155,SC2046,SC2086,SC2034
# Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
# ==============================================
# Entware Manager - демон мониторинга сети
# Версия: 1.4 (ротация с timestamp, копирование в архив)
# Дата: 2026-04-05
# ==============================================

export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
export HOME=/tmp

. /opt/web_entware/lib/common.sh

CONFIG_FILE="/opt/web_entware/network_config.json"
PID_FILE="/tmp/entware/pid/network_watchdog.pid"
STATE_FILE="/tmp/entware/pid/network_watchdog_state.json"
LOG_FILE="/tmp/entware/logs/network_events.log"
LOG_MAX_SIZE=1048576

load_config() {
    if [ -f "$CONFIG_FILE" ] && command -v jq >/dev/null 2>&1; then
        i=0
        while IFS= read -r _v; do
            i=$((i + 1))
            case "$i" in
                1) INTERVAL=$_v ;;
                2) PING_HOST=$_v ;;
                3) PING_TIMEOUT=$_v ;;
                4) AUTOSTART=$_v ;;
            esac
        done << EOF
$(jq -r '.watchdog.interval // 30, .watchdog.ping_host // "8.8.8.8", .watchdog.ping_timeout // 5, .autostart // false' "$CONFIG_FILE")
EOF
    else
        INTERVAL=30
        PING_HOST="8.8.8.8"
        PING_TIMEOUT=5
        AUTOSTART=false
    fi
    
    [ -z "$INTERVAL" ] && INTERVAL=30
}

get_interface_state() {
    local iface="$1"
    ip link show "$iface" 2>/dev/null | grep -o "state [A-Z]*" | cut -d' ' -f2
}

get_interface_ip() {
    local iface="$1"
    ip -4 -o addr show "$iface" 2>/dev/null | awk '{print $4}' | cut -d'/' -f1 | head -1
}

check_interfaces() {
    local ifaces=$(jq -r '.watchdog.watch_interfaces // ["eth0"]' "$CONFIG_FILE" 2>/dev/null)
    
    for iface in $(echo "$ifaces" | jq -r '.[]' 2>/dev/null); do
        local current_state=$(get_interface_state "$iface")
        local previous_state=$(jq -r ".interfaces.\"$iface\".state // \"\"" "$STATE_FILE" 2>/dev/null)
        
        if [ "$current_state" = "DOWN" ] && [ "$previous_state" = "UP" ]; then
            log_message "ERROR" "[network] $iface: interface_down (was UP)"
        fi
        
        if [ "$current_state" = "UP" ] && [ "$previous_state" = "DOWN" ]; then
            log_message "INFO" "[network] $iface: interface_up"
        fi
        
        local current_ip=$(get_interface_ip "$iface")
        local previous_ip=$(jq -r ".interfaces.\"$iface\".ip // \"\"" "$STATE_FILE" 2>/dev/null)
        
        if [ -n "$current_ip" ] && [ -n "$previous_ip" ] && [ "$current_ip" != "$previous_ip" ]; then
            log_message "INFO" "[network] $iface: ip_changed ($previous_ip -> $current_ip)"
        fi
    done
}

check_internet() {
    local ping_result=$(ping -c 1 -W "$PING_TIMEOUT" "$PING_HOST" 2>/dev/null)
    local ping_ok=$?
    
    local previous_internet=$(jq -r '.internet // true' "$STATE_FILE" 2>/dev/null)
    
    if [ $ping_ok -ne 0 ]; then
        if [ "$previous_internet" = "true" ]; then
            log_message "WARN" "[network] internet: no_internet (ping $PING_HOST)"
        fi
        echo "false"
    else
        echo "true"
    fi
}

save_state() {
    local internet="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    local ifaces=$(jq -r '.watchdog.watch_interfaces // ["eth0"]' "$CONFIG_FILE" 2>/dev/null)
    local iface_states="{}"
    
    for iface in $(echo "$ifaces" | jq -r '.[]' 2>/dev/null); do
        local state=$(get_interface_state "$iface")
        local ip=$(get_interface_ip "$iface")
        iface_states=$(echo "$iface_states" | jq --arg i "$iface" --arg s "$state" --arg ip "$ip" '. + {($i): {state: $s, ip: $ip}}' 2>/dev/null)
    done
    
    echo "{\"timestamp\":\"$timestamp\",\"internet\":$internet,\"interfaces\":$iface_states}" > "$STATE_FILE"
}

daemon_loop() {
    load_config
    
    mkdir -p "$(dirname "$STATE_FILE")" "$(dirname "$LOG_FILE")" 2>/dev/null
    
    log_message "INFO" "[network] watchdog started (interval=${INTERVAL}s, ping=$PING_HOST)"
    
    trap 'log_message "INFO" "[network] watchdog stopped (pid=$$)"; rm -f "$PID_FILE"; exit 0' TERM
    trap 'load_config; log_message "INFO" "[network] watchdog config_reloaded (interval=$INTERVAL)"' HUP
    
    while true; do
        check_interfaces
        local internet=$(check_internet)
        save_state "$internet"
        
        sleep "$INTERVAL"
    done
}

case "$1" in
    start)
        daemon_start "network" "$PID_FILE" "$LOG_FILE" "(^|[/ ])network_watchdog\.sh daemon"
        ;;
    stop)
        daemon_stop "$PID_FILE" "(^|[/ ])network_watchdog\.sh daemon" 'rm -f "$STATE_FILE"'
        log_message "INFO" "[network] Демон мониторинга сети остановлен"
        ;;
    restart)
        "$0" stop
        sleep 1
        "$0" start
        ;;
    status)
        daemon_status "$PID_FILE" "(^|[/ ])network_watchdog\.sh daemon"
        ;;
    daemon)
        daemon_loop
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|daemon}"
        exit 1
        ;;
esac
