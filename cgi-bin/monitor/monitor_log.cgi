#!/bin/sh
# ==============================================
# Entware Manager - лог демона защиты
# Версия: 1.4 (читает из дневного лога через log_message)
# ==============================================

export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
echo "Content-type: text/plain; charset=utf-8"
echo ""

LOG_FILE="/tmp/entware/logs/$(date '+%Y-%m-%d').log"

if [ -f "$LOG_FILE" ]; then
    grep -i '\[monitor\]' "$LOG_FILE" | tail -n 200
else
    echo "Лог-файл не найден"
fi
