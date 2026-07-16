#!/bin/sh
# ==============================================
# Entware Manager - статистика системы
# Версия: 0.26 (топ-3 процессов по памяти)
# Дата: 2026-03-29
# ==============================================

. /opt/web_entware/lib/common.sh

html_header

get_value() {
    value=$(eval "$1" 2>/dev/null)
    if [ -z "$value" ]; then
        echo "$2"
    else
        echo "$value"
    fi
}

get_percent_class() {
    percent=$1
    if [ "$percent" -gt 90 ]; then
        echo "critical"
    elif [ "$percent" -gt 70 ]; then
        echo "warning"
    else
        echo "normal"
    fi
}

KERNEL=$(get_value "uname -r" "н/д")
ARCH=$(get_value "uname -m" "н/д")
HOSTNAME=$(get_value "hostname" "н/д")
UPTIME=$(get_value "uptime | sed 's/.*up \([^,]*\),.*/\1/'" "н/д")

MODEL="н/д"
if [ -f /proc/device-tree/model ]; then
    MODEL=$(tr -d '\0' < /proc/device-tree/model 2>/dev/null)
elif [ -f /tmp/sysinfo/model ]; then
    MODEL=$(cat /tmp/sysinfo/model 2>/dev/null)
elif [ -f /etc/openwrt_release ]; then
    MODEL=$(grep "DISTRIB_DESCRIPTION" /etc/openwrt_release | cut -d"'" -f2)
fi

MEM_TOTAL=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}')
MEM_FREE=$(grep MemFree /proc/meminfo 2>/dev/null | awk '{print $2}')
if [ -n "$MEM_TOTAL" ] && [ -n "$MEM_FREE" ]; then
    MEM_USED=$((MEM_TOTAL - MEM_FREE))
    MEM_TOTAL_MB=$((MEM_TOTAL / 1024))
    MEM_USED_MB=$((MEM_USED / 1024))
    MEM_INFO="${MEM_USED_MB} MB / ${MEM_TOTAL_MB} MB"
    MEM_PERCENT=$((MEM_USED * 100 / MEM_TOTAL))
    MEM_CLASS=$(get_percent_class $MEM_PERCENT)
    MEM_TEXT_CLASS="stat-value-$MEM_CLASS"
else
    MEM_INFO="н/д"
    MEM_PERCENT=0
    MEM_CLASS=""
    MEM_TEXT_CLASS=""
fi

# Топ-6 процессов по памяти
TOP_MEM_ROWS=$(for pid in $(ls /proc | grep '^[0-9]'); do
    rss=$(grep VmRSS /proc/$pid/status 2>/dev/null | awk '{print $2}')
    [ -z "$rss" ] || [ "$rss" = "0" ] && continue
    printf "%d %s\n" "$rss" "$(cat /proc/$pid/comm 2>/dev/null)"
done | sort -rn | head -6 | awk '{
    rss=$1; $1=""; cmd=substr($0,2); gsub(/^ /,"",cmd)
    if (rss >= 1024) printf "<tr><td>%s</td><td>%d MB</td></tr>\n", cmd, rss/1024
    else printf "<tr><td>%s</td><td>%d KB</td></tr>\n", cmd, rss
}')

TMPFS_ROWS=$(df -h 2>/dev/null | awk '/^tmpfs/ {
    fs=$1; size=$2; used=$3; avail=$4; use=$5; mount=$6
    gsub(/%/,"",use)
    if (use > 90) class="critical"; else if (use > 70) class="warning"; else class="normal"
    printf "   <tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td><span class='\''stat-value-%s'\''>%s</span></td><td><a href=\"/entware-cgi/tmpfs.cgi?path=%s\" style=\"text-decoration:none; color:inherit;\">%s</a></td></tr>\n", fs, size, used, avail, class, $5, mount, mount
}')
[ -z "$TMPFS_ROWS" ] && TMPFS_ROWS="<tr><td colspan='6'>Нет данных о tmpfs</td></tr>"

BLOCK_ROWS=$(df -h 2>/dev/null | awk '/^\/dev\// {
    fs=$1; size=$2; used=$3; avail=$4; use=$5; mount=$6
    gsub(/%/,"",use)
    if (use > 90) class="critical"; else if (use > 70) class="warning"; else class="normal"
    printf "<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td><span class='\''stat-value-%s'\''>%s</span></td><td><a href=\"/entware-cgi/tmpfs.cgi?path=%s\" style=\"text-decoration:none; color:inherit;\">%s</a></td></tr>\n", fs, size, used, avail, class, $5, mount, mount
}')
[ -z "$BLOCK_ROWS" ] && BLOCK_ROWS="<tr><td colspan='6'>Нет данных о блочных устройствах</td></tr>"

if command -v /opt/bin/opkg >/dev/null 2>&1; then
    installed_count=$(/opt/bin/opkg list-installed 2>/dev/null | wc -l)
    available_count=$(/opt/bin/opkg list 2>/dev/null | wc -l)
else
    installed_count="н/д"
    available_count="н/д"
fi

df_output=$(df -h /opt 2>/dev/null | tail -n 1)
if [ -n "$df_output" ]; then
    fs_size=$(echo "$df_output" | awk '{print $2}')
    fs_used=$(echo "$df_output" | awk '{print $3}')
    fs_avail=$(echo "$df_output" | awk '{print $4}')
    fs_use_percent=$(echo "$df_output" | awk '{print $5}')
    fs_mount=$(echo "$df_output" | awk '{print $6}')
    fs_use_num=$(echo "$fs_use_percent" | sed 's/%//')
    fs_class=$(get_percent_class $fs_use_num)
    fs_text_class="stat-value-$fs_class"
