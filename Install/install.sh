#!/bin/sh
# ==============================================
# Полная установка Entware Manager на роутер
# ==============================================

# Цвета (ANSI, совместимо с BusyBox)
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m' # No Color

STEP=0
ERRORS=""

step() {
	STEP=$((STEP + 1))
	echo ""
	echo "${BOLD}[${STEP}/7] $1${NC}"
	echo "────────────────────────────────────────"
}

fail() {
	echo "${RED}  ✗ $1${NC}"
	ERRORS="$ERRORS\n  [$STEP] $1"
}

ok() {
	echo "${GREEN}  ✓ $1${NC}"
}

warn() {
	echo "${YELLOW}  ⚠ $1${NC}"
}

SELF_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TARGET_DIR="/opt/web_entware"
LIGHTTPD_CONF="/opt/etc/lighttpd/lighttpd.conf"

echo "${BOLD}========================================"
echo " Установка Entware Manager"
echo "========================================${NC}"

# ========== 1. ПРОВЕРКА ИСТОЧНИКА ==========
step "Проверка исходных файлов"

if [ ! -d "$SELF_DIR/lib" ]; then
	fail "Исходный каталог $SELF_DIR не найден"
	echo "  Скопируй папку deploy на роутер и запусти install.sh из неё."
	echo ""
	exit 1
fi
ok "Исходный каталог: $SELF_DIR"

# ========== 2. ПРОВЕРКА ПАКЕТОВ ==========
step "Проверка установленных пакетов"

PACKAGES="\
lighttpd|/opt/sbin/lighttpd
ttyd|/opt/bin/ttyd
htop|/opt/bin/htop
jq|/opt/bin/jq
coreutils-base|/opt/bin/dirname
coreutils-timeout|/opt/bin/timeout
procps-ng|/opt/bin/ps
bridge-utils|/opt/sbin/brctl
ip-full|/opt/sbin/ip
sudo|/opt/bin/sudo
curl|/opt/bin/curl
smartmontools|/opt/sbin/smartctl
smartmontools-drivedb|/opt/share/smartmontools/drivedb.h"

MISSING_PKGS=$(echo "$PACKAGES" | while IFS='|' read -r pkg check_path; do
	[ -z "$pkg" ] && continue
	[ -f "$check_path" ] || [ -x "$check_path" ] || printf "%s " "$pkg"
done)
MISSING_PKGS=$(echo "$MISSING_PKGS" | sed 's/ $//')

if [ -z "$MISSING_PKGS" ]; then
	ok "Все пакеты уже установлены"
else
	warn "Отсутствуют:$MISSING_PKGS"

	step "Установка отсутствующих пакетов"

	if opkg update; then
		ok "Списки пакетов обновлены"
	else
		warn "opkg update не удался, пробуем продолжить"
	fi

	for pkg in $MISSING_PKGS; do
		echo "  → $pkg..."
		if opkg install "$pkg"; then
			ok "$pkg установлен"
		else
			fail "$pkg не установился"
		fi
	done

	# Проверка после установки
	for pkg in $MISSING_PKGS; do
		check_path=$(echo "$PACKAGES" | grep "^${pkg}|" | cut -d'|' -f2)
		if [ -n "$check_path" ] && [ ! -f "$check_path" ] && [ ! -x "$check_path" ]; then
			fail "$pkg — бинарник не найден после установки"
		fi
	done
fi

# ========== 3. НАСТРОЙКА LIGHTTPD ==========
step "Настройка lighttpd"

if [ ! -f "/opt/lib/lighttpd/mod_cgi.so" ]; then
	echo "  → mod_cgi.so нет, устанавливаю lighttpd-mod-cgi..."
	opkg install lighttpd-mod-cgi 2>/dev/null && ok "lighttpd-mod-cgi установлен" || warn "lighttpd-mod-cgi не установился"
fi

# alias.url
sed -i '\|"/entware-manager/"|d' "$LIGHTTPD_CONF" 2>/dev/null
sed -i '\|"/entware-cgi/"|d' "$LIGHTTPD_CONF" 2>/dev/null

if grep -q 'alias\.url' "$LIGHTTPD_CONF" 2>/dev/null; then
	ALIAS_OP="+="
else
	ALIAS_OP="="
fi

cat >> "$LIGHTTPD_CONF" <<EOF
alias.url $ALIAS_OP (
    "/entware-manager/" => "/opt/web_entware/",
    "/entware-cgi/" => "/opt/web_entware/cgi-bin/"
)
EOF
if grep -q '/entware-manager/' "$LIGHTTPD_CONF"; then
	ok "alias.url: /entware-manager/ + /entware-cgi/"
