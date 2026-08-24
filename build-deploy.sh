#!/bin/bash
# shellcheck disable=SC2034
# ==============================================
# Сборка deploy-папки для Entware Manager
# Версия: 3.0 — multi-arch Go compilation
# Использование: ./build-deploy.sh [--arch ARCH] [--tar]
#   --arch ARCH  — собрать только для одной архи (arm64/mips/mipsel)
#                  По умолчанию: все архитектуры
#   --tar        — дополнительно создать tar.gz архив
# ==============================================

set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEPLOY_DIR="$PROJECT_DIR/deploy"
TIMESTAMP=$(date '+%Y%m%d_%H%M%S')

ARCH_NAMES=(arm64 mips mipsel)
ARCH_GOARCH=(arm64 mips mipsle)
ARCH_FLAGS=("" "GOMIPS=softfloat" "GOMIPS=softfloat")

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
        deploy|go|tmp|test|dist|build-deploy.sh|Makefile|build-ipk.sh|forum_post.md|TECH_SPEC.md|RULES.md|links.json|DEVLOG.md|DEVICE.md|BUILD.md|router_backup|conffiles|control|postinst|prerm|*_config.json|*.tar.gz|*.ipk)
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
rm -rf "$DEPLOY_DIR/doc/local"
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

    for cmd in entware-pkg entware-stats entware-net entware-logger entware-services entware-monitor entware-smart entware-server entware-rdp entware-telegram entware-bridge; do
        echo -n "    $cmd... "
        out="$DEPLOY_DIR/cgi-bin/go/$dir_name/$cmd"
        env GOOS=linux GOARCH="$goarch" CGO_ENABLED=0 $goflags go build -ldflags="-s -w" -o "$out" "./cmd/$cmd/" 2>&1
        echo "OK ($(du -h "$out" | cut -f1))"
    done
done

# Приоритет UPX: 5.2.0 (новый, быстрее) → 4.2.4 → системный.
if [ -x /tmp/upx-5.2.0-amd64_linux/upx ]; then
    UPX=/tmp/upx-5.2.0-amd64_linux/upx
elif [ -x /tmp/upx-4.2.4-amd64_linux/upx ]; then
    UPX=/tmp/upx-4.2.4-amd64_linux/upx
elif command -v upx &>/dev/null; then
    UPX=$(command -v upx)
fi
echo ""
echo "=== UPX сжатие ==="
if [ -z "${UPX:-}" ]; then
    echo "  UPX не найден — сжатие пропущено (ставьте upx в PATH)"
else
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

