#!/bin/sh
# ==============================================
# Entware Manager - сетевой статус для карточки статистики
# Версия: 1.9 (исправлен парсинг brctl и трафик WiFi)
# Дата: 2026-04-05
# ==============================================

. /opt/web_entware/lib/common.sh

get_interfaces_with_ips() {
    local result=""
    local lan_ips=""
    
    while IFS= read -r line; do
        iface=$(echo "$line" | awk '{print $NF}')
        ip=$(echo "$line" | awk '{print $2}' | cut -d'/' -f1)
        
        [ "$ip" = "127.0.0.1" ] && continue
        
        if [ -n "$result" ]; then
            result="$result,"
        fi
        result="$result{\"iface\":\"$iface\",\"ip\":\"$ip\"}"
        
        if [ -n "$lan_ips" ]; then
            lan_ips="$lan_ips, $ip"
        else
            lan_ips="$ip"
        fi
    done <<EOF
$(ip -4 addr show 2>/dev/null | grep 'inet ')
EOF
    
    [ -z "$result" ] && result="[{\"iface\":\"-\",\"ip\":\"--\"}]" || result="[$result]"
    [ -z "$lan_ips" ] && lan_ips="--"
    
    echo "$result|$lan_ips"
}

get_wifi_info() {
    local result=""
    local bridge=""
    local ifs=""
    
    if command -v brctl >/dev/null 2>&1; then
        while IFS= read -r line; do
            first_col=$(echo "$line" | awk '{print $1}')
            first_char=$(echo "$first_col" | cut -c1)
            
            if [ "$first_col" = "bridge" ] || [ "$first_col" = "" ]; then
                continue
            fi
            
            if [ "$first_char" = "b" ]; then
                if [ -n "$bridge" ] && [ -n "$ifs" ]; then
                    result="$result$(get_wifi_bridge_info "$bridge" "$ifs"),"
                fi
                bridge="$first_col"
                ifs=""
            fi
            
            remaining=$(echo "$line" | sed 's/^[ \t]*//')
            for word in $remaining; do
                case "$word" in
                    ra*|rai*) ifs="$ifs $word" ;;
                esac
            done
        done <<EOF
$(brctl show 2>/dev/null)
EOF
        
        if [ -n "$bridge" ] && [ -n "$ifs" ]; then
            result="$result$(get_wifi_bridge_info "$bridge" "$ifs"),"
        fi
    fi
    
    [ -z "$result" ] && result="[{\"name\":\"--\",\"2g\":\"--\",\"5g\":\"--\",\"rx\":\"--\",\"tx\":\"--\"}]" || result="[${result%,}]"
    echo "$result"
}

get_wifi_bridge_info() {
    local bridge="$1"
    local ifs="$2"
    local wifi_2g=""
    local wifi_5g=""
    local max_rx=0
    local max_tx=0
    
    for iface in $ifs; do
        case "$iface" in
            ra*) 
                if [ -z "$wifi_2g" ]; then
                    wifi_2g="$iface"
                fi
                ;;
            rai*) 
                if [ -z "$wifi_5g" ]; then
                    wifi_5g="$iface"
                fi
                ;;
        esac
    done
    
    local dev_data=$(cat "/proc/net/dev" 2>/dev/null)
    for iface in $wifi_2g $wifi_5g; do
        [ -z "$iface" ] && continue
        rx_bytes=$(echo "$dev_data" | grep " $iface:" | head -1 | awk '{print $2}')
        tx_bytes=$(echo "$dev_data" | grep " $iface:" | head -1 | awk '{print $10}')
        
        [ -z "$rx_bytes" ] && rx_bytes="0"
        [ -z "$tx_bytes" ] && tx_bytes="0"
        
        rx_val=$((rx_bytes / 1024 / 1024))
        tx_val=$((tx_bytes / 1024 / 1024))
        
        if [ "$rx_val" -gt "$max_rx" ]; then
            max_rx="$rx_val"
        fi
        if [ "$tx_val" -gt "$max_tx" ]; then
            max_tx="$tx_val"
        fi
    done
    
    if [ "$max_rx" -gt 1024 ]; then
        rx_val=$((max_rx / 1024))
        rx_unit="GB"
    else
        rx_val="$max_rx"
        rx_unit="MB"
    fi
    
    if [ "$max_tx" -gt 1024 ]; then
        tx_val=$((max_tx / 1024))
        tx_unit="GB"
    else
        tx_val="$max_tx"
        tx_unit="MB"
    fi
    
    [ -z "$wifi_2g" ] && wifi_2g="--"
    [ -z "$wifi_5g" ] && wifi_5g="--"
    
    case "$bridge" in
        br0) name="LAN" ;;
        br1) name="Guest" ;;
        br2) name="Guest2" ;;
        *) name="$bridge" ;;
    esac
    
    echo "{\"name\":\"$name\",\"2g\":\"$wifi_2g\",\"5g\":\"$wifi_5g\",\"rx\":\"$rx_val $rx_unit\",\"tx\":\"$tx_val $tx_unit\"}"
}

get_wifi_status() {
    local wifi_state="--"
    if ip link show 2>/dev/null | grep -E "br0|ra|rai" | grep -q "UP"; then
        wifi_state="подключено"
    fi
    echo "$wifi_state"
}

