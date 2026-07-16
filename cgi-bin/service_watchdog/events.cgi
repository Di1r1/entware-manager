#!/bin/sh
# ==============================================
# Entware Manager - события демона мониторинга служб
# Версия: 2.5 (использует parse_log_events из common.sh)
# ==============================================

export PATH=/opt/bin:/bin:/usr/bin:/sbin:/usr/sbin:/opt/sbin:/usr/sbin

. /opt/web_entware/lib/common.sh

QUERY_STRING="${QUERY_STRING:-}"
LIMIT=$(echo "$QUERY_STRING" | sed -n 's/.*limit=\([0-9]*\).*/\1/p')
[ -z "$LIMIT" ] && LIMIT=20

json_out "$(parse_log_events "service" "$LIMIT")"