# ==============================================
# RDP-артефакты (WASM-клиент grdpwasm + grdp-proxy)
# Форк grdpwasm — ВНЕ репозитория (gitignored static/), собран в /tmp/opencode/grdpwasm.
# Если форк доступен — собираем grdp-proxy под каждую arch и копируем WASM-клиент.
# Если нет — предупреждаем (RDP-вкладка будет недоступна до ручной сборки).
# ==============================================
GRDP_FORK="${GRDP_FORK:-/tmp/opencode/grdpwasm}"
if [ -d "$GRDP_FORK/proxy" ] && [ -f "$GRDP_FORK/static/index.html" ]; then
    echo ""
    echo "=== RDP-артефакты (форк grdpwasm: $GRDP_FORK) ==="

    # WASM-клиент (архитектурно независим, один раз)
    mkdir -p "$DEPLOY_DIR/static/rdp"
    cp "$GRDP_FORK/static/index.html" "$DEPLOY_DIR/static/rdp/"
    if [ ! -f "$GRDP_FORK/static/main.wasm" ]; then
        echo "  WASM main.wasm не найден — собираю (может занять несколько минут)..."
        ( cd "$GRDP_FORK" && GOOS=js GOARCH=wasm go build -o static/main.wasm . ) || echo "  WARNING: сборка WASM не удалась"
    fi
    cp "$GRDP_FORK/static/main.wasm" "$DEPLOY_DIR/static/rdp/" 2>/dev/null && echo "  WASM-клиент: index.html + main.wasm ($(du -h "$DEPLOY_DIR/static/rdp/main.wasm" | cut -f1))"
    cp "$GRDP_FORK/static/wasm_exec.js" "$DEPLOY_DIR/static/rdp/" 2>/dev/null || true

    # grdp-proxy под каждую arch
    cd "$PROJECT_DIR/go"
    for i in "${!ARCH_NAMES[@]}"; do
        arch_name="${ARCH_NAMES[$i]}"
        goarch="${ARCH_GOARCH[$i]}"
        goflags="${ARCH_FLAGS[$i]}"
        if [ -n "$BUILD_ARCHS" ] && [ "$arch_name" != "$BUILD_ARCHS" ]; then
            continue
        fi
        echo -n "  [grdp-proxy $arch_name]... "
        out="$DEPLOY_DIR/cgi-bin/go/$arch_name/grdp-proxy"
        ( cd "$GRDP_FORK" && env GOOS=linux GOARCH="$goarch" CGO_ENABLED=0 $goflags go build -ldflags="-s -w" -o "$out" ./proxy/ ) 2>&1 && {
            "$UPX" -9 "$out" -o "$out.tmp" 2>/dev/null && mv "$out.tmp" "$out" || true
            echo "OK ($(du -h "$out" | cut -f1))"
        } || echo "FAIL"
    done
    cd "$PROJECT_DIR"

    chmod 755 "$DEPLOY_DIR/static/rdp/index.html"
    chmod 644 "$DEPLOY_DIR/static/rdp/main.wasm" "$DEPLOY_DIR/static/rdp/wasm_exec.js" 2>/dev/null
    chmod 755 "$DEPLOY_DIR"/cgi-bin/go/*/grdp-proxy 2>/dev/null
else
    echo ""
    echo "WARNING: форк grdpwasm не найден ($GRDP_FORK) — RDP-артефакты не собраны."
    echo "         Вкладка RDP будет недоступна до ручной сборки (см. doc/RDP_MODULE.md)."
fi

cp "$PROJECT_DIR/cgi-bin/go.cgi" "$DEPLOY_DIR/cgi-bin/go.cgi"

# ==============================================
# Форк index.html ttyd (перехват вставки Ctrl+V)
# Источник — сгенерированный артефакт вне репозитория (как grdpwasm).
# В xterm.js 5.4 (ttyd 1.7.7) Ctrl+V шлёт в PTY ^V — форк добавляет term.paste().
# ==============================================
TTYD_FORK="${TTYD_FORK:-/tmp/opencode/ttyd-fork.html}"
if [ -f "$TTYD_FORK" ]; then
    mkdir -p "$DEPLOY_DIR/static/ttyd"
    cp "$TTYD_FORK" "$DEPLOY_DIR/static/ttyd/index.html"
    chmod 644 "$DEPLOY_DIR/static/ttyd/index.html"
    echo "  ttyd index.html: форк ($(du -h "$DEPLOY_DIR/static/ttyd/index.html" | cut -f1))"
else
    echo "WARNING: форк ttyd index.html не найден ($TTYD_FORK) — вставка Ctrl+V в терминале не будет перехватываться."
    echo "         (сгенерируйте: curl встроенный index.html ttyd + инъекция term.paste)."
fi

echo ""
echo "=== Симлинки cgi → go.cgi ==="

cd "$DEPLOY_DIR/cgi-bin"
for ep in api auth_config available backup backup_restore bridge_action bridge_auth bridge_card bridge_delete bridge_discover bridge_manifest bridge_prefs bridge_save bridge_stats bridge_watch bridge_status check_deps check_syntax crontab crontab_update delete_file help install installed kill_pid links_load links_save login logout network_action network_arp network_events network_interfaces network_routes network_stats network_status packages prepare_offline rdp_config rdp_start rdp_status rdp_stop remove service_action services session smart stats telegram_config telegram_test temp_history temperature tmpfs tmpfs_clean ttyd_control update update_check update_run update_status upgradable upgrade version view_file wifi_temp wifi_temp_history; do
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
    mkdir -p "$PROJECT_DIR/dist"
    ARCHIVE="$PROJECT_DIR/dist/entware-manager-all.tar.gz"
    tar -czf "$ARCHIVE" -C "$PROJECT_DIR" deploy/
    echo ""
    echo "Архив: $ARCHIVE ($(du -h "$ARCHIVE" | cut -f1))"
fi

echo ""
echo "Для установки на роутер:"
echo "  1. Скопируйте deploy/ в /opt/tmp/ на роутере"
echo "  2. cd /opt/tmp/deploy && sh Install/install.sh"
echo "  (install.sh сам определит архитектуру и возьмёт нужные бинарники)"
