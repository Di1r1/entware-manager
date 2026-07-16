#!/bin/sh
# ==============================================
# Entware Manager - конфигурация защиты
# Версия: 1.5 (полные пути /opt/bin для CGI контекста)
# Дата: 2026-03-31
# ==============================================

. /opt/web_entware/lib/common.sh

CONFIG_FILE="/opt/web_entware/monitor_config.json"

if [ "$REQUEST_METHOD" = "GET" ]; then
    if [ -f "$CONFIG_FILE" ]; then
        if ! grep -q '"max_processes"' "$CONFIG_FILE" 2>/dev/null; then
            jq '. + {"max_processes": 200}' "$CONFIG_FILE" > "${CONFIG_FILE}.tmp" 2>/dev/null && mv "${CONFIG_FILE}.tmp" "$CONFIG_FILE"
        fi
        /opt/bin/cat "$CONFIG_FILE"
    else
        printf '%s\n' '{
  "enabled": false,
  "interval": 10,
  "individual": {
    "enabled": true,
    "threshold_cpu": 80,
    "threshold_time": 300
  },
  "ignore": ["lighttpd","cron","ttyd","watchdog"],
  "ignore_ps": true,
  "max_processes": 200,
  "log_file": "/tmp/entware/logs/monitor.log",
  "log_max_size": 1048576
}'
    fi
    exit 0
fi

if [ "$REQUEST_METHOD" = "POST" ]; then
    POST_DATA=$(cat)
    if echo "$POST_DATA" | /opt/bin/jq empty 2>/dev/null; then
        echo "$POST_DATA" > "$CONFIG_FILE"
        pid=$(/opt/bin/cat "/tmp/entware/pid/watchdog.pid" 2>/dev/null)
        [ -n "$pid" ] && kill -HUP "$pid" 2>/dev/null
        log_action "INFO" "Сохранены настройки защиты"
        json_out '{"status":"ok"}'
    else
        json_out '{"status":"error","message":"Invalid JSON"}'
    fi
    exit 0
fi

json_out '{"error":"Method not allowed"}'