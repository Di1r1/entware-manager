#!/bin/sh
# ==============================================
# Entware Manager - загрузка ссылок
# Версия: 0.02 (использование common.sh)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

LINKS_FILE="/opt/web_entware/links.json"

if [ -f "$LINKS_FILE" ]; then
    cat "$LINKS_FILE"
else
    cat <<DEF
[
  {"name":"Роутер","url":"http://192.168.3.1","icon":"router"},
  {"name":"Entware Manager","url":"http://192.168.3.1:8087/entware-manager/","icon":"package"},
  {"name":"AdGuard Home","url":"http://192.168.3.1:3000","icon":"shield"},
  {"name":"Transmission","url":"http://192.168.3.1:9091","icon":"download"},
  {"name":"Netdata","url":"http://192.168.3.1:19999","icon":"chart"},
  {"name":"htop (ttyd)","url":"http://192.168.3.1:8089","icon":"process"},
  {"name":"Терминал (ttyd)","url":"http://192.168.3.1:9089","icon":"terminal"}
]
DEF
fi
