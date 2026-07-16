#!/bin/sh
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin

. /opt/web_entware/lib/common.sh

DEMON_PID_FILE="/tmp/entware/pid/watchdog.pid"

if [ -f "$DEMON_PID_FILE" ]; then
    pid=$(cat "$DEMON_PID_FILE")
    if pid_is_alive "$pid"; then
        demon_status="running"
        demon_pid="$pid"
    else
        demon_status="stopped"
        demon_pid=""
        rm -f "$DEMON_PID_FILE"
    fi
else
    demon_status="stopped"
    demon_pid=""
fi

top5=$(top -bn1 2>/dev/null | sed -n '/^  PID/,$ p' | sed '1d' | head -5 | awk '
BEGIN { count=0; print "[" }
{
    if ($1 ~ /^[0-9]+$/ && $1 != "" && $1 != "PID") {
        count++
        if (count > 5) exit
        pid=$1; pcpu=$7;
        cmd=""; for(i=8;i<=NF;i++) cmd=cmd" "$i; gsub(/^ /,"",cmd); gsub(/"/,"\\\"",cmd);
        if (count>1) printf ",\n";
        printf "  {\"pid\":%d,\"pcpu\":\"%s\",\"time\":\"N/A\",\"command\":\"%s\"}", pid, pcpu, cmd
    }
}
END { print "\n]" }')
[ -z "$top5" ] && top5="[]"

json_out "$(cat <<JSON
{
    "demon_status": "$demon_status",
    "demon_pid": "$demon_pid",
    "processes": $top5
}
JSON
)"