else
	fail "alias.url не добавился в $LIGHTTPD_CONF"
fi

# server.port
grep -q 'server\.port' "$LIGHTTPD_CONF" 2>/dev/null || {
	echo 'server.port = 8087' >> "$LIGHTTPD_CONF"
	ok "server.port = 8087"
}

# server.modules
grep -q 'mod_alias' "$LIGHTTPD_CONF" 2>/dev/null || \
	echo 'server.modules += ( "mod_alias" )' >> "$LIGHTTPD_CONF"
grep -q 'mod_cgi' "$LIGHTTPD_CONF" 2>/dev/null || \
	echo 'server.modules += ( "mod_cgi" )' >> "$LIGHTTPD_CONF"

# Удаляем cgi.execute-x-only из main.conf
sed -i '/cgi\.execute-x-only/d' "$LIGHTTPD_CONF" 2>/dev/null

# 30-cgi.conf
CGI_CONF="/opt/etc/lighttpd/conf.d/30-cgi.conf"
mkdir -p "$(dirname "$CGI_CONF")" 2>/dev/null
cat > "$CGI_CONF" <<'CGIEOF'
cgi.assign = ( ".cgi" => "/bin/sh" )
cgi.execute-x-only = "enable"
CGIEOF
if [ -f "$CGI_CONF" ]; then
	ok "30-cgi.conf: .cgi → /bin/sh, execute-x-only"
else
	fail "30-cgi.conf не создался"
fi

# static-file.exclude-extensions
if grep -q 'static-file\.exclude-extensions' "$LIGHTTPD_CONF" 2>/dev/null; then
	if ! grep -q 'static-file\.exclude-extensions.*\.cgi' "$LIGHTTPD_CONF" 2>/dev/null; then
		sed -i '/static-file\.exclude-extensions = (/s/)$/, ".cgi")/' "$LIGHTTPD_CONF"
	fi
fi

# Валидация
if lighttpd -t -f "$LIGHTTPD_CONF" 2>/dev/null; then
	ok "Конфигурация lighttpd валидна"
else
	fail "Конфигурация lighttpd содержит ошибки"
	echo "    lighttpd -t -f $LIGHTTPD_CONF"
fi

# ========== 4. КОПИРОВАНИЕ ФАЙЛОВ ==========
step "Копирование файлов"

mkdir -p "$TARGET_DIR" || {
	fail "Не удалось создать $TARGET_DIR"
	echo "  Проверь права доступа и место на диске"
}

rm -f "$TARGET_DIR"/cgi-bin/*.cgi 2>/dev/null
rm -f "$TARGET_DIR"/cgi-bin/*/*.cgi 2>/dev/null

