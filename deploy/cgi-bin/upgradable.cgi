#!/bin/sh
# ==============================================
# Entware Manager - список пакетов с обновлениями
# Версия: 0.13 (простой awk парсинг)
# Дата: 2026-04-07
# ==============================================

. /opt/web_entware/lib/common.sh

echo "Content-type: application/json"
echo ""

json=$(opkg list-upgradable 2>/dev/null | awk 'BEGIN { first=1 } {
    pkg=$1
    cur=$3
    new=$5
    if(pkg != "" && cur != "" && new != "" && new != "-") {
        if(first) first=0; else printf ","
        printf "{\"package\":\"%s\",\"current\":\"%s\",\"new\":\"%s\"}", pkg, cur, new
    }
} END { if(first) print "[]" }')

if [ -z "$json" ]; then
    printf "[]"
else
    printf "[%s]" "$json"
fi