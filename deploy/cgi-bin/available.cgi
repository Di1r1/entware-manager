#!/bin/sh
# ==============================================
# Entware Manager - список доступных пакетов (JSON API)
# Версия: 2.9 (возврат sed-экранирования)
# Дата: 2026-03-29
# ==============================================

. /opt/web_entware/lib/common.sh

json_out "$(/opt/bin/opkg list 2>/dev/null | sed \
    -e 's/\\/\\\\/g' \
    -e 's/"/\\"/g' \
    -e 's/\t/\\t/g' \
    -e 's/\r/\\r/g' \
    -e 's/\n/ /g' \
    | awk '
BEGIN {first=1; print "["}
{
    if (NF==0) next
    pos = index($0, " - ")
    if (pos==0) next
    pkg = substr($0, 1, pos-1)
    rest = substr($0, pos+3)
    pos2 = index(rest, " - ")
    if (pos2>0) {
        ver = substr(rest, 1, pos2-1)
        desc = substr(rest, pos2+3)
    } else {
        ver = rest
        desc = ""
    }
    if (!first) printf ",\n"
    first=0
    printf "  {\"package\":\"%s\", \"version\":\"%s\", \"description\":\"%s\"}", pkg, ver, desc
}
END {print "\n]"}')"
