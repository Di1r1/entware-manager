#!/bin/bash
# shellcheck disable=SC2034
# ==============================================
# Сборка deploy-папки для Entware Manager
# Версия: 3.0 — multi-arch Go compilation
# Использование: ./build-deploy.sh [--arch ARCH] [--tar]
#   --arch ARCH  — собрать только для одной архи (arm64/arm/mips/mipsel)
#                  По умолчанию: все архитектуры
#   --tar        — дополнительно создать tar.gz архив
# ==============================================

set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEPLOY_DIR="$PROJECT_DIR/deploy"
TIMESTAMP=$(date '+%Y%m%d_%H%M%S')

ARCH_NAMES=(arm64 arm mips mipsel)
ARCH_GOARCH=(arm64 arm mips mipsle)
ARCH_FLAGS=("" "GOARM=5" "GOMIPS=softfloat" "GOMIPS=softfloat")

BUILD_ARCHS=""
BUILD_TAR=false
for arg in "$@"; do
    case "$arg" in
        --tar) BUILD_TAR=true ;;
        --arch=*) BUILD_ARCHS="${arg#--arch=}" ;;
    esac
done

rm -rf "$DEPLOY_DIR"
mkdir -p "$DEPLOY_DIR"

echo "=== Сборка deploy ==="

for f in "$PROJECT_DIR"/*; do
    name=$(basename "$f")
    case "$name" in
        deploy|go|tmp|build-deploy.sh|Makefile|build-ipk.sh|forum_post.md|TECH_SPEC.md|RULES.md|links.json|DEVLOG.md|DEVICE.md|BUILD.md|router_backup|conffiles|control|postinst|prerm|*_config.json|*.tar.gz|*.ipk)
            continue ;;
    esac
    if [ -d "$f" ]; then
        cp -a "$f" "$DEPLOY_DIR/"
        echo "  DIR:  $name"
    elif [ -f "$f" ]; then
        cp -a "$f" "$DEPLOY_DIR/"
        echo "  FILE: $name"
    fi
done

# Удаляем dev-артефакты и пользовательские конфиги из deploy
rm -f "$DEPLOY_DIR/Install/Install.txt" "$DEPLOY_DIR/doc/NETWORK_PROMPT.md" "$DEPLOY_DIR/doc/IPK_BUILD.md" "$DEPLOY_DIR/doc/NETDATA_MANUAL.md" "$DEPLOY_DIR/logger/config.json" 2>/dev/null

echo ""
echo "=== Компиляция Go ==="
rm -rf "$DEPLOY_DIR/cgi-bin/go"
mkdir -p "$DEPLOY_DIR/cgi-bin/go"
cd "$PROJECT_DIR/go"

for i in "${!ARCH_NAMES[@]}"; do
    arch_name="${ARCH_NAMES[$i]}"
    goarch="${ARCH_GOARCH[$i]}"
    goflags="${ARCH_FLAGS[$i]}"
    dir_name="$arch_name"

    if [ -n "$BUILD_ARCHS" ] && [ "$arch_name" != "$BUILD_ARCHS" ]; then
        continue
    fi

    mkdir -p "$DEPLOY_DIR/cgi-bin/go/$dir_name"
    echo ""
    echo "  [$arch_name] (GOARCH=$goarch${goflags:+ $goflags})"

    for cmd in entware-pkg entware-stats entware-net entware-logger entware-services entware-monitor entware-smart; do
        echo -n "    $cmd... "
        out="$DEPLOY_DIR/cgi-bin/go/$dir_name/$cmd"
        env GOOS=linux GOARCH="$goarch" CGO_ENABLED=0 $goflags go build -ldflags="-s -w" -o "$out" "./cmd/$cmd/" 2>&1
        echo "OK ($(du -h "$out" | cut -f1))"
    done
done

if command -v upx &>/dev/null || [ -x /tmp/upx-4.2.4-amd64_linux/upx ]; then
    UPX=$(command -v upx || echo "/tmp/upx-4.2.4-amd64_linux/upx")
    echo ""
    echo "=== UPX сжатие ==="
    for arch_dir in "$DEPLOY_DIR"/cgi-bin/go/*/; do
        [ -d "$arch_dir" ] || continue
        echo "  [$(basename "$arch_dir")]"
        for f in "$arch_dir"entware-*; do
            [ -f "$f" ] || continue
            echo -n "    $(basename $f)... "
            tmpf=$(mktemp -u)
            "$UPX" -9 "$f" -o "$tmpf" 2>/dev/null && mv "$tmpf" "$f" && echo "OK ($(du -h "$f" | cut -f1))" || { rm -f "$tmpf"; echo "SKIP"; }
        done
    done
