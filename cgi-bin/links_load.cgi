#!/bin/sh
# ==============================================
# Entware Manager - загрузка ссылок
# Версия: 0.02 (использование common.sh)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

LINKS_FILE="/opt/web_entware/links.json"
ROUTER_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
[ -z "$ROUTER_IP" ] && ROUTER_IP="192.168.3.1"

if [ -f "$LINKS_FILE" ]; then
    json_out "$(cat "$LINKS_FILE")"
else
    json_out "$(cat <<JSONEOF
[
  {"name":"Роутер","url":"http://${ROUTER_IP}","icon":"router"},
  {"name":"Entware Manager","url":"http://${ROUTER_IP}:8087/entware-manager/","icon":"package"},
  {"name":"AdGuard Home","url":"http://${ROUTER_IP}:3000","icon":"shield"},
  {"name":"Transmission","url":"http://${ROUTER_IP}:9091","icon":"download"},
  {"name":"Netdata","url":"http://${ROUTER_IP}:19999","icon":"chart"},
  {"name":"htop (ttyd)","url":"http://${ROUTER_IP}:8089","icon":"process"},
  {"name":"Терминал (ttyd)","url":"http://${ROUTER_IP}:9089","icon":"terminal"}
]
JSONEOF
)"
fi
