#!/bin/sh
# ==============================================
# Entware Manager - ARP таблица
# Версия: 1.8 (добавлено имя хоста)
# Дата: 2026-04-04
# ==============================================

. /opt/web_entware/lib/common.sh

if [ "$REQUEST_METHOD" = "GET" ]; then
    echo "Content-type: application/json"
    echo ""
    
    ip neigh show 2>/dev/null | grep -v "^fe80" | grep -v "^::" > /tmp/.net_arp
    
    result=""
    while IFS= read -r line; do
        [ -z "$line" ] && continue
        
        ip_addr=$(echo "$line" | awk '{print $1}')
        [ -z "$ip_addr" ] && continue
        
        iface=$(echo "$line" | awk '{print $3}')
        mac=$(echo "$line" | awk '{print $5}')
        state=$(echo "$line" | awk '{print $NF}')
        
        name=$(getent hosts "$ip_addr" 2>/dev/null | awk '{print $2}')
        [ -z "$name" ] && name=""
        
        [ -z "$mac" ] || [ "$mac" = "used" ] || [ "$mac" = "ref" ] && mac="--"
        [ -z "$state" ] && state="UNKNOWN"
        [ -z "$iface" ] && iface="--"
        
        if [ -n "$result" ]; then
            result="$result,"
        fi
        
        result="$result{\"ip\":\"$ip_addr\",\"mac\":\"$mac\",\"interface\":\"$iface\",\"state\":\"$state\",\"name\":\"$name\"}"
    done < /tmp/.net_arp
    
    rm -f /tmp/.net_arp
    
    echo "{\"entries\":[$result]}"
    exit 0
fi

json_out '{"error":"Method not allowed"}'
