#!/bin/sh
# ==============================================
# Просмотр системного лога событий логирования
# Версия: 1.8 (полные пути /opt/bin для CGI контекста)
# Дата: 2026-04-01
# ==============================================

SYSTEM_LOG="/opt/var/log/entware/system.log"

echo "Content-type: text/html"
echo ""

cat <<'HTMLHEAD'
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { margin: 0; padding: 16px; background: transparent; color: #e0e0e0; font-family: monospace; font-size: 13px; }
        pre { margin: 0; white-space: pre-wrap; word-break: break-all; }
        .empty { color: #718096; text-align: center; padding: 20px; }
    </style>
</head>
<body>
HTMLHEAD

if [ -f "$SYSTEM_LOG" ] && [ -s "$SYSTEM_LOG" ]; then
    echo '<pre>'
    /opt/bin/cat "$SYSTEM_LOG" | /opt/bin/sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g'
    echo '</pre>'
else
    echo '<div class="empty">Системный лог пуст</div>'
fi

cat <<'HTMLFOOT'
</body>
</html>
HTMLFOOT
