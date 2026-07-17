#!/bin/sh
# ==============================================
# Entware Manager - список интерфейсов
# Версия: 2.3 (system ip)
# Дата: 2026-04-02
# ==============================================

. /opt/web_entware/lib/common.sh

if [ "$REQUEST_METHOD" = "GET" ]; then
    ip -o link show 2>/dev/null > /tmp/.net_ifaces
    
    result=""
    while IFS= read -r line; do
        name=$(echo "$line" | cut -d: -f2 | cut -d' ' -f2)
        [ -z "$name" ] && continue
        
        flags=$(echo "$line" | cut -d: -f3)
        
        state="UNKNOWN"
        echo "$flags" | grep -q "UP" && state="UP"
        echo "$flags" | grep -q "DOWN" && state="DOWN"
        
        ip_addr="--"
        if ip -4 addr show "$name" 2>/dev/null | grep -q "inet "; then
            ip_addr=$(ip -4 addr show "$name" 2>/dev/null | grep "inet " | head -1 | awk '{print $2}' | cut -d/ -f1)
        fi
        
        if [ -n "$result" ]; then
            result="$result,"
        fi
        result="$result{\"name\":\"$name\",\"state\":\"$state\",\"ip\":\"$ip_addr\"}"
    done < /tmp/.net_ifaces
    
    rm -f /tmp/.net_ifaces
    
    json_out "{\"interfaces\":[$result]}"
fi

json_out '{"error":"Method not allowed"}'
