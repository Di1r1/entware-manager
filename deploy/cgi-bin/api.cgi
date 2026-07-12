#!/bin/sh
# ==============================================
# Entware Manager - информация о пакете (JSON)
# Версия: 0.05 (полные пути /opt/bin для CGI контекста)
# Дата: 2026-03-30
# ==============================================

. /opt/web_entware/lib/common.sh

if ! /opt/bin/jq --version >/dev/null 2>&1; then
    json_out '{"error":"jq not found"}'
    exit 0
fi

action=$(get_param "action" "")
if [ "$action" != "info" ]; then
    json_out '{"error":"Invalid action"}'
    exit 0
fi

pkg_raw=$(get_param "package" "")
pkg_clean=$(sanitize_alnum "$pkg_raw")
if [ -z "$pkg_clean" ]; then
    json_out '{"error":"Invalid package name"}'
    exit 0
fi

info=$(/opt/bin/opkg -f /opt/etc/opkg.conf info "$pkg_clean" 2>&1)

if [ -z "$info" ]; then
    json_out '{"error":"No information returned by opkg"}'
    exit 0
fi

installed_time_raw=$(echo "$info" | grep "^Installed-Time:" | cut -d' ' -f2)
if [ -n "$installed_time_raw" ]; then
    installed_date=$(date -d "@$installed_time_raw" "+%Y-%m-%d %H:%M:%S" 2>/dev/null)
    if [ -n "$installed_date" ]; then
        info="$info"$'\n'"Installed-Date: $installed_date"
    fi
fi

echo "$info" | /opt/bin/jq -R -s '{info: .}'
