#!/bin/sh
# ==============================================
# Логирование действий менеджера
# Версия: 1.7 (чистая версия без DEBUG)
# Дата: 2026-04-01
# ==============================================

CONFIG_FILE="/opt/web_entware/logger/config.json"
LOG_ENABLED="true"

if [ -f "$CONFIG_FILE" ] && /opt/bin/jq --version >/dev/null 2>&1; then
    ENABLED_VALUE=$(/opt/bin/jq -r '.enabled' "$CONFIG_FILE" 2>/dev/null)
    if [ "$ENABLED_VALUE" = "true" ]; then
        LOG_ENABLED="true"
    else
        LOG_ENABLED="false"
    fi
else
    LOG_ENABLED="true"
fi

log_action_original() {
    if [ "$LOG_ENABLED" != "true" ]; then
        return 0
    fi
    level="$1"
    message="$2"
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    ip="${REMOTE_ADDR:-localhost}"
    pid=$$
    script_name="${SCRIPT_NAME:-unknown}"
    script_base=$(basename "$script_name" 2>/dev/null | sed 's/\.cgi$//')
    LOG_DIR="/tmp/entware/logs"
    log_file="$LOG_DIR/$(date +%Y-%m-%d).log"
    mkdir -p "$LOG_DIR" 2>/dev/null
    echo "[$timestamp] [$level] [$ip] [$pid] [$script_base] $message" >> "$log_file"
}

# Обёртка для обратной совместимости
log_action() {
    log_action_original "$@"
}
