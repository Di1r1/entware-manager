#!/bin/bash
# ==============================================
# Сборка ipk для Entware Manager (tar.gz формат)
# Использование: ./build-ipk.sh [--arch ARCH]
#   --arch ARCH  — собрать только для одной архи (arm64/mips/mipsel)
# ==============================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

VERSION="${VERSION:-$(jq -r '.version' version.json 2>/dev/null || python3 -c "import json; print(json.load(open('version.json'))['version'])" 2>/dev/null || grep -o '"version"[^,]*' version.json | cut -d'"' -f4)}"
if [ -z "$VERSION" ]; then
    echo "ERROR: не удалось определить версию из version.json" >&2
    exit 1
fi
ARCHS=("arm64" "mips" "mipsel")
BUILD_ARCH=""

for arg in "$@"; do
    case "$arg" in
        --arch=*) BUILD_ARCH="${arg#--arch=}" ;;
    esac
done

DEPLOY_DIR="$SCRIPT_DIR/deploy"
OUT_DIR="$SCRIPT_DIR/dist"
mkdir -p "$OUT_DIR"
PKG_TMP="/tmp/entware-ipk-$$"

cleanup() { rm -rf "$PKG_TMP"; }
trap cleanup EXIT

echo "=== Сборка ipk для Entware Manager v$VERSION ==="

for arch in "${ARCHS[@]}"; do
    [ -n "$BUILD_ARCH" ] && [ "$arch" != "$BUILD_ARCH" ] && continue

    echo ""
    echo "--- $arch ---"

    rm -rf "$PKG_TMP"
    mkdir -p "$PKG_TMP/control" "$PKG_TMP/data"

    # debian-binary
    echo "2.0" > "$PKG_TMP/debian-binary"

    DEPS="lighttpd, lighttpd-mod-cgi, lighttpd-mod-proxy, lighttpd-mod-deflate, lighttpd-mod-access, jq, curl, ttyd, htop, coreutils, coreutils-timeout, procps-ng, bridge, ip-full, sudo, bash, smartmontools, smartmontools-drivedb"

    # control (Architecture: all — не зависит от версии ядра)
    cat > "$PKG_TMP/control/control" <<EOF
Package: entware-manager
Version: $VERSION
Architecture: all
Maintainer: Di1r1
Priority: optional
Section: admin
Description: Web panel for Entware management on Keenetic/Netcraze
Depends: $DEPS
EOF

# conffiles — seed-манифесты моста: при opkg upgrade пользовательские
    # правки через UI (koffe.json и др.) не затираются (opkg сохраняет).
    cat > "$PKG_TMP/control/conffiles" <<'CONFEOF'
/opt/web_entware/bridge/adguard.json
/opt/web_entware/bridge/koffe.json
/opt/web_entware/bridge/syncthing.json
/opt/web_entware/bridge/transmission.json
CONFEOF

    # postinst
    cat > "$PKG_TMP/control/postinst" <<'INSTEOF'
#!/bin/sh
# postinst — настройка после установки.
# OPKG_POSTINST=1 сообщает install.sh, что мы внутри postinst: opkg уже держит
# lock на эту установку, поэтому install.sh НЕ должен вызывать opkg update/install
# (иначе self-deadlock). Модули lighttpd ставятся через Depends.
OPKG_POSTINST=1 /opt/web_entware/Install/install.sh
# Перезапуск веб-сервера, чтобы сбросить состояние proxy-бэкендов
# (grdp-proxy мог быть остановлен во время обновления файлов → 503 на /rdp/).
if [ -x /opt/etc/init.d/S80lighttpd ]; then
	/opt/etc/init.d/S80lighttpd restart >/dev/null 2>&1 || true
fi
exit 0
INSTEOF
    chmod 755 "$PKG_TMP/control/postinst"

    # prerm
    cat > "$PKG_TMP/control/prerm" <<'RMEEOF'
#!/bin/sh
# prerm — чистка конфигов ПЕРЕД УДАЛЕНИЕМ пакета.
#
# ВАЖНО: при upgrade ($1=upgrade) конфиги НЕ трогаем — opkg вызывает prerm
# перед установкой новой версии, и если процесс оборвётся (kill, обрыв SSH,
# таймаут), роутер останется без конфигов веб-сервера при живом пакете.
# Это была причина инцидента: повторный `opkg install` удалил конфиги,
# opkg был убит до записи статуса → панель мертва, база говорит "installed".
# Новые конфиги создаст postinst новой версии.
case "$1" in
	remove|purge|deconfigure) ;;
	upgrade|*) exit 0 ;;
esac

LIGHTTPD_CONF="/opt/etc/lighttpd/lighttpd.conf"
CGI_CONF="/opt/etc/lighttpd/conf.d/30-cgi.conf"
SUDOERS_FILE="/opt/etc/sudoers.d/entware-smartctl"

# Останавливаем entware-server (если был)
if [ -x /opt/etc/init.d/S80entware-server ]; then
	/opt/etc/init.d/S80entware-server stop 2>/dev/null
fi
rm -f /opt/etc/init.d/S80entware-server 2>/dev/null