cp -a "$SELF_DIR"/* "$TARGET_DIR/"
if [ -f "$TARGET_DIR/version.json" ]; then
	ok "Файлы скопированы в $TARGET_DIR ($(du -sh "$TARGET_DIR" | cut -f1))"
else
	fail "Копирование файлов не удалось — $TARGET_DIR пуст"
fi

# ========== 5. ОПРЕДЕЛЕНИЕ АРХИТЕКТУРЫ ==========
step "Настройка архитектуры"

detect_arch() {
	case "$(uname -m)" in
		aarch64)  echo "arm64" ;;
		armv7l|armv6l|armv5tejl|armv5tel) echo "arm" ;;
		mips)     echo "mips" ;;
		mipsel)   echo "mipsel" ;;
		x86_64|amd64) echo "amd64" ;;
		i[3-6]86) echo "386" ;;
		*)        echo "" ;;
	esac
}

ROUTER_ARCH=$(detect_arch)
GO_DIR="$TARGET_DIR/cgi-bin/go"

if [ -n "$ROUTER_ARCH" ]; then
	ok "Обнаружена архитектура: ${BOLD}$ROUTER_ARCH${NC}"

	if [ -d "$GO_DIR" ]; then
		# Удаляем чужие архитектуры
		for dir in "$GO_DIR"/*/; do
			[ -d "$dir" ] || continue
			arch_name=$(basename "$dir")
			if [ "$arch_name" != "$ROUTER_ARCH" ]; then
				echo "  → удаляю $arch_name"
				rm -rf "$dir"
			fi
		done

		# Перемещаем нужные бинарники на уровень выше
		if [ -d "$GO_DIR/$ROUTER_ARCH" ]; then
			rm -f "$GO_DIR"/entware-*
			mv "$GO_DIR/$ROUTER_ARCH"/* "$GO_DIR/" 2>/dev/null
			rmdir "$GO_DIR/$ROUTER_ARCH"
			ok "Установлены бинарники для $ROUTER_ARCH"
		else
			warn "Бинарники для $ROUTER_ARCH не найдены в $GO_DIR"
		fi
	fi
else
	warn "Не удалось определить архитектуру роутера ($(uname -m))"
	echo "  → оставляю все бинарники, go.cgi выберет подходящий"
fi

# ========== 6. SUDOERS + ПРАВА ==========
step "Настройка sudoers и прав доступа"

if [ -x /opt/bin/sudo ]; then
	if [ ! -f /opt/etc/sudoers.d/entware-smartctl ]; then
		mkdir -p /opt/etc/sudoers.d
		echo 'nobody ALL=(ALL) NOPASSWD: /opt/sbin/smartctl' > /opt/etc/sudoers.d/entware-smartctl
		chmod 440 /opt/etc/sudoers.d/entware-smartctl
		ok "sudoers: nobody → smartctl без пароля"
	else
		ok "sudoers уже настроен"
	fi
else
	warn "sudo не установлен — пропускаю sudoers"
fi

find "$TARGET_DIR/cgi-bin" -type l -exec chmod 755 {} \; 2>/dev/null
chmod 755 "$TARGET_DIR/cgi-bin/go.cgi" 2>/dev/null
[ -d "$TARGET_DIR/cgi-bin/go" ] && chmod 755 "$TARGET_DIR"/cgi-bin/go/* 2>/dev/null
chmod 755 "$TARGET_DIR"/watchdog.sh "$TARGET_DIR"/network_watchdog.sh "$TARGET_DIR"/service_watchdog.sh "$TARGET_DIR"/backup.sh 2>/dev/null
find "$TARGET_DIR" -type f -name "*.sh" -exec chmod 755 {} \; 2>/dev/null
find "$TARGET_DIR" -type f \( -name "*.js" -o -name "*.css" -o -name "*.html" -o -name "*.json" -o -name "*.svg" \) -exec chmod 644 {} \; 2>/dev/null
find "$TARGET_DIR/cgi-bin" -type d -exec chmod 755 {} \; 2>/dev/null
ok "Права доступа установлены"

# ========== 7. ЗАПУСК LIGHTTPD ==========
step "Запуск lighttpd"

if /opt/etc/init.d/S80lighttpd status 2>/dev/null | grep -q running; then
	ok "lighttpd уже запущен"
	echo "  → перезапуск..."
	/opt/etc/init.d/S80lighttpd restart 2>&1
	sleep 1
else
	echo "  → запуск..."
	/opt/etc/init.d/S80lighttpd start 2>&1
	sleep 1
fi

if pgrep -f lighttpd >/dev/null; then
	ok "lighttpd работает"
else
	fail "lighttpd не запустился"
	echo "    Для диагностики: lighttpd -D -f $LIGHTTPD_CONF"
fi

# ========== ИТОГ ==========
echo ""
echo "${BOLD}========================================"
echo " РЕЗУЛЬТАТ УСТАНОВКИ"
echo "========================================${NC}"

if [ -n "$ERRORS" ]; then
	echo ""
	echo "${RED}${BOLD}  ОШИБКИ:${NC}$ERRORS"
	echo ""
fi

echo "${GREEN}  ✓ Архитектура:${NC} $ROUTER_ARCH"
echo "${GREEN}  ✓ Файлы:${NC}    $TARGET_DIR"
echo "${GREEN}  ✓ Статика:${NC}  http://$(hostname):8087/entware-manager/"
echo "${GREEN}  ✓ CGI:${NC}      http://$(hostname):8087/entware-cgi/"
echo ""
echo "  Версия: $(jq -r .version "$TARGET_DIR/version.json" 2>/dev/null || echo 'неизвестна')"
echo ""
echo "  Открой в браузере: http://$(hostname -I | awk '{print $1}'):8087/entware-manager/"
echo ""

if [ -n "$ERRORS" ]; then
	echo "${YELLOW}  Часть шагов завершилась с ошибками (см. выше).${NC}"
	echo "${YELLOW}  Проверь логи:${NC}"
else
	echo "  Если что-то пошло не так:"
fi
echo "    /opt/var/log/lighttpd/error.log"
echo "    /tmp/entware/logs/"
echo ""
