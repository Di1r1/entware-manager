#!/bin/sh
# fix-lighttpd.sh — диагностика lighttpd для Entware Manager (ТОЛЬКО --check).
#
# Ничего не меняет: читает конфиги, логи и порты, печатает отчёт [OK]/[WARN]/[INFO]
# с готовыми командами фикса. Создан по мотивам инцидента nfqws2 (v1.09.20):
# cgi.assign в чужом conf.d при незагруженном mod_cgi даёт 403 на PHP-интерфейсах.
#
# Актуально в первую очередь для запасного lighttpd-режима (EWM_MODE=lighttpd)
# и сторонних веб-приложений рядом с панелью. В go-режиме (по умолчанию) панель
# работает на собственном entware-server и от lighttpd не зависит.
#
# Использование: sh fix-lighttpd.sh [--help]
# Exit: 0 — проблем не найдено, 1 — есть предупреждения.
set -eu
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin

BASE_DIR="/opt/web_entware"
LIGHTTPD_CONF="/opt/etc/lighttpd/lighttpd.conf"
CONF_D="/opt/etc/lighttpd/conf.d"
ERR_LOG="/opt/var/log/lighttpd/error.log"

case "${1:-}" in
    -h|--help|"")
        ;;
    *)
        echo "Неизвестный аргумент: $1 (скрипт работает только в режиме проверки)"; exit 2 ;;
esac
if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    cat <<USAGE
fix-lighttpd.sh — диагностика lighttpd (только проверка, ничего не меняет)

Проверяет: работу lighttpd, загрузку mod_cgi, конфликт cgi.assign без модуля,
свежие WARNING в error.log, состояние конфигов Entware Manager, порты 8086/8087.

Exit codes: 0 — чисто, 1 — есть предупреждения.
USAGE
    exit 0
fi

# find_pids() из lib/common.sh (без pgrep-зависимости там, где его нет)
if [ -f "$BASE_DIR/lib/common.sh" ]; then
    # shellcheck disable=SC1091
    . "$BASE_DIR/lib/common.sh"
else
    find_pids() {
        ps w 2>/dev/null | grep -v grep | grep -E "$1" | awk '{print $1}'
    }
fi

WARN_COUNT=0
ok()   { echo "  [OK]   $1"; }
warn() { echo "  [WARN] $1"; WARN_COUNT=$((WARN_COUNT + 1)); }
info() { echo "  [INFO] $1"; }

echo "=== Диагностика lighttpd ($(date '+%Y-%m-%d %H:%M:%S')) ==="
echo ""

echo "--- 1. Процессы ---"
if find_pids lighttpd 2>/dev/null | grep -q .; then
    ok "lighttpd запущен (pid: $(find_pids lighttpd | tr '\n' ' '))"
else
    warn "lighttpd не запущен — все веб-интерфейсы на нём недоступны (/opt/etc/init.d/S**lighttpd start)"
fi
EWM_PID=$(find_pids "entware-server" 2>/dev/null | head -1 || true)
if [ -n "$EWM_PID" ]; then
    info "entware-server запущен (pid: $EWM_PID) — панель в go-режиме, от lighttpd не зависит"
else
    info "entware-server не запущен — если панель настроена на lighttpd-режим, это ожидаемо"
fi
echo ""

echo "--- 2. Модуль mod_cgi ---"
MOD_SO="/opt/lib/lighttpd/mod_cgi.so"
if [ -f "$MOD_SO" ]; then
    ok "$MOD_SO установлен"
else
    warn "$MOD_SO отсутствует → opkg install lighttpd-mod-cgi"
