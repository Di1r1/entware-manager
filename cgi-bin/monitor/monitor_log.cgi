#!/bin/sh
# ==============================================
# Entware Manager - лог демона защиты
# Версия: 1.3 (полные пути /opt/bin для CGI контекста)
# Дата: 2026-03-30
# ==============================================

export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
echo "Content-type: text/plain; charset=utf-8"
echo ""

LOG_FILE="/tmp/entware/logs/monitor.log"
CONFIG="/opt/web_entware/monitor_config.json"
if [ -f "$CONFIG" ] && /opt/bin/jq --version >/dev/null 2>&1; then
    LOG_FILE=$(/opt/bin/jq -r '.log_file' "$CONFIG" 2>/dev/null)
fi

if [ -f "$LOG_FILE" ]; then
    tail -n 200 "$LOG_FILE"
else
    echo "Лог-файл не найден"
fi