get_wan_status() {
    local wan="--"
    
    for iface in ppp0 ppoe0 eth2 eth2.1 wwan0; do
        if ip link show "$iface" 2>/dev/null | grep -q "UP"; then
            wan="$iface (up)"
            break
        fi
    done
    
    if [ "$wan" = "--" ]; then
        default_if=$(ip route show default 2>/dev/null | awk '{print $5}' | head -1)
        if [ -n "$default_if" ]; then
            if ip link show "$default_if" 2>/dev/null | grep -q "UP"; then
                wan="$default_if (up)"
            else
                wan="down"
            fi
        fi
    fi
    echo "$wan"
}

get_physical_ports() {
    local result=""
    for iface in /sys/class/net/eth*; do
        [ -d "$iface" ] || continue
        name=$(basename "$iface")
        case "$name" in
            *.*) continue ;;  # пропускаем VLAN (eth2.1, eth2.3)
        esac
        local carrier=$(cat "$iface/carrier" 2>/dev/null)
        local speed=""
        local state="—"
        if [ "$carrier" = "1" ]; then
            state="✓"
            speed=$(cat "$iface/speed" 2>/dev/null)
            if [ -z "$speed" ] || [ "$speed" = "0" ] || [ "$speed" = "-1" ]; then
                speed="—"
            else
                speed="${speed}Mbps"
            fi
        fi
        if [ -n "$result" ]; then
            result="$result,"
        fi
        result="$result{\"iface\":\"$name\",\"speed\":\"$speed\",\"carrier\":\"$state\"}"
    done
    [ -z "$result" ] && result="{\"iface\":\"—\",\"speed\":\"—\",\"carrier\":\"—\"}"
    echo "[$result]"
}

get_networks_status() {
    local result=""
    local bridge=""
    local members=""
    
    if command -v brctl >/dev/null 2>&1; then
        while IFS= read -r line; do
            first_col=$(echo "$line" | awk '{print $1}')
            first_char=$(echo "$first_col" | cut -c1)
            
            if [ "$first_col" = "bridge" ] || [ "$first_col" = "" ]; then
                continue
            fi
            
            if [ "$first_char" = "b" ]; then
                if [ -n "$bridge" ] && [ -n "$members" ]; then
                    case "$bridge" in
                        br0) name="LAN" ;;
                        br1) name="Guest" ;;
                        br2) name="Guest" ;;
                        *) name="$bridge" ;;
                    esac
                    members=$(echo "$members" | tr ' ' '\n' | grep -v '^$' | sort | tr '\n' ' ')
                    members=$(echo "$members" | sed 's/ $//')
                    result="$result{\"name\":\"$name\",\"bridge\":\"$bridge\",\"members\":\"$members\"},"
                fi
                bridge="$first_col"
                members=""
                continue
            fi
            
            remaining=$(echo "$line" | sed 's/^[ \t]*//')
            for word in $remaining; do
                members="$members $word"
            done
        done <<EOF
$(brctl show 2>/dev/null)
EOF
        
        if [ -n "$bridge" ] && [ -n "$members" ]; then
            case "$bridge" in
                br0) name="LAN" ;;
                br1) name="Guest" ;;
                br2) name="Guest" ;;
                *) name="$bridge" ;;
            esac
            members=$(echo "$members" | tr ' ' '\n' | grep -v '^$' | sort | tr '\n' ' ')
            members=$(echo "$members" | sed 's/ $//')
            result="$result{\"name\":\"$name\",\"bridge\":\"$bridge\",\"members\":\"$members\"},"
        fi
    fi
    
    # WAN
    local wan_iface="—"
    for iface in ppp0 ppoe0 wwan0; do
        if ip link show "$iface" 2>/dev/null | grep -q "UP"; then
            wan_iface="$iface"
            break
        fi
    done
    if [ "$wan_iface" = "—" ]; then
        default_if=$(ip route show default 2>/dev/null | awk '{print $5}' | head -1)
        [ -n "$default_if" ] && wan_iface="$default_if"
    fi
    result="$result{\"name\":\"WAN\",\"bridge\":\"$wan_iface\",\"members\":\"\"}"
    
    echo "[${result%,}]"
}

get_wifi_status() {
    local wifi_state="—"
    if ip link show 2>/dev/null | grep -E "br0|ra|rai" | grep -q "UP"; then
        wifi_state="подключено"
    fi
    echo "$wifi_state"
}

interfaces_data=$(get_interfaces_with_ips)
interfaces=$(echo "$interfaces_data" | cut -d'|' -f1)
lan=$(echo "$interfaces_data" | cut -d'|' -f2)
wifi=$(get_wifi_status)
wifi_info=$(get_wifi_info)
wan=$(get_wan_status)
ports=$(get_physical_ports)
networks=$(get_networks_status)

echo "Content-type: application/json"
echo ""
echo "{\"interfaces\":$interfaces,\"lan\":\"$lan\",\"wifi\":\"$wifi\",\"wifi_info\":$wifi_info,\"wan\":\"$wan\",\"ports\":$ports,\"networks\":$networks}"
