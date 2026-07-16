#!/bin/sh
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin

. /opt/web_entware/lib/common.sh

DAEMON_PID_FILE="/tmp/entware/pid/watchdog.pid"

if [ -f "$DAEMON_PID_FILE" ]; then
    pid=$(cat "$DAEMON_PID_FILE")
    if pid_is_alive "$pid"; then
        daemon_status="running"
        daemon_pid="$pid"
    else
        daemon_status="stopped"
        daemon_pid=""
        rm -f "$DAEMON_PID_FILE"
    fi
else
    daemon_status="stopped"
    daemon_pid=""
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
    "daemon_status": "$daemon_status",
    "daemon_pid": "$daemon_pid",
    "processes": $top5
}
JSON
)"
