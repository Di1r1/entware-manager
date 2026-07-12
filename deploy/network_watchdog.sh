#!/bin/sh
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

log_event() {
    level="$1"
    event="$2"
    details="$3"
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null
    mkdir -p /opt/var/log/entware 2>/dev/null
    
    if [ -f "$LOG_FILE" ]; then
        size=$(stat -c %s "$LOG_FILE" 2>/dev/null)
        if [ -n "$size" ] && [ "$size" -gt "$LOG_MAX_SIZE" ]; then
            ts=$(date +%Y%m%d_%H%M%S)
            mv "$LOG_FILE" "${LOG_FILE}.${ts}.old"
            touch "$LOG_FILE"
            cp "$LOG_FILE" /opt/var/log/entware/network_events.log 2>/dev/null
        fi
    fi
    
    echo "$timestamp [$level] [NETWORK] $event $details" >> "$LOG_FILE"
}

load_config() {
    if [ -f "$CONFIG_FILE" ] && command -v jq >/dev/null 2>&1; then
        INTERVAL=$(jq -r '.watchdog.interval // 30' "$CONFIG_FILE")
        PING_HOST=$(jq -r '.watchdog.ping_host // "8.8.8.8"' "$CONFIG_FILE")
        PING_TIMEOUT=$(jq -r '.watchdog.ping_timeout // 5' "$CONFIG_FILE")
    else
        INTERVAL=30
        PING_HOST="8.8.8.8"
        PING_TIMEOUT=5
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
            log_event "ERROR" "$iface" "interface_down (was UP)"
        fi
        
        if [ "$current_state" = "UP" ] && [ "$previous_state" = "DOWN" ]; then
            log_event "INFO" "$iface" "interface_up"
        fi
        
        local current_ip=$(get_interface_ip "$iface")
        local previous_ip=$(jq -r ".interfaces.\"$iface\".ip // \"\"" "$STATE_FILE" 2>/dev/null)
        
        if [ -n "$current_ip" ] && [ -n "$previous_ip" ] && [ "$current_ip" != "$previous_ip" ]; then
            log_event "INFO" "$iface" "ip_changed ($previous_ip -> $current_ip)"
        fi
    done
}

check_internet() {
    local ping_result=$(ping -c 1 -W "$PING_TIMEOUT" "$PING_HOST" 2>/dev/null)
    local ping_ok=$?
    
    local previous_internet=$(jq -r '.internet // true' "$STATE_FILE" 2>/dev/null)
    
    if [ $ping_ok -ne 0 ]; then
        if [ "$previous_internet" = "true" ]; then
            log_event "WARN" "internet" "no_internet (ping $PING_HOST)"
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
    
    log_event "INFO" "network_watchdog" "started (interval=${INTERVAL}s, ping=$PING_HOST, pid=$$)"
    
    trap 'log_event "INFO" "network_watchdog" "stopped (pid=$$)"; rm -f "$PID_FILE"; exit 0' TERM
    trap 'load_config; log_event "INFO" "network_watchdog" "config_reloaded (interval=$INTERVAL)"' HUP
    
    while true; do
        check_interfaces
        local internet=$(check_internet)
        save_state "$internet"
        
        sleep "$INTERVAL"
    done
}

mkdir -p "$(dirname "$PID_FILE")" 2>/dev/null

case "$1" in
    start)
        if [ -f "$PID_FILE" ]; then
            pid=$(cat "$PID_FILE" 2>/dev/null)
            if [ -n "$pid" ] && pid_is_alive "$pid"; then
                echo "Already running with PID $pid"
                exit 1
            fi
            rm -f "$PID_FILE"
        fi
        
        existing=$(find_pids "network_watchdog\.sh daemon" | head -1)
        if [ -n "$existing" ]; then
            echo "Already running with PID $existing"
            exit 1
        fi
        
        load_config
        mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null
        
        sh "$0" daemon >> "$LOG_FILE" 2>&1 &
        echo $! > "$PID_FILE"
        log_action "INFO" "Демон мониторинга сети запущен (PID: $(cat $PID_FILE))"
        ;;
    stop)
        for p in $(find_pids "network_watchdog\.sh daemon" 2>/dev/null); do
            kill -9 "$p" 2>/dev/null
        done
        rm -f "$PID_FILE"
        rm -f "$STATE_FILE"
        log_action "INFO" "Демон мониторинга сети остановлен"
        ;;
    restart)
        "$0" stop
        sleep 2
        for p in $(find_pids "network_watchdog\.sh daemon" 2>/dev/null); do
            kill -9 "$p" 2>/dev/null
        done
        rm -f "$PID_FILE"
        rm -f "$STATE_FILE"
        sleep 1
        
        load_config
        mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null
        
        sh "$0" daemon >> "$LOG_FILE" 2>&1 &
        echo $! > "$PID_FILE"
        log_action "INFO" "Демон мониторинга сети перезапущен (PID: $(cat $PID_FILE))"
        ;;
    status)
        pid=""
        if [ -f "$PID_FILE" ]; then
            pid=$(cat "$PID_FILE")
            if pid_is_alive "$pid"; then
                echo "Running with PID $pid"
                exit 0
            fi
        fi
        pid=$(find_pids "network_watchdog\.sh daemon" | head -1)
        if [ -n "$pid" ] && pid_is_alive "$pid"; then
            echo "Running with PID $pid"
            exit 0
        fi
        echo "Not running"
        exit 1
        ;;
    daemon)
        daemon_loop
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|daemon}"
        exit 1
        ;;
esac
