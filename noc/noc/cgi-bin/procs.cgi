#!/bin/sh
# procs.cgi — полный слепок процессов для NOC-ветки (experiment branch).
# GET → JSON {"cpu_count":3,"loadavg":"0.26 0.57 0.52","mem_total_kb":1015072,
#   "mem_avail_kb":317748,"mem_swap_kb":223624,
#   "processes":[{"pid":1,"pcpu":"0.0","time":"5d21h","command":"...","rss_kb":17408}]}.
# CPU% — двухточечный замер (/proc/stat + utime/stime, окно 1с), как monitor/status.cgi.
# BusyBox ash: только POSIX, без jq/grep -P/pgrep. set -eu запрещён для CGI.
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
PROC_ROOT=/proc
CLK_TCK=100
SLEEP_S=1

if [ "${REQUEST_METHOD:-GET}" != "GET" ]; then
    printf 'Status: 405 Method Not Allowed\r\nContent-type: application/json; charset=utf-8\r\n\r\n{"error":"method not allowed"}\n'
    exit 0
fi

# jesc: экранирование строки для JSON ( backslash, кавычка; CR/LF/TAB режем )
jesc() {
    printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' | tr -d '\n\r\t'
}

# fmt_dur: секунды → 5s / 5m22s / 15h53m / 5d21h (как formatElapsed в Go)
fmt_dur() {
    s=$1
    if [ "$s" -lt 60 ]; then
        printf '%ds' "$s"
    elif [ "$s" -lt 3600 ]; then
        printf '%dm%ds' $((s / 60)) $((s % 60))
    elif [ "$s" -lt 86400 ]; then
        printf '%dh%dm' $((s / 3600)) $(((s % 3600) / 60))
    else
        printf '%dd%dh' $((s / 86400)) $(((s % 86400) / 3600))
    fi
}

# total_cpu: сумма всех полей строки 'cpu ' из /proc/stat
total_cpu() {
    line=$(sed -n 's/^cpu //p' "$PROC_ROOT/stat" 2>/dev/null | head -n 1)
    t=0
    for f in $line; do
        case $f in
            ''|*[!0-9]*) ;;
            *) t=$((t + f)) ;;
        esac
    done
    echo "$t"
}

# snap_tick: pid → "ticks start" в $SNAP (utime+stime, starttime).
# Один awk вместо cat на каждый pid: свои воркеры не раздувают окно замера.
snap_tick() {
    awk '{
        f=FILENAME; sub(/\/[^\/]*$/,"",f); sub(/.*\//,"",f); pid=f+0; if(pid<=0) next;
        i=match($0,/\)[^)]*$/); if(i==0) next;
        rest=substr($0,i+1); n=split(rest,a," ");
        if(n<20) next;
        if(a[12]~/^[0-9]+$/ && a[13]~/^[0-9]+$/ && a[20]~/^[0-9]+$/)
            print pid, a[12]+a[13], a[20];
    }' "$PROC_ROOT"/[0-9]*/stat 2>/dev/null > "$SNAP"
}

SNAP1=/tmp/noc-snap1.$$
SNAP2=/tmp/noc-snap2.$$
JOINED=/tmp/noc-joined.$$
CPU1=/tmp/noc-cpu1.$$
CPU2=/tmp/noc-cpu2.$$
trap 'rm -f "$SNAP1" "$SNAP2" "$JOINED" "$CPU1" "$CPU2"' EXIT

UPTIME=$(cut -d' ' -f1 "$PROC_ROOT/uptime" 2>/dev/null | cut -d'.' -f1)
case $UPTIME in ''|*[!0-9]*) UPTIME=0 ;; esac

SNAP=$SNAP1
snap_tick
T1=$(total_cpu)
sed -n 's/^cpu //p' "$PROC_ROOT/stat" 2>/dev/null | head -n 1 > "$CPU1"
sleep "$SLEEP_S"
sed -n 's/^cpu //p' "$PROC_ROOT/stat" 2>/dev/null | head -n 1 > "$CPU2"
SNAP=$SNAP2
snap_tick
T2=$(total_cpu)
DT=$((T2 - T1))
if [ "$DT" -le 0 ]; then DT=1; fi

# один join вместо grep на каждый pid; pcpu считаем сразу здесь.
# self ($$) исключаем: наш замер не должен светиться в топе.
# shellcheck disable=SC2086
awk -v dt="$DT" -v self="$$" 'NR==FNR{t[$1]=$2; s[$1]=$3; next} ($1 in t)&&($1!=self){d=t[$1]-$2; if(d<0)d=0; printf "%s %.1f %s\n", $1, d*100/dt, s[$1]}' "$SNAP2" "$SNAP1" > "$JOINED"

