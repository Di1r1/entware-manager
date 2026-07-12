#!/bin/sh
# ==============================================
# Возвращает температуру WiFi в формате JSON
# Версия: 0.04 (полные пути /opt/bin для CGI контекста)
# Дата: 2026-03-30
# ==============================================

. /opt/web_entware/lib/common.sh

get_temp() {
    iface=$1
    url="http://localhost:79/rci/show/interface/$iface"
    data=$(wget -qO- "$url" 2>/dev/null)
    if [ -z "$data" ]; then
        echo "null"
        return
    fi
    if /opt/bin/jq --version >/dev/null 2>&1; then
        temp=$(echo "$data" | /opt/bin/jq -r '.temperature // empty' 2>/dev/null)
        if [ -n "$temp" ] && [ "$temp" != "null" ]; then
            echo "$temp"
        else
            temp=$(echo "$data" | /opt/bin/jq -r '."температура" // empty' 2>/dev/null)
            echo "${temp:-null}"
        fi
    else
        temp=$(echo "$data" | grep -oE '"temperature":\s*[0-9]+' | head -1 | cut -d: -f2 | tr -d ' ')
        if [ -z "$temp" ]; then
            temp=$(echo "$data" | grep -oE '"температура":\s*[0-9]+' | head -1 | cut -d: -f2 | tr -d ' ')
        fi
        echo "${temp:-null}"
    fi
}

temp0=$(get_temp "WifiMaster0")
temp1=$(get_temp "WifiMaster1")

combined="null"
if [ "$temp0" != "null" ] || [ "$temp1" != "null" ]; then
    if [ "$temp0" = "null" ] && [ "$temp1" != "null" ]; then
        combined="WiFi1: ${temp1}C"
    elif [ "$temp1" = "null" ] && [ "$temp0" != "null" ]; then
        combined="WiFi0: ${temp0}C"
    elif [ "$temp0" != "null" ] && [ "$temp1" != "null" ]; then
        combined="WiFi0: ${temp0}C / WiFi1: ${temp1}C"
    fi
fi

json_out "{\"temp0\":$temp0,\"temp1\":$temp1,\"combined\":\"$combined\"}"