fi

cp "$PROJECT_DIR/cgi-bin/go.cgi" "$DEPLOY_DIR/cgi-bin/go.cgi"

echo ""
echo "=== Симлинки cgi → go.cgi ==="

cd "$DEPLOY_DIR/cgi-bin"
for ep in api auth_config available backup backup_restore check_deps check_syntax crontab crontab_update delete_file help install kill_pid links_load links_save network_action network_arp network_events network_interfaces network_routes network_stats network_status packages prepare_offline remove service_action services smart stats temp_history temperature tmpfs ttyd_control update update_check update_run update_status upgradable upgrade version view_file wifi_temp wifi_temp_history; do
    ln -sf go.cgi "$ep.cgi"
    echo "  $ep.cgi -> go.cgi"
done

for d in network logger monitor service_watchdog; do
    mkdir -p "$d"
done

cd network
for ep in action arp config events interfaces routes stats status; do
    ln -sf ../go.cgi "$ep.cgi"
done
echo "  network/*.cgi -> ../go.cgi"

cd ../logger
for ep in clear config debug debug_path find_by_name rotate system_log system_logs view; do
    ln -sf ../go.cgi "$ep.cgi"
done
echo "  logger/*.cgi -> ../go.cgi"

cd ../monitor
for ep in action config log status; do
    ln -sf ../go.cgi "$ep.cgi"
done
echo "  monitor/*.cgi -> ../go.cgi"

cd ../service_watchdog
for ep in action config events status; do
    ln -sf ../go.cgi "$ep.cgi"
done
echo "  service_watchdog/*.cgi -> ../go.cgi"

cd "$DEPLOY_DIR"

find "$DEPLOY_DIR/cgi-bin" -type l -exec chmod 755 {} \;
chmod 755 "$DEPLOY_DIR/cgi-bin/go.cgi"
find "$DEPLOY_DIR/cgi-bin/go" -type f -exec chmod 755 {} \;
find "$DEPLOY_DIR" -type d -exec chmod 755 {} \;
find "$DEPLOY_DIR" -type f \( -name "*.js" -o -name "*.css" -o -name "*.html" -o -name "*.json" -o -name "*.svg" \) -exec chmod 644 {} + 2>/dev/null || true
find "$DEPLOY_DIR" -type f -name "*.sh" -exec chmod 755 {} \;

find "$DEPLOY_DIR" -name '*.bak' -delete 2>/dev/null

echo ""
echo "=== Deploy собран: $DEPLOY_DIR ==="
echo "Размер: $(du -sh "$DEPLOY_DIR" | cut -f1)"
echo "Файлов: $(find "$DEPLOY_DIR" -type f | wc -l)"

echo ""
echo "Архитектуры в сборке:"
for arch_dir in "$DEPLOY_DIR"/cgi-bin/go/*/; do
    [ -d "$arch_dir" ] || continue
    count=$(find "$arch_dir" -type f | wc -l)
    total=$(du -sh "$arch_dir" | cut -f1)
    echo "  $arch_dir → $count файлов, $total"
done

if $BUILD_TAR; then
    ARCHIVE="$PROJECT_DIR/entware-manager-all.tar.gz"
    tar -czf "$ARCHIVE" -C "$PROJECT_DIR" deploy/
    echo ""
    echo "Архив: $ARCHIVE ($(du -h "$ARCHIVE" | cut -f1))"
fi

echo ""
echo "Для установки на роутер:"
echo "  1. Скопируйте deploy/ в /opt/tmp/ на роутере"
echo "  2. cd /opt/tmp/deploy && sh Install/install.sh"
echo "  (install.sh сам определит архитектуру и возьмёт нужные бинарники)"