printf 'Content-type: application/json; charset=utf-8\r\n\r\n'

CPU_COUNT=$(grep -c '^processor' "$PROC_ROOT/cpuinfo" 2>/dev/null)
case $CPU_COUNT in ''|*[!0-9]*|0) CPU_COUNT=0 ;; esac
LOADAVG=$(cut -d' ' -f1-3 "$PROC_ROOT/loadavg" 2>/dev/null)
MEM_T=$(sed -n 's/^MemTotal:[[:space:]]*\([0-9]*\).*/\1/p' "$PROC_ROOT/meminfo" 2>/dev/null | head -n 1)
MEM_A=$(sed -n 's/^MemAvailable:[[:space:]]*\([0-9]*\).*/\1/p' "$PROC_ROOT/meminfo" 2>/dev/null | head -n 1)
SWAP_T=$(sed -n 's/^SwapTotal:[[:space:]]*\([0-9]*\).*/\1/p' "$PROC_ROOT/meminfo" 2>/dev/null | head -n 1)
SWAP_F=$(sed -n 's/^SwapFree:[[:space:]]*\([0-9]*\).*/\1/p' "$PROC_ROOT/meminfo" 2>/dev/null | head -n 1)
case $MEM_T in ''|*[!0-9]*) MEM_T=0 ;; esac
case $MEM_A in ''|*[!0-9]*) MEM_A=0 ;; esac
case $SWAP_T in ''|*[!0-9]*) SWAP_T=0 ;; esac
case $SWAP_F in ''|*[!0-9]*) SWAP_F=$SWAP_T ;; esac
SWAP_U=$((SWAP_T - SWAP_F))
if [ "$SWAP_U" -lt 0 ]; then SWAP_U=0; fi

# разбивка CPU за то же окно: user(+nice), system, iowait, idle — в процентах
CPU_JSON=$(awk '{for(i=1;i<=NF;i++)t[NR]+= $i; u[NR]=$1+$2; s[NR]=$3; w[NR]=$5; d[NR]=$4} END{dt=t[2]-t[1]; if(dt<=0)dt=1; printf "{\"user\":%.1f,\"system\":%.1f,\"iowait\":%.1f,\"idle\":%.1f}", (u[2]-u[1])*100/dt, (s[2]-s[1])*100/dt, (w[2]-w[1])*100/dt, (d[2]-d[1])*100/dt}' "$CPU1" "$CPU2" 2>/dev/null)
case $CPU_JSON in '') CPU_JSON='{"user":0,"system":0,"iowait":0,"idle":100}' ;; esac

printf '{"cpu_count":%s,"loadavg":"%s","mem_total_kb":%s,"mem_avail_kb":%s,"mem_swap_kb":%s,"cpu":%s,"processes":[' \
    "$CPU_COUNT" "$LOADAVG" "$MEM_T" "$MEM_A" "$SWAP_U" "$CPU_JSON"
first=1
while read -r pid pcpu start; do
    cmd=$(tr '\0' ' ' < "$PROC_ROOT/$pid/cmdline" 2>/dev/null | sed -e 's/ *$//')
    cr=$(awk '/^Name:/{sub(/^Name:[ \t]+/,""); n=$0} /^VmRSS:/{for(i=1;i<=NF;i++) if($i~/^[0-9]+$/){r=$i; break}} END{print n "|" (r+0)}' "$PROC_ROOT/$pid/status" 2>/dev/null)
    comm=${cr%%|*}
    rss=${cr##*|}
    if [ -z "$cmd" ]; then cmd="[$comm]"; fi
    if [ "$cmd" = "[]" ]; then cmd="[pid $pid]"; fi
    case $rss in ''|*[!0-9]*) rss=0 ;; esac
    secs=$((UPTIME - start / CLK_TCK))
    if [ "$secs" -lt 0 ]; then secs=0; fi
    dur=$(fmt_dur "$secs")
    if [ "$first" -eq 1 ]; then first=0; else printf ','; fi
    printf '{"pid":%s,"pcpu":"%s","time":"%s","command":"%s","rss_kb":%s}' \
        "$pid" "$pcpu" "$dur" "$(jesc "$cmd")" "$rss"
done < "$JOINED"
printf ']}\n'
