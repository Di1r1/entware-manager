#!/bin/sh
# ==============================================
# Просмотр системных логов (поддержка параметра file)
# Версия: 1.2 (полные пути /opt/bin для CGI контекста)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

html_header

QUERY_STRING="${QUERY_STRING:-}"
source_name=$(echo "$QUERY_STRING" | sed -n 's/.*source=\([^&]*\).*/\1/p')
file_path=$(echo "$QUERY_STRING" | sed -n 's/.*file=\([^&]*\).*/\1/p')
level_filter=$(echo "$QUERY_STRING" | sed -n 's/.*level=\([^&]*\).*/\1/p')
search=$(echo "$QUERY_STRING" | sed -n 's/.*search=\([^&]*\).*/\1/p')
search=$(url_decode "$search")

source_name=$(url_decode "$source_name")
file_path=$(url_decode "$file_path")

if [ -n "$file_path" ] && [ -f "$file_path" ]; then
    LOG_FILE="$file_path"
else
    CONFIG="/opt/web_entware/logger/system_sources.json"
    if [ -f "$CONFIG" ] && /opt/bin/jq --version >/dev/null 2>&1; then
        LOG_FILE=$(/opt/bin/jq -r --arg name "$source_name" '.sources[] | select(.name == $name) | .file' "$CONFIG")
    fi
fi

if [ -z "$LOG_FILE" ] || [ ! -f "$LOG_FILE" ]; then
    echo "<div class='no-logs'>Лог-файл не найден: $LOG_FILE</div>"
    exit 0
fi

cat << 'EOFHTML'
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Системные логи</title>
    <link rel="stylesheet" href="/entware-manager/logger/style.css">
    <style>
        .logs-container {
            padding: 1rem;
        }
        .log-line {
            font-family: monospace;
            font-size: 13px;
            padding: 4px 8px;
            border-bottom: 1px solid var(--border-color);
            white-space: pre-wrap;
            word-break: break-all;
            background: var(--input-bg);
            margin: 2px 0;
            border-radius: 6px;
            color: var(--text-primary);
        }
        .no-logs {
            text-align: center;
            padding: 2rem;
            color: var(--text-muted);
        }
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
<div class="logs-container">
EOFHTML

if [ -n "$search" ]; then
    grep -i "$search" "$LOG_FILE" 2>/dev/null | while IFS= read -r line; do
        echo "<div class='log-line'>$(echo "$line" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')</div>"
    done
else
    tail -n 500 "$LOG_FILE" 2>/dev/null | while IFS= read -r line; do
        echo "<div class='log-line'>$(echo "$line" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')</div>"
    done
fi

echo '</div></body></html>'
