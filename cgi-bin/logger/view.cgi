#!/bin/sh
# ==============================================
# Просмотр логов с фиксированным фильтром и прокруткой
# Версия: 1.4 (используется index() вместо regex для поиска)
# Дата: 2026-04-01
# ==============================================

. /opt/web_entware/lib/common.sh

html_header

QUERY_STRING="${QUERY_STRING:-}"
date_filter=$(echo "$QUERY_STRING" | sed -n 's/.*date=\([^&]*\).*/\1/p')
level_filter=$(echo "$QUERY_STRING" | sed -n 's/.*level=\([^&]*\).*/\1/p')
search=$(echo "$QUERY_STRING" | sed -n 's/.*search=\([^&]*\).*/\1/p' | sed 's/+/ /g; s/%/\\x/g')
search=$(printf '%b' "$search" 2>/dev/null)

TMP_LOG_DIR="/tmp/entware/logs"
PERM_LOG_DIR="/opt/var/log/entware"

if [ -z "$date_filter" ]; then
    today=$(date +%Y-%m-%d)
    if [ -f "$TMP_LOG_DIR/$today.log" ]; then
        LOG_FILE="$TMP_LOG_DIR/$today.log"
    else
        LOG_FILE="$PERM_LOG_DIR/$today.log"
    fi
else
    if [ -f "$TMP_LOG_DIR/$date_filter.log" ]; then
        LOG_FILE="$TMP_LOG_DIR/$date_filter.log"
    else
        LOG_FILE="$PERM_LOG_DIR/$date_filter.log"
    fi
fi

cat << 'EOFHTML'
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Логи Entware Manager</title>
    <link rel="stylesheet" href="/entware-manager/logger/style.css">
    <style>
        html.night { background: #1a202c; }
        html:not(.night) { background: #f9fafb; }
    </style>
    <script>
        (function() {
            try {
                const isNight = localStorage.getItem('entware_theme') === 'night';
                if (isNight) {
                    document.documentElement.classList.add('night');
                } else {
                    document.documentElement.classList.remove('night');
                }
            } catch(e) {}
        })();
        window.addEventListener('storage', function(e) {
            if (e.key === 'entware_theme') {
                const isNight = e.newValue === 'night';
                if (isNight) {
                    document.documentElement.classList.add('night');
                } else {
                    document.documentElement.classList.remove('night');
                }
            }
        });
    </script>
</head>
<body>
EOFHTML

echo '<form class="log-filter" method="get">'
echo '<div><label>Дата (YYYY-MM-DD)</label><input type="date" name="date" value="'"$date_filter"'"></div>'
echo '<div><label>Уровень</label><select name="level">'
echo '<option value="">Все</option>'
if [ "$level_filter" = "INFO" ]; then
    echo '<option value="INFO" selected>INFO</option>'
else
    echo '<option value="INFO">INFO</option>'
fi
if [ "$level_filter" = "WARN" ]; then
    echo '<option value="WARN" selected>WARN</option>'
else
    echo '<option value="WARN">WARN</option>'
fi
if [ "$level_filter" = "ERROR" ]; then
    echo '<option value="ERROR" selected>ERROR</option>'
else
    echo '<option value="ERROR">ERROR</option>'
fi
echo '</select></div>'
echo '<div><label>Поиск</label><input type="text" name="search" value="'"$search"'" placeholder="текст для поиска"></div>'
echo '<div><button type="submit"><svg class="icon" width="14" height="14" style="fill: none; stroke: currentColor;"><use href="/entware-manager/icons.svg?v=2#icon-search"/></svg> Фильтровать</button></div>'
echo '</form>'

echo '<div class="logs-container">'

if [ ! -f "$LOG_FILE" ]; then
    echo "<div class='no-logs'>📄 Лог-файл не найден: $LOG_FILE</div>"
else
    awk -v level="$level_filter" -v search="$search" '
        BEGIN {
            skip = 0;
            level_pat = "\\[" level "\\]";
        }
        {
            if (level != "" && $0 !~ level_pat) skip = 1;
            if (search != "" && index($0, search) == 0) skip = 1;
            if (!skip) print $0;
            skip = 0;
        }
    ' "$LOG_FILE" | while IFS= read -r line; do
        echo "<div class='log-line'>$(echo "$line" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')</div>"
    done
fi

echo '</div>'
echo '</body>'
echo '</html>'
