#!/bin/sh
# ==============================================
# Entware Manager - SMART мониторинг дисков
# GET: ?action=list|info|attributes|health|selftest&device=/dev/sdX
# POST: action=selftest&device=/dev/sdX&type=short|long|conveyance
# ==============================================

. /opt/web_entware/lib/common.sh

action=$(get_param "action" "list")
device=$(get_param "device" "")

case "$action" in
    list)
        # Заглушка - возвращаем пустой массив
        json_out '{"disks":[]}'
        ;;
    info)
        json_out '{"status":"error","message":"Not implemented"}'
        ;;
    attributes)
        json_out '{"status":"error","message":"Not implemented"}'
        ;;
    health)
        json_out '{"status":"error","message":"Not implemented"}'
        ;;
    selftest)
        json_out '{"status":"error","message":"Not implemented"}'
        ;;
    *)
        json_out '{"status":"error","message":"Unknown action"}'
        ;;
esac