fi
CGI_LOADED_FILE=""
for f in "$LIGHTTPD_CONF" "$CONF_D"/*.conf; do
    [ -f "$f" ] || continue
    if grep -q 'mod_cgi' "$f" 2>/dev/null; then
        CGI_LOADED_FILE="$f"
        break
    fi
done
if [ -n "$CGI_LOADED_FILE" ]; then
    ok "mod_cgi загружается из: $CGI_LOADED_FILE"
else
    warn "server.modules += ( \"mod_cgi\" ) не найден ни в одном конфиге — CGI/PHP у всех приложений не работает"
fi
echo ""

echo "--- 3. Конфликты: cgi.assign без mod_cgi (маркер инцидента nfqws) ---"
if grep -q 'mod_cgi' "$LIGHTTPD_CONF" "$CONF_D"/*.conf 2>/dev/null; then
    ok "конфликт невозможен: mod_cgi объявлен"
else
    FOUND_ASSIGN=""
    for f in "$CONF_D"/*.conf; do
        [ -f "$f" ] || continue
        if grep -q 'cgi\.assign' "$f" 2>/dev/null; then
            FOUND_ASSIGN="$FOUND_ASSIGN $f"
        fi
    done
    if [ -n "$FOUND_ASSIGN" ]; then
        for f in $FOUND_ASSIGN; do
            warn "$f использует cgi.assign, но mod_cgi не загружен → 403 на PHP/CGI."
            echo "         Фикс: добавить первой строкой в $f:"
            echo '           server.modules += ( "mod_cgi" )'
        done
    else
        ok "cgi.assign в conf.d не используется"
    fi
fi
echo ""

echo "--- 4. Свежие WARNING в error.log ---"
if [ -f "$ERR_LOG" ]; then
    if tail -n 40 "$ERR_LOG" 2>/dev/null | grep -i "warning" > /tmp/ewm_lt_warn.$$ 2>/dev/null && [ -s /tmp/ewm_lt_warn.$$ ]; then
        warn "есть свежие предупреждения (последние строки):"
        sed 's/^/         /' /tmp/ewm_lt_warn.$$ | tail -5
        if grep -q "unknown config-key" /tmp/ewm_lt_warn.$$; then
            info "«unknown config-key» почти всегда = директива модуля, который не загружен (см. п.2–3)"
        fi
    else
        ok "свежих предупреждений нет"
    fi
    rm -f /tmp/ewm_lt_warn.$$
else
    info "$ERR_LOG отсутствует (lighttpd ещё не писал логи)"
fi
echo ""

echo "--- 5. Конфиги Entware Manager ---"
OUR_CONF="$CONF_D/90-entware-manager.conf"
if [ -f "$OUR_CONF" ]; then
    if grep -q "listen.*8086\|server.port.*8086" "$OUR_CONF" 2>/dev/null; then
        info "$OUR_CONF — порт-хранитель (:8086), панель на собственном сервере"
    elif grep -q "server.port.*8087" "$OUR_CONF" 2>/dev/null; then
        info "$OUR_CONF — полный режим панели через lighttpd (:8087)"
    else
        warn "$OUR_CONF без узнаваемого порта (8086/8087) — возможно, повреждён → переустановка панели восстановит"
    fi
else
    info "$OUR_CONF отсутствует (панель не установлена в lighttpd или go-режим)"
fi
SHARED_CGI="$CONF_D/30-cgi.conf"
if [ -f "$SHARED_CGI" ]; then
    info "$SHARED_CGI существует — общий файл, EM его не изменяет (политика v1.09.20)"
fi
echo ""

echo "--- 6. Порты 8086/8087 ---"
if command -v netstat >/dev/null 2>&1; then
    LISTEN=$(netstat -tln 2>/dev/null || true)
    for port in 8087 8086; do
        if echo "$LISTEN" | grep -q ":${port}[[:space:]]"; then
            ok "порт :$port слушается"
        else
            info "порт :$port свободен"
        fi
    done
    if echo "$LISTEN" | grep -q ":8087" && [ -z "$EWM_PID" ] && ! echo "$LISTEN" | grep -q ":8086"; then
        ok ":8087 занят при незапущенном entware-server → это lighttpd-режим панели"
    fi
else
    info "netstat недоступен — проверка портов пропущена"
fi
echo ""

if [ "$WARN_COUNT" -eq 0 ]; then
    echo "=== Итог: проблем не найдено ==="
    exit 0
fi
echo "=== Итог: предупреждений: $WARN_COUNT (см. [WARN] выше) ==="
exit 1
