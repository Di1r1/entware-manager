#!/bin/sh
# ==============================================
# Entware Manager - таблица маршрутизации
# Версия: 1.7 (system ip)
# Дата: 2026-04-02
# ==============================================

. /opt/web_entware/lib/common.sh

if [ "$REQUEST_METHOD" = "GET" ]; then
    echo "Content-type: application/json"
    echo ""
    
    ip route show 2>/dev/null > /tmp/.net_routes
    
    result=""
    while IFS= read -r line; do
        [ -z "$line" ] && continue
        
        dest=$(echo "$line" | awk '{print $1}')
        gateway="0.0.0.0"
        iface="--"
        metric=""
        
        prev=""
        for word in $line; do
            case "$word" in
                via) ;;
                dev) ;;
                metric) ;;
                *) 
                    [ "$prev" = "via" ] && gateway="$word"
                    [ "$prev" = "dev" ] && iface="$word"
                    [ "$prev" = "metric" ] && metric="$word"
                    ;;
            esac
            prev="$word"
        done
        
        if [ -n "$result" ]; then
            result="$result,"
        fi
        
        if [ -n "$metric" ]; then
            result="$result{\"destination\":\"$dest\",\"gateway\":\"$gateway\",\"interface\":\"$iface\",\"metric\":\"$metric\"}"
        else
            result="$result{\"destination\":\"$dest\",\"gateway\":\"$gateway\",\"interface\":\"$iface\"}"
        fi
    done < /tmp/.net_routes
    
    rm -f /tmp/.net_routes
    
    echo "{\"routes\":[$result]}"
    exit 0
fi

json_out '{"error":"Method not allowed"}'
