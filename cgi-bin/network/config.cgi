#!/bin/sh
# ==============================================
# Entware Manager - конфигурация демона сети
# Версия: 1.0
# Дата: 2026-04-02
# ==============================================

. /opt/web_entware/lib/common.sh

CONFIG_FILE="/opt/web_entware/network_config.json"

get_config() {
    if [ -f "$CONFIG_FILE" ] && /opt/bin/jq --version >/dev/null 2>&1; then
        /opt/bin/jq '.' "$CONFIG_FILE" 2>/dev/null
    else
        /opt/bin/cat <<'DEFAULT'
{
  "enabled": true,
  "interval": 30,
  "watch_interfaces": ["eth0"],
  "watch_internet": true,
  "ping_host": "8.8.8.8",
  "ping_timeout": 5,
  "notify_on": ["interface_down", "no_internet", "ip_changed"]
}
DEFAULT
    fi
}

save_config() {
    local data="$1"
    mkdir -p "$(dirname "$CONFIG_FILE")" 2>/dev/null
    echo "$data" > "$CONFIG_FILE"
}

if [ "$REQUEST_METHOD" = "GET" ]; then
    json_out "$(get_config)"
fi

if [ "$REQUEST_METHOD" = "POST" ]; then
    POST_DATA=$(cat)
    
    if [ -z "$POST_DATA" ]; then
        json_out '{"status":"error","message":"Empty request"}'
    fi
    
    if ! echo "$POST_DATA" | /opt/bin/jq empty 2>/dev/null; then
        json_out '{"status":"error","message":"Invalid JSON"}'
    fi
    
    save_config "$POST_DATA"
    json_out '{"status":"ok"}'
fi

json_out '{"error":"Method not allowed"}'
