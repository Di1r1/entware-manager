#!/bin/sh
# ==============================================
# Entware Manager - SMART мониторинг дисков
# GET: ?action=list|info|attributes|health|selftest&device=/dev/sdX
# POST: action=selftest&device=/dev/sdX&type=short|long|conveyance
# ==============================================

. /opt/web_entware/lib/common.sh
. /opt/web_entware/lib/smart.sh

action=$(get_param "action" "list")
device_raw=$(get_param "device" "")
device=$(echo "$device_raw" | sed 's|^/dev/||')

case "$action" in
    list)
        disks_json=""
        first=1
        for d in $(smart_discover_disks); do
            [ "$first" -eq 1 ] && first=0 || disks_json="${disks_json},"
            disks_json="${disks_json}$(smart_disk_json "$d")"
        done
        json_out "{\"disks\":[${disks_json}]}"
        ;;
    info)
        [ -z "$device" ] && json_out '{"status":"error","message":"device required"}'
        json_out "$(smart_info_json "$device")"
        ;;
    attributes)
        [ -z "$device" ] && json_out '{"status":"error","message":"device required"}'
        json_out "$(smart_attributes_json "$device")"
        ;;
    health)
        [ -z "$device" ] && json_out '{"status":"error","message":"device required"}'
        json_out "$(smart_health_json "$device")"
        ;;
    selftest)
        [ -z "$device" ] && json_out '{"status":"error","message":"device required"}'
        if [ "$REQUEST_METHOD" = "POST" ]; then
            type=$(post_param "type" "short")
            json_out "$(smart_test_start "$device" "$type")"
        else
            json_out "$(smart_test_status "$device")"
        fi
        ;;
    *)
        json_out '{"status":"error","message":"Unknown action"}'
        ;;
esac
