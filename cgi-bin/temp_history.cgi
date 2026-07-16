#!/bin/sh
# ==============================================
# Entware Manager - история температур CPU
# Версия: 1.8 (надёжный awk парсинг)
# Дата: 2026-04-08
# ==============================================

. /opt/web_entware/lib/common.sh

MAX_DAYS=7
BASE_DIR="/tmp/entware/temp_history"
CPU_FILE="$BASE_DIR/cpu"
CLEANUP_MARKER="$BASE_DIR/.cpu_cleanup"

mkdir -p "$BASE_DIR" 2>/dev/null

save_temp() {
    temp="$1"
    case "$temp" in
        ''|*[!0-9]*) return ;;
    esac
    today=$(date +%Y-%m-%d)
    echo "$(date '+%H:%M:%S')|$temp" >> "$CPU_FILE.$today"
}

cleanup_old() {
    if [ -f "$CLEANUP_MARKER" ]; then
        last_clean=$(cat "$CLEANUP_MARKER")
        today=$(date +%Y-%m-%d)
        if [ "$last_clean" = "$today" ]; then
            return
        fi
    fi
    find "$BASE_DIR" -type f -name "cpu.*" -mtime +$MAX_DAYS -delete 2>/dev/null
    date +%Y-%m-%d > "$CLEANUP_MARKER"
}

get_history() {
    result=""
    for f in $(ls -1t "$BASE_DIR"/cpu.* 2>/dev/null); do
        [ -f "$f" ] || continue
        line=$(cat "$f" 2>/dev/null)
        [ -n "$line" ] && result="${result}${line}"$'\n'
    done
    printf "%s" "$result"
}

action=$(get_param "action" "history")

if [ "$action" = "save" ]; then
    cleanup_old
    temp=$(get_param "temp" "")
    save_temp "$temp"
    json_out '{"status":"ok"}'
fi

hist_data=$(get_history)

if [ -z "$hist_data" ]; then
    json_out '[]'
fi

result=$(echo "$hist_data" | awk '
BEGIN { first = 1; printf "[" }
{
    n = split($0, arr, "|")
    for (i = 1; i <= n; i += 2) {
        if (arr[i] != "" && arr[i+1] != "") {
            if (first) first = 0; else printf ","
            printf "{\"time\":\"%s\",\"temp\":%s}", arr[i], arr[i+1]
        }
    }
}
END { print "]" }')
json_out "$result"