# Порт-хранитель общего lighttpd НЕ удаляем (иначе lighttpd упадёт на порт 80
# вместе с koffe/web4static). Полноценную lighttpd-панель заменяем порт-хранителем.
if [ -f "/opt/web_entware/lib/migrate.sh" ]; then
	. /opt/web_entware/lib/migrate.sh
	if migrate_is_portkeeper "/opt/etc/lighttpd/conf.d/90-entware-manager.conf"; then
		: # порт-хранитель — оставляем как есть
	elif [ -f "/opt/etc/lighttpd/conf.d/90-entware-manager.conf" ]; then
		PK=$(migrate_choose_portkeeper)
		if [ -n "$PK" ]; then
			migrate_write_portkeeper "$PK"
		else
			rm -f "/opt/etc/lighttpd/conf.d/90-entware-manager.conf" 2>/dev/null
		fi
	fi
else
	rm -f "/opt/etc/lighttpd/conf.d/90-entware-manager.conf" 2>/dev/null
fi

# 30-cgi.conf — общий файл lighttpd (может принадлежать web4static/nfqws2).
# Удаляем ТОЛЬКО если это наш устаревший артефакт (ровно наш шаблон).
if [ -f "$CGI_CONF" ] && ! grep -Eq 'perl|ruby|python|php|\.pl|\.rb|\.erb|\.py|\.php' "$CGI_CONF" 2>/dev/null \
	&& [ "$(wc -l < "$CGI_CONF" 2>/dev/null || echo 0)" -le 3 ] \
	&& grep -q 'cgi\.assign.*\.cgi.*/bin/sh' "$CGI_CONF" 2>/dev/null; then
	rm -f "$CGI_CONF" 2>/dev/null
fi

# Чистим старые строки из lighttpd.conf
[ -f "$LIGHTTPD_CONF" ] && sed -i '\|"/entware-manager/"|d' "$LIGHTTPD_CONF" || true
[ -f "$LIGHTTPD_CONF" ] && sed -i '\|"/entware-cgi/"|d' "$LIGHTTPD_CONF" || true
sed -i '/^[[:space:]]*server\.port[[:space:]]*=.*8087.*/d' "$LIGHTTPD_CONF" 2>/dev/null

# sudoers
rm -f "$SUDOERS_FILE" 2>/dev/null

# Симлинк RDP-прокси (в ipk он в cgi-bin/go/ — opkg удалит сам, симлинк висит)
rm -f /opt/etc/init.d/S90grdp-proxy 2>/dev/null

# flatten-копии Go-бинарников, которые install.sh раскладывает из
# cgi-bin/go/<arch>/ в cgi-bin/go/ (opkg удаляет только пути из data.tar.gz)
rm -f /opt/web_entware/cgi-bin/go/entware-* 2>/dev/null
rm -f /opt/web_entware/cgi-bin/go/grdp-proxy 2>/dev/null

exit 0
RMEEOF
    chmod 755 "$PKG_TMP/control/prerm"

    # control.tar.gz (с ./ префиксом — как в стандартных ipk)
    cd "$PKG_TMP/control"
    tar -czf "$PKG_TMP/control.tar.gz" ./control ./postinst ./prerm ./conffiles
    cd "$PKG_TMP"

    # data.tar.gz — файлы проекта в /opt/web_entware/
    mkdir -p "$PKG_TMP/data/opt"
    cp -a "$DEPLOY_DIR" "$PKG_TMP/data/opt/web_entware"

    # чистим лишние архитектуры
    find "$PKG_TMP/data/opt/web_entware/cgi-bin/go" -mindepth 1 -maxdepth 1 -type d ! -name "$arch" -exec rm -rf {} + 2>/dev/null

    chmod 755 "$PKG_TMP/data/opt/web_entware/cgi-bin/go.cgi" 2>/dev/null
    find "$PKG_TMP/data/opt/web_entware/cgi-bin/go" -type f -exec chmod 755 {} \; 2>/dev/null
    find "$PKG_TMP/data/opt/web_entware" -type d -exec chmod 755 {} \; 2>/dev/null
    find "$PKG_TMP/data/opt/web_entware" -type f \( -name "*.js" -o -name "*.css" -o -name "*.html" -o -name "*.json" -o -name "*.svg" \) -exec chmod 644 {} \; 2>/dev/null || true
    find "$PKG_TMP/data/opt/web_entware" -type f -name "*.sh" -exec chmod 755 {} \; 2>/dev/null

    cd "$PKG_TMP/data"
    tar -czf "$PKG_TMP/data.tar.gz" .
    cd "$PKG_TMP"

    # Сборка ipk в tar.gz-формате (как в Entware — opkg принимает именно его)
    IPK_FILE="$OUT_DIR/entware-manager_${arch}.ipk"
    rm -f "$IPK_FILE"
    tar -czf "$IPK_FILE" \
        ./debian-binary \
        ./control.tar.gz \
        ./data.tar.gz

    SIZE=$(du -h "$IPK_FILE" | cut -f1)
    echo "  → $IPK_FILE ($SIZE)"
done

echo ""
echo "Готово."