else
    fs_size="н/д"; fs_used="н/д"; fs_avail="н/д"; fs_use_percent="н/д"; fs_mount="/opt"
    fs_use_num=0; fs_class=""; fs_text_class=""; fs_bar_width=0
fi

cat <<STATS
<h2 style="display: flex; align-items: center; gap: 10px;">
    <span class="stat-icon" style="width: 32px; height: 32px;">
        <svg class="icon" width="32" height="32">
            <use href="/entware-manager/icons.svg?v=2#icon-stats"/>
        </svg>
    </span>
    Статистика системы
</h2>

<div class="stats-grid">
    <div class="stat-card system">
        <h3>
            <span class="stat-icon">
                <svg class="icon" width="24" height="24">
                    <use href="/entware-manager/icons.svg?v=2#icon-stats"/>
                </svg>
            </span>
            Система
        </h3>
        <table class="stat-table">
            <tr><td>Модель:</td><td>$(html_escape "$MODEL")</td></tr>
            <tr><td>Имя хоста:</td><td>$(html_escape "$HOSTNAME")</td></tr>
            <tr><td>Архитектура:</td><td>$(html_escape "$ARCH")</td></tr>
            <tr><td>Версия ядра:</td><td>$(html_escape "$KERNEL")</td></tr>
            <tr><td>Время работы:</td><td>$(html_escape "$UPTIME")</td></tr>
        </table>
    </div>

    <div class="stat-card memory">
        <h3>
            <span class="stat-icon">
                <svg class="icon" width="24" height="24">
                    <use href="/entware-manager/icons.svg?v=2#icon-memory"/>
                </svg>
            </span>
            Память (RAM)
        </h3>
        <table class="stat-table">
            <tr><td>Использовано / Всего:</td><td>$(html_escape "$MEM_INFO")</td></tr>
            <tr><td>Загрузка:</td><td><span class="$MEM_TEXT_CLASS">$MEM_PERCENT%</span></td></tr>
        </table>
        <div class="progress-bar">
            <div class="progress-bar-fill fill-$MEM_CLASS" style="width: ${MEM_PERCENT}%;"></div>
        </div>
        $(if [ -n "$TOP_MEM_ROWS" ]; then echo '<div class="top-mem-wrapper top-mem-'"$MEM_CLASS"'"><table class="top-mem"><tr><th colspan="2">Топ по памяти</th></tr>'"$TOP_MEM_ROWS"'</table></div>'; fi)
    </div>

    <div class="stat-card packages">
        <h3>
            <span class="stat-icon">
                <svg class="icon" width="24" height="24">
                    <use href="/entware-manager/icons.svg?v=2#icon-package"/>
                </svg>
            </span>
            Пакеты Entware
        </h3>
        <table class="stat-table">
            <tr><td>Установлено:</td><td>$(html_escape "$installed_count")</td></tr>
            <tr><td>Доступно:</td><td>$(html_escape "$available_count")</td></tr>
        </table>
    </div>

    <div class="stat-card disk">
        <h3>
            <span class="stat-icon">
                <svg class="icon" width="24" height="24">
                    <use href="/entware-manager/icons.svg?v=2#icon-disk"/>
                </svg>
            </span>
            Диск (/opt)
        </h3>
        <table class="stat-table">
            <tr><td>Размер:</td><td>$(html_escape "$fs_size")</td></tr>
            <tr><td>Использовано:</td><td>$(html_escape "$fs_used")</td></tr>
            <tr><td>Доступно:</td><td>$(html_escape "$fs_avail")</td></tr>
            <tr><td>Загрузка:</td><td><span class="$fs_text_class">$(html_escape "$fs_use_percent")</span></td></tr>
        </table>
        <div class="progress-bar">
            <div class="progress-bar-fill fill-$fs_class" style="width: ${fs_use_num}%;"></div>
        </div>
    </div>
</div>

<div class="stat-card network" id="networkCard">
    <h3>
        <span class="stat-icon">
            <svg class="icon" width="24" height="24">
                <use href="/entware-manager/icons.svg?v=2#icon-router"/>
            </svg>
        </span>
        Сеть
    </h3>
    <div id="networkTable">
        <div style="padding: 0.5rem 1rem;">Загрузка...</div>
    </div>
    <div style="margin-top: 8px; display: flex; gap: 8px;">
        <button id="network-refresh" class="packages-delete-btn" style="padding: 4px 8px; font-size: 12px;">
            <svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg>
        </button>
    </div>
</div>

<div class="stat-card tmpfs">
    <h3>
        <span class="stat-icon">
            <svg class="icon" width="24" height="24">
                <use href="/entware-manager/icons.svg?v=2#icon-folder"/>
            </svg>
        </span>
        tmpfs
    </h3>
    <div class="table-wrapper">
        <table>
            <thead><tr><th>ФС</th><th>Размер</th><th>Использовано</th><th>Доступно</th><th>Загрузка</th><th>Точка монтирования</th></tr></thead>
            <tbody>
            $TMPFS_ROWS
            </tbody>
        </table>
    </div>
</div>

<div class="stat-card storage">
    <h3>
        <span class="stat-icon">
            <svg class="icon" width="24" height="24">
                <use href="/entware-manager/icons.svg?v=2#icon-disk"/>
            </svg>
        </span>
        Блочные устройства
    </h3>
    <div class="table-wrapper">
        <table>
            <thead><tr><th>ФС</th><th>Размер</th><th>Использовано</th><th>Доступно</th><th>Загрузка</th><th>Точка монтирования</th></tr></thead>
            <tbody>
            $BLOCK_ROWS
            </tbody>
        </table>
    </div>
</div>
STATS
