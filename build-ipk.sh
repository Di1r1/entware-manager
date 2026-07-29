#!/bin/bash
# ==============================================
# Сборка ipk для Entware Manager (tar.gz формат)
# Использование: ./build-ipk.sh [--arch ARCH]
#   --arch ARCH  — собрать только для одной архи (arm64/arm/mips/mipsel)
# ==============================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

VERSION=$(jq -r '.version' version.json 2>/dev/null || python3 -c "import json; print(json.load(open('version.json'))['version'])" 2>/dev/null || grep -o '"version"[^,]*' version.json | cut -d'"' -f4 || echo "1.06.4")
ARCHS=("arm64" "arm" "mips" "mipsel")
BUILD_ARCH=""

for arg in "$@"; do
    case "$arg" in
        --arch=*) BUILD_ARCH="${arg#--arch=}" ;;
    esac
done

DEPLOY_DIR="$SCRIPT_DIR/deploy"
OUT_DIR="$SCRIPT_DIR"
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

    # Per-arch зависимости (на aarch64 пакеты называются иначе)
    case "$arch" in
        arm64)   DEPS="lighttpd, lighttpd-mod-cgi, jq, curl, ttyd, coreutils, coreutils-timeout, procps-ng, bridge, ip-full, sudo, smartmontools, smartmontools-drivedb" ;;
        *)       DEPS="lighttpd, lighttpd-mod-cgi, jq, curl, ttyd, coreutils-base, coreutils-timeout, procps-ng, bridge-utils, ip-full, sudo, smartmontools, smartmontools-drivedb" ;;
    esac

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

    # postinst
    cat > "$PKG_TMP/control/postinst" <<'INSTEOF'
#!/bin/sh
# postinst — настройка после установки
/opt/web_entware/Install/install.sh
INSTEOF
    chmod 755 "$PKG_TMP/control/postinst"

    # prerm
    cat > "$PKG_TMP/control/prerm" <<'RMEEOF'
#!/bin/sh
# prerm — чистка конфигов перед удалением

LIGHTTPD_CONF="/opt/etc/lighttpd/lighttpd.conf"
CGI_CONF="/opt/etc/lighttpd/conf.d/30-cgi.conf"
SUDOERS_FILE="/opt/etc/sudoers.d/entware-smartctl"

# Удаляем наши конфиги
rm -f "/opt/etc/lighttpd/conf.d/90-entware-manager.conf" 2>/dev/null
rm -f "$CGI_CONF" 2>/dev/null

# Чистим старые строки из lighttpd.conf
[ -f "$LIGHTTPD_CONF" ] && sed -i '\|"/entware-manager/"|d' "$LIGHTTPD_CONF" || true
[ -f "$LIGHTTPD_CONF" ] && sed -i '\|"/entware-cgi/"|d' "$LIGHTTPD_CONF" || true
sed -i '/^[[:space:]]*server\.port[[:space:]]*=.*8087.*/d' "$LIGHTTPD_CONF" 2>/dev/null

# sudoers
rm -f "$SUDOERS_FILE" 2>/dev/null

exit 0
RMEEOF
    chmod 755 "$PKG_TMP/control/prerm"

    # control.tar.gz (с ./ префиксом — как в стандартных ipk)
    cd "$PKG_TMP/control"
    tar -czf "$PKG_TMP/control.tar.gz" ./control ./postinst ./prerm
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

    # Сборка ipk в tar.gz-формате (как в Entware)
    IPK_FILE="$OUT_DIR/entware-manager_${VERSION}_${arch}.ipk"
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
