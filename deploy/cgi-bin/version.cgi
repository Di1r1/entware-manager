#!/bin/sh
# ==============================================
# Entware Manager - получение версии
# Версия: 1.1 (чтение из version.json)
# Дата: 2026-03-30
# ==============================================

echo "Content-type: application/json"
echo ""

if [ -f /opt/web_entware/version.json ]; then
    cat /opt/web_entware/version.json
else
    echo '{"version":"unknown","date":"unknown"}'
fi
