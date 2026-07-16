#!/bin/sh
# ==============================================
# Entware Manager - история температур WiFi
# Версия: 1.5 (устойчивость к битым данным)
# ==============================================

. /opt/web_entware/lib/common.sh

MAX_DAYS=7
BASE_DIR="/tmp/entware/temp_history"
WIFI_FILE="$BASE_DIR/wifi"
CLEANUP_MARKER="$BASE_DIR/.wifi_cleanup"

mkdir -p "$BASE_DIR" 2>/dev/null

save_wifi_temp() {
    temp0="$1"; temp1="$2"
    case "$temp0" in ''|*[!0-9]*) temp0="-" ;; esac
    case "$temp1" in ''|*[!0-9]*) temp1="-" ;; esac
    today=$(date +%Y-%m-%d)
    echo "$(date '+%H:%M:%S')|$temp0|$temp1" >> "$WIFI_FILE.$today"
}

cleanup_old() {
    [ -f "$CLEANUP_MARKER" ] && [ "$(cat "$CLEANUP_MARKER")" = "$(date +%Y-%m-%d)" ] && return
    find "$BASE_DIR" -type f -name "wifi.*" -mtime +$MAX_DAYS -delete 2>/dev/null
    date +%Y-%m-%d > "$CLEANUP_MARKER"
}

get_history() {
    for f in $(ls -1t "$BASE_DIR"/wifi.* 2>/dev/null); do
        [ -f "$f" ] || continue
        grep -E '^([0-9]{2}:){2}[0-9]{2}\|(-|[0-9]+)\|(-|[0-9]+)$' "$f" 2>/dev/null
    done
}

case "$(get_param "action" "current")" in
    save)
        cleanup_old
        save_wifi_temp "$(get_param "temp0" "-")" "$(get_param "temp1" "-")"
        json_out '{"status":"ok"}'
        ;;
    history|*)
        hist_data=$(get_history)
        echo "$hist_data" | awk -F'|' '
BEGIN {first=1; print "["}
{
    time=$1; t0=$2; t1=$3
    if (time == "") next
    if (t0 == "-" || t0 == "") t0_val = "null"; else t0_val = t0
    if (t1 == "-" || t1 == "") t1_val = "null"; else t1_val = t1
    if (!first) printf ","
    first=0
    printf "{\"time\":\"%s\",\"temp0\":%s,\"temp1\":%s}", time, t0_val, t1_val
}
END {print "]"}'
        ;;
esac
