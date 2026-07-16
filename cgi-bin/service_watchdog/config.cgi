#!/bin/sh
# ==============================================
# Entware Manager - конфигурация демона мониторинга служб
# Версия: 1.6 (исправлен парсинг auto_restart, дефолт true)
# Дата: 2026-04-07
# ==============================================

. /opt/web_entware/lib/common.sh

CONFIG="/opt/web_entware/service_config.json"
PIDFILE="/tmp/entware/pid/service_watchdog.pid"
LOG_FILE="/tmp/entware/logs/service_events.log"

# Дефолтные значения
DEFAULT_WATCH_LIST="\"lighttpd\",\"cron\",\"ttyd\",\"AdGuardHome\",\"koolproxy\",\"xray\""
DEFAULT_EXCLUDE="\"dropbear\",\"kvas-ws\",\"service_watchdog\""

method="$REQUEST_METHOD"

if [ "$method" = "GET" ] || [ -z "$method" ]; then
    if [ -f "$CONFIG" ] && command -v jq >/dev/null 2>&1; then
        json_out "$(cat "$CONFIG")"
    else
        json_out "{\"enabled\":true,\"interval\":10,\"mode\":\"initd\",\"watch_list\":[$DEFAULT_WATCH_LIST],\"auto_restart\":false,\"exclude_list\":[$DEFAULT_EXCLUDE],\"log_to_monitor\":true,\"pid_history_days\":7}"
    fi
elif [ "$method" = "POST" ]; then
    POST_DATA=$(cat)
    
    enabled=$(echo "$POST_DATA" | sed -n 's/.*"enabled": *\([^,}]*\).*/\1/p')
    interval=$(echo "$POST_DATA" | sed -n 's/.*"interval": *\([^,}]*\).*/\1/p')
    mode=$(echo "$POST_DATA" | sed -n 's/.*"mode": *"\([^"]*\)".*/\1/p')
    watch_list=$(echo "$POST_DATA" | sed -n 's/.*"watch_list": *\[//;s/\].*//;s/"//g;p')
    auto_restart=$(echo "$POST_DATA" | sed -n 's/.*"auto_restart": *\([a-z]*\).*/\1/p')
    exclude_list=$(echo "$POST_DATA" | tr -d '\n' | sed -n 's/.*"exclude_list": *\[\([^]]*\)\].*/\1/p; s/.*"exclude_list": *null.*/NULL/p')
    log_to_monitor=$(echo "$POST_DATA" | sed -n 's/.*"log_to_monitor": *\([a-z]*\).*/\1/p')
    pid_history_days=$(echo "$POST_DATA" | sed -n 's/.*"pid_history_days": *\([0-9]*\).*/\1/p')
    
    [ -z "$enabled" ] && enabled="true"
    [ -z "$interval" ] && interval="10"
    [ -z "$mode" ] && mode="initd"
    [ -z "$auto_restart" ] && auto_restart="true"
    [ -z "$log_to_monitor" ] && log_to_monitor="true"
    [ -z "$pid_history_days" ] && pid_history_days="7"
    
    if [ -z "$watch_list" ]; then
        watch_list="$DEFAULT_WATCH_LIST"
    else
        watch_list="\"$(echo "$watch_list" | sed 's/"//g; s/, */","/g')\""
    fi
    
    if [ "$exclude_list" = "NULL" ] || [ -z "$exclude_list" ]; then
        exclude_list="$DEFAULT_EXCLUDE"
    elif [ "$exclude_list" = "emptylist" ]; then
        exclude_list="[]"
    else
        exclude_list="\"$(echo "$exclude_list" | sed 's/"//g; s/, */","/g')\""
    fi
    
    TEMP_CONFIG=$(mktemp)
    cat > "$TEMP_CONFIG" << EOF
{
  "enabled": $enabled,
  "interval": $interval,
  "mode": "$mode",
  "watch_list": [$watch_list],
  "auto_restart": $auto_restart,
  "exclude_list": [$exclude_list],
  "log_to_monitor": $log_to_monitor,
  "pid_history_days": $pid_history_days
}
EOF
    
    if command -v jq >/dev/null 2>&1 && jq -c . "$TEMP_CONFIG" >/dev/null 2>&1; then
        mv "$TEMP_CONFIG" "$CONFIG"
        log_action "INFO" "Настройки watchdog: mode=$mode, auto_restart=$auto_restart, interval=${interval}s, exclude=[$exclude_list]"
    else
        rm -f "$TEMP_CONFIG"
        json_out '{"status":"error","message":"Invalid JSON configuration"}'
    fi
    
    if [ -f "$PIDFILE" ]; then
        pid=$(cat "$PIDFILE" 2>/dev/null)
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            kill -HUP "$pid" 2>/dev/null
            sleep 1
            json_out '{"status":"ok","message":"Конфигурация сохранена, демон перезагружен"}'
        else
            json_out '{"status":"ok","message":"Конфигурация сохранена (демон не запущен)"}'
        fi
    else
        json_out '{"status":"ok","message":"Конфигурация сохранена (демон не запущен)"}'
    fi
else
    json_out '{"status":"error","message":"Метод не поддерживается"}'
fi
