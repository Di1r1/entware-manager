#!/bin/sh
# shellcheck disable=SC3043,SC3037,SC3057,SC1090,SC1091,SC2155,SC2046,SC2086,SC2034
# Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-commander
# ==============================================
# Entware Manager - демон мониторинга сети
# Версия: 1.5 (интернет восстановлен, Wi-Fi клиенты, слабый сигнал)
# Дата: 2026-09-09
# ==============================================

export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
export HOME=/tmp

. /opt/entware-commander/lib/common.sh

CONFIG_FILE="/opt/entware-commander/network_config.json"
PID_FILE="/tmp/entware-commander/pid/network_watchdog.pid"
STATE_FILE="/tmp/entware-commander/pid/network_watchdog_state.json"
WIFI_STATE_FILE="/tmp/entware-commander/pid/network_watchdog_wifi.json"
RCI_BASE="http://127.0.0.1:79"
LOG_FILE="/tmp/entware-commander/logs/network_events.log"
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
        if [ "$previous_internet" = "false" ]; then
            log_message "INFO" "[network] internet: restored (ping $PING_HOST)"
        fi
        echo "true"
    fi
}

# check_wifi — подключения/отключения клиентов и слабый сигнал по RCI Keenetic.
# Состояние (mac -> name/rssi/weak) — в WIFI_STATE_FILE, флаг weak шлётся один раз
# до восстановления уровня выше порога.
check_wifi() {
    local watch_wifi threshold hosts wifi_state cur_json new_state
    watch_wifi=$(jq -r '.watchdog.watch_wifi // true' "$CONFIG_FILE" 2>/dev/null)
    [ "$watch_wifi" = "true" ] || return 0
    command -v curl >/dev/null 2>&1 || return 0
    threshold=$(jq -r '.watchdog.wifi_weak_rssi // -75' "$CONFIG_FILE" 2>/dev/null)
    case "$threshold" in ''|*[!0-9-]*) threshold=-75 ;; esac

    hosts=$(curl -s -m 5 "$RCI_BASE/rci/show/ip/hotspot/host" 2>/dev/null)
    [ -n "$hosts" ] || return 0

    wifi_state="$WIFI_STATE_FILE"
    cur_json="$wifi_state.cur"
    new_state="$wifi_state.new"
    [ -f "$wifi_state" ] || echo '{}' > "$wifi_state"
    echo "$hosts" | jq -c '[.[] | select(type == "object") | select(((.ssid // "") != "") or ((.ap // "") != "")) | {mac: (.mac // ""), name: (.name // .hostname // "?"), ssid: (if ((.ssid // "") != "") then .ssid elif ((.ap // "") != "") then .ap else "?" end), rssi: (.rssi // 0)}] | map(select(.mac != ""))' 2>/dev/null > "$cur_json" || return 0

    local line mac name ssid rssi notified old_name old_mac
    # подключения + слабый сигнал
    for line in $(jq -r '.[] | @base64' "$cur_json" 2>/dev/null); do
        line=$(echo "$line" | base64 -d 2>/dev/null)
        mac=$(echo "$line" | jq -r '.mac' 2>/dev/null)
        [ -n "$mac" ] && [ "$mac" != "null" ] || continue
        name=$(echo "$line" | jq -r '.name' 2>/dev/null)
        ssid=$(echo "$line" | jq -r '.ssid' 2>/dev/null)
        rssi=$(echo "$line" | jq -r '.rssi' 2>/dev/null)
        case "$rssi" in ''|*[!0-9-]*) rssi=0 ;; esac
        if [ "$(jq -r --arg m "$mac" 'has($m)' "$wifi_state" 2>/dev/null)" != "true" ]; then
            log_message "INFO" "[network] wifi: client $name connected ($ssid)"
        fi
        if [ "$rssi" -le "$threshold" ] 2>/dev/null; then
            notified=$(jq -r --arg m "$mac" '.[$m].weak // false' "$wifi_state" 2>/dev/null)
            if [ "$notified" != "true" ]; then
                log_message "WARN" "[network] wifi: client $name weak signal ($rssi dBm)"
            fi
        fi
    done
    # отключения
    for old_mac in $(jq -r 'keys[]' "$wifi_state" 2>/dev/null); do
        if ! grep -qF "\"mac\":\"$old_mac\"" "$cur_json" 2>/dev/null; then
            old_name=$(jq -r --arg m "$old_mac" '.[$m].name // "?"' "$wifi_state" 2>/dev/null)
            log_message "INFO" "[network] wifi: client $old_name disconnected"
        fi
    done
    # новое состояние: weak=true пока уровень ниже порога
    jq -s --argjson thr "$threshold" '.[0] as $old | .[1] | map({key: .mac, value: {name: .name, rssi: .rssi, weak: (.rssi <= $thr)}}) | from_entries' "$wifi_state" "$cur_json" 2>/dev/null > "$new_state" \
        && mv "$new_state" "$wifi_state"
    rm -f "$cur_json"
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
        check_wifi
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
