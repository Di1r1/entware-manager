#!/bin/sh
# shellcheck disable=SC3043,SC3037,SC3057,SC1090,SC1091,SC2034
# Di1r1
# ==============================================
# Полная установка Entware Manager на роутер
# ==============================================

RED=$(printf '\033[1;31m')
GREEN=$(printf '\033[1;32m')
YELLOW=$(printf '\033[1;33m')
BOLD=$(printf '\033[1m')
NC=$(printf '\033[0m')

STEP=0
ERRORS=""

# --- Лог ---
LOG_DIR="/tmp/entware/install-logs"
LOG_FILE="$LOG_DIR/install-$(date +%Y%m%d-%H%M%S).log"
mkdir -p "$LOG_DIR" 2>&1 || {
	echo "Ошибка: не удалось создать $LOG_DIR"
	exit 1
}

ESC=$(printf '\033')
log() {
	echo "$1" | sed "s/${ESC}\[[0-9;]*m//g" >> "$LOG_FILE" 2>/dev/null || true
}
log "=== Установка Entware Manager ==="
log "Дата: $(date '+%Y-%m-%d %H:%M:%S')"

if [ ! -f "$LOG_FILE" ]; then
	echo "Ошибка: лог-файл $LOG_FILE не создался"
	exit 1
fi

step() {
	STEP=$((STEP + 1))
	echo ""
	echo "${BOLD}[${STEP}] $1${NC}"
	echo "────────────────────────────────────────"
	log "--- ШАГ $STEP: $1 ---"
}

fail() {
	echo "${RED}  ✗ $1${NC}"
	ERRORS="$ERRORS
  [$STEP] $1"
	log "  ✗ $1"
}

ok() {
	echo "${GREEN}  ✓ $1${NC}"
	log "  ✓ $1"
}

warn() {
	echo "${YELLOW}  ⚠ $1${NC}"
	log "  ⚠ $1"
}

backup_file() {
	local src="$1"
	local label="$2"
	[ ! -f "$src" ] && return
	local rel="${src#/}"
	local dst="$BACKUP_DIR/$rel"
	mkdir -p "$(dirname "$dst")" 2>/dev/null
	cp -a "$src" "$dst"
	ok "  бэкап $label → $dst"
}

SELF_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TARGET_DIR="/opt/web_entware"
BACKUP_DIR="$TARGET_DIR/backup"
BACKUP_DIR="$TARGET_DIR/backup"
LIGHTTPD_CONF="/opt/etc/lighttpd/lighttpd.conf"
CGI_CONF="/opt/etc/lighttpd/conf.d/30-cgi.conf"

echo "${BOLD}========================================"
echo " Установка Entware Manager"
echo "========================================${NC}"
echo ""
echo "  Лог: $LOG_FILE"
echo "  Смотреть: tail -f $LOG_FILE"
echo ""

# ========== 1. ПРОВЕРКА ИСТОЧНИКА ==========
step "Проверка исходных файлов"

if [ ! -d "$SELF_DIR/lib" ]; then
	fail "Исходный каталог $SELF_DIR не найден"
	echo "  Скопируй папку deploy на роутер и запусти install.sh из неё."
	exit 1
fi
ok "Исходный каталог: $SELF_DIR"

# Проверка архива: есть ли все основные папки
for d in cgi-bin lib Install doc; do
	[ -d "$SELF_DIR/$d" ] && ok "  папка $d/" || fail "  папка $d/ отсутствует"
done

# ========== 2. ПРОВЕРКА ПАКЕТОВ ==========
step "Проверка установленных пакетов"

echo "  → обновление списков пакетов..."
if opkg update >/dev/null 2>&1; then
	ok "Списки пакетов обновлены"
else
	warn "opkg update не удался, пробуем продолжить"
fi

PACKAGES="\
lighttpd|/opt/sbin/lighttpd
ttyd|/opt/bin/ttyd
jq|/opt/bin/jq
coreutils|/opt/bin/dirname
coreutils-timeout|/opt/bin/timeout
procps-ng|/opt/bin/ps
bridge|/opt/sbin/brctl
ip-full|/opt/sbin/ip
sudo|/opt/bin/sudo
curl|/opt/bin/curl
bash|/opt/bin/bash
smartmontools|/opt/sbin/smartctl
smartmontools-drivedb|/opt/share/smartmontools/drivedb.h"

MISSING_PKGS=$(echo "$PACKAGES" | while IFS='|' read -r pkg check_path; do
	[ -z "$pkg" ] && continue
	if [ -f "$check_path" ] || [ -x "$check_path" ]; then
		ok "  $pkg — $(basename "$check_path")" >&2
	else
		printf "%s " "$pkg"
	fi
done)
MISSING_PKGS=$(echo "$MISSING_PKGS" | sed 's/ $//')

if [ -z "$MISSING_PKGS" ]; then
	echo ""
	ok "Все пакеты уже установлены"
else
	warn "Отсутствуют:$MISSING_PKGS"
	step "Установка отсутствующих пакетов"

	for pkg in $MISSING_PKGS; do
		echo "  → $pkg..."
		if opkg install "$pkg"; then
			ok "$pkg установлен"
		else
			fail "$pkg не установился"
		fi
	done

	for pkg in $MISSING_PKGS; do
		check_path=$(echo "$PACKAGES" | grep "^${pkg}|" | cut -d'|' -f2)
		if [ -n "$check_path" ] && [ ! -f "$check_path" ] && [ ! -x "$check_path" ]; then
			fail "$pkg — бинарник не найден после установки"
		fi
	done
fi

# ========== 3. БЭКАП СУЩЕСТВУЮЩИХ ФАЙЛОВ ==========
step "Бэкап существующих файлов"

if [ -f "$LIGHTTPD_CONF" ]; then
	backup_file "$LIGHTTPD_CONF" "lighttpd.conf"
fi
if [ -f "$CGI_CONF" ]; then
	backup_file "$CGI_CONF" "30-cgi.conf"
fi

if [ -d "$BACKUP_DIR/opt/etc" ]; then
	ok "Бэкапы сохранены в $BACKUP_DIR"
else
	warn "Нечего бэкапить — чистая установка"
fi

# ========== 4. НАСТРОЙКА LIGHTTPD ==========
step "Настройка lighttpd"

LIGHTTPD_ERR=""

if [ ! -f "/opt/lib/lighttpd/mod_cgi.so" ]; then
	echo "  → mod_cgi.so нет, устанавливаю lighttpd-mod-cgi..."
	opkg install lighttpd-mod-cgi 2>/dev/null && ok "lighttpd-mod-cgi установлен" || LIGHTTPD_ERR="$LIGHTTPD_ERR lighttpd-mod-cgi"
fi

# Удаляем старые строки entware из lighttpd.conf (совместимость с предыдущими версиями)
sed -i '\|"/entware-manager/"|d' "$LIGHTTPD_CONF" 2>/dev/null
sed -i '\|"/entware-cgi/"|d' "$LIGHTTPD_CONF" 2>/dev/null
sed -i '/^[[:space:]]*server\.port[[:space:]]*=.*/d' "$LIGHTTPD_CONF"
sed -i '/cgi\.execute-x-only/d' "$LIGHTTPD_CONF" 2>/dev/null
# Удаляем пустые alias.url = ( ) если образовались
sed -i '/^alias\.url = (\s*)$/d' "$LIGHTTPD_CONF" 2>/dev/null

# Отдельный конфиг — не конфликтует с nfqws и другими пакетами
CONF_FILE="/opt/etc/lighttpd/conf.d/90-entware-manager.conf"
mkdir -p "$(dirname "$CONF_FILE")" 2>/dev/null
cat > "$CONF_FILE" <<'EOF'
server.port = 8087
server.modules += ( "mod_alias" )
server.modules += ( "mod_cgi" )

alias.url += (
    "/entware-manager/" => "/opt/web_entware/",
    "/entware-cgi/" => "/opt/web_entware/cgi-bin/"
)
EOF
if [ -f "$CONF_FILE" ]; then
	ok "90-entware-manager.conf: port, modules, alias"
else
	LIGHTTPD_ERR="$LIGHTTPD_ERR 90-entware-manager.conf"
	fail "90-entware-manager.conf не создался"
fi

# 30-cgi.conf — cgi.assign
CGI_CONF="/opt/etc/lighttpd/conf.d/30-cgi.conf"
mkdir -p "$(dirname "$CGI_CONF")" 2>/dev/null
cat > "$CGI_CONF" <<'CGIEOF'
cgi.assign = ( ".cgi" => "/bin/sh" )
cgi.execute-x-only = "enable"
CGIEOF
if [ -f "$CGI_CONF" ]; then
	ok "30-cgi.conf: .cgi → /bin/sh, execute-x-only"
else
	LIGHTTPD_ERR="$LIGHTTPD_ERR 30-cgi.conf"
	fail "30-cgi.conf не создался"
fi

# Добавляем .cgi в static-file.exclude-extensions в светеотдельный конфиг
# (уже есть в lighttpd.conf, но на всякий случай дублируем в наш файл)
if ! grep -q 'static-file\.exclude-extensions.*\.cgi' "$LIGHTTPD_CONF" 2>/dev/null; then
	if grep -q 'static-file\.exclude-extensions' "$LIGHTTPD_CONF" 2>/dev/null; then
		sed -i '/static-file\.exclude-extensions = (/s/)$/, ".cgi")/' "$LIGHTTPD_CONF"
	fi
fi

if lighttpd -t -f "$LIGHTTPD_CONF" 2>/dev/null; then
	ok "Конфигурация lighttpd валидна"
else
	LIGHTTPD_ERR="$LIGHTTPD_ERR invalid-conf"
	fail "Конфигурация lighttpd содержит ошибки"
	echo "    lighttpd -t -f $LIGHTTPD_CONF"
fi

if [ -n "$LIGHTTPD_ERR" ]; then
	fail "Проблемы с lighttpd:$LIGHTTPD_ERR"
fi

# ========== 5. КОПИРОВАНИЕ ФАЙЛОВ ==========
step "Копирование файлов"

if [ "$SELF_DIR" = "$TARGET_DIR" ]; then
	ok "Файлы уже на месте (установка через ipk)"
else
	mkdir -p "$TARGET_DIR" || {
		fail "Не удалось создать $TARGET_DIR"
	}

	rm -f "$TARGET_DIR"/cgi-bin/*.cgi 2>/dev/null
	rm -f "$TARGET_DIR"/cgi-bin/*/*.cgi 2>/dev/null

	cp -a "$SELF_DIR"/* "$TARGET_DIR/"
fi
if [ -f "$TARGET_DIR/version.json" ]; then
	VERSION=$(jq -r .version "$TARGET_DIR/version.json" 2>/dev/null || echo '?')
	ok "Файлы скопированы в $TARGET_DIR (версия $VERSION, $(du -sh "$TARGET_DIR" | cut -f1))"
else
	fail "Копирование файлов не удалось — $TARGET_DIR пуст"
fi

rm -f "$TARGET_DIR/DEVLOG.md" "$TARGET_DIR/DEVICE.md" "$TARGET_DIR/BUILD.md" "$TARGET_DIR/RULES.md" "$TARGET_DIR/TECH_SPEC.md" "$TARGET_DIR/forum_post.md" "$TARGET_DIR/Install/Install.txt" "$TARGET_DIR/doc/NETWORK_PROMPT.md" "$TARGET_DIR/doc/IPK_BUILD.md" "$TARGET_DIR/conffiles" "$TARGET_DIR/control" "$TARGET_DIR/postinst" "$TARGET_DIR/prerm" 2>/dev/null || true

# ========== 6. ОПРЕДЕЛЕНИЕ АРХИТЕКТУРЫ ==========
step "Настройка архитектуры"

detect_arch() {
	local arch
	arch=$(opkg print-architecture 2>/dev/null | awk '/^arch /{print $2}' | grep -v '^all$\|^noarch$' | head -1)
	[ -n "$arch" ] && echo "$arch" | sed 's/-[^-]*$//; s/aarch64/arm64/; s/x86_64/amd64/; s/i[3-6]86/386/' && return

	case "$(uname -m)" in
		aarch64)  echo "arm64" ;;
		armv7l|armv6l|armv5tejl|armv5tel) echo "arm" ;;
		mips)
			ELF="/opt/bin/opkg"
			[ ! -f "$ELF" ] && ELF="/bin/sh"
			byte=$(dd if="$ELF" bs=1 count=6 2>/dev/null | od -b | head -1 | awk '{print $7}')
			[ "$byte" = "001" ] && echo "mipsel" || echo "mips"
			;;
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
		for dir in "$GO_DIR"/*/; do
			[ -d "$dir" ] || continue
			arch_name=$(basename "$dir")
			if [ "$arch_name" != "$ROUTER_ARCH" ]; then
				echo "  → удаляю $arch_name"
				rm -rf "$dir"
			fi
		done
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
fi

# ========== 7. SUDOERS + ПРАВА ==========
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
[ -d "$TARGET_DIR/cgi-bin/go" ] && chmod -R 755 "$TARGET_DIR"/cgi-bin/go/* 2>/dev/null
chmod 755 "$TARGET_DIR"/watchdog.sh "$TARGET_DIR"/network_watchdog.sh "$TARGET_DIR"/service_watchdog.sh "$TARGET_DIR"/backup.sh 2>/dev/null
find "$TARGET_DIR" -type f -name "*.sh" -exec chmod 755 {} \; 2>/dev/null
find "$TARGET_DIR" -type f \( -name "*.js" -o -name "*.css" -o -name "*.html" -o -name "*.json" -o -name "*.svg" \) -exec chmod 644 {} \; 2>/dev/null
find "$TARGET_DIR/cgi-bin" -type d -exec chmod 755 {} \; 2>/dev/null
ok "Права доступа установлены"

# ========== 8. ЗАПУСК LIGHTTPD ==========
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

# ========== 9. ПРОВЕРКА УСТАНОВКИ ==========
step "Проверка установки"

CHECK_ERRS=""

# Проверка пакетов
echo "  ${BOLD}Пакеты:${NC}"
for pkg in $PACKAGES; do
	check_path=$(echo "$pkg" | cut -d'|' -f2)
	pkg_name=$(echo "$pkg" | cut -d'|' -f1)
	if [ -f "$check_path" ] || [ -x "$check_path" ]; then
		ok "  $pkg_name ($(basename "$check_path"))"
	else
		CHECK_ERRS="$CHECK_ERRS
    ✗ $pkg_name — не найден $check_path"
		echo "  ${RED}✗ $pkg_name${NC}"
		log "  ✗ $pkg_name — не найден $check_path"
	fi
done

# Проверка go.cgi
echo "  ${BOLD}Диспетчер:${NC}"
if [ -x "$TARGET_DIR/cgi-bin/go.cgi" ]; then
	ok "  go.cgi ($(wc -l < "$TARGET_DIR/cgi-bin/go.cgi") строк)"
else
	CHECK_ERRS="$CHECK_ERRS
    ✗ go.cgi — не найден или не исполняемый"
	fail "  go.cgi не найден"
fi

# Проверка симлинков .cgi
echo "  ${BOLD}Симлинки .cgi:${NC}"
CGI_COUNT=$(find "$TARGET_DIR/cgi-bin" -maxdepth 1 -name "*.cgi" ! -name "go.cgi" | wc -l)
SYMLINK_OK=0
SYMLINK_BAD=0
for f in "$TARGET_DIR"/cgi-bin/*.cgi; do
	[ "$f" = "$TARGET_DIR/cgi-bin/go.cgi" ] && continue
	[ -f "$f" ] || continue
	if [ -L "$f" ]; then
		target=$(readlink "$f")
		echo "    $f → $target"
		SYMLINK_OK=$((SYMLINK_OK + 1))
	else
		CHECK_ERRS="$CHECK_ERRS
    ✗ $f — не симлинк"
		fail "  $f — не симлинк"
		SYMLINK_BAD=$((SYMLINK_BAD + 1))
	fi
done
for d in monitor network logger service_watchdog; do
	dir="$TARGET_DIR/cgi-bin/$d"
	[ -d "$dir" ] || continue
	for f in "$dir"/*.cgi; do
		[ -f "$f" ] || continue
		if [ -L "$f" ]; then
			target=$(readlink "$f")
			SYMLINK_OK=$((SYMLINK_OK + 1))
		else
			CHECK_ERRS="$CHECK_ERRS
    ✗ $f — не симлинк"
			fail "  $f — не симлинк"
			SYMLINK_BAD=$((SYMLINK_BAD + 1))
		fi
	done
done
if [ $SYMLINK_BAD -eq 0 ]; then
	ok "  $SYMLINK_OK симлинков, все целые"
else
	fail "  $SYMLINK_BAD симлинков биты"
fi

# Проверка Go-бинарников
echo "  ${BOLD}Go-бинарники:${NC}"
GO_BINS="entware-logger entware-monitor entware-net entware-pkg entware-services entware-smart entware-stats"
GO_OK=0
for bin in $GO_BINS; do
	if [ -x "$TARGET_DIR/cgi-bin/go/$bin" ]; then
		echo "    $bin ($(du -h "$TARGET_DIR/cgi-bin/go/$bin" | cut -f1))"
		GO_OK=$((GO_OK + 1))
	else
		CHECK_ERRS="$CHECK_ERRS
    ✗ $bin — не найден"
		fail "  $bin не найден в cgi-bin/go/"
	fi
done
if [ $GO_OK -eq 7 ]; then
	ok "  Все 7 бинарников ($GO_OK)"
else
	fail "  Найдено $GO_OK из 7 бинарников"
fi

# Проверка веб-файлов
echo "  ${BOLD}Веб-файлы:${NC}"
for f in index.html style.css entware.js icons.svg version.json; do
	if [ -f "$TARGET_DIR/$f" ]; then
		ok "  $f"
	else
		CHECK_ERRS="$CHECK_ERRS
    ✗ $f — не найден"
		fail "  $f не найден"
	fi
done

# Проверка lighttpd
echo "  ${BOLD}Lighttpd:${NC}"
if pgrep -f lighttpd >/dev/null; then
	PID=$(pgrep -f lighttpd | head -1)
	ok "  lighttpd (PID $PID)"
else
	CHECK_ERRS="$CHECK_ERRS
    ✗ lighttpd не запущен"
	fail "  lighttpd не запущен"
fi

# Проверка HTTP-ответа
LIGHTTPD_PORT=$(grep 'server\.port' "$LIGHTTPD_CONF" 2>/dev/null | grep -o '[0-9]*' | head -1)
LIGHTTPD_PORT=${LIGHTTPD_PORT:-8087}
echo "  ${BOLD}HTTP-ответ:${NC}"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 2 "http://127.0.0.1:$LIGHTTPD_PORT/entware-cgi/version.cgi" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
	ok "  HTTP 200 (127.0.0.1:$LIGHTTPD_PORT/entware-cgi/version.cgi)"
else
	CHECK_ERRS="$CHECK_ERRS
    ✗ HTTP $HTTP_CODE"
	fail "  HTTP $HTTP_CODE (127.0.0.1:$LIGHTTPD_PORT/entware-cgi/version.cgi)"
fi

if [ -n "$CHECK_ERRS" ]; then
	echo ""
	echo "${RED}  ✗ Проверка установки выявила ошибки:${NC}"
fi

# ========== ИТОГ ==========
echo ""
echo "${BOLD}========================================"
echo " РЕЗУЛЬТАТ УСТАНОВКИ"
echo "========================================${NC}"

if [ -n "$ERRORS" ]; then
	echo ""
	echo "${RED}${BOLD}  ОШИБКИ В ХОДЕ УСТАНОВКИ:${NC}"
	echo "$ERRORS" | while IFS= read -r line; do
		[ -n "$line" ] && echo "  $line"
	done
fi

if [ -n "$CHECK_ERRS" ]; then
	echo ""
	echo "${RED}${BOLD}  ОШИБКИ ПРОВЕРКИ:${NC}"
	echo "$CHECK_ERRS" | while IFS= read -r line; do
		[ -n "$line" ] && echo "$line"
	done
fi

echo ""
if [ -z "$ERRORS" ] && [ -z "$CHECK_ERRS" ]; then
	echo "${GREEN}${BOLD}  УСТАНОВКА ЗАВЕРШЕНА УСПЕШНО${NC}"
	echo "$ROUTER_ARCH" > "$TARGET_DIR/.arch" 2>/dev/null
	ok "Архитектура сохранена в .arch"
fi
echo ""
echo "${GREEN}  ✓ Архитектура:${NC} $ROUTER_ARCH"
echo "${GREEN}  ✓ Файлы:${NC}    $TARGET_DIR"
echo "${GREEN}  ✓ Статика:${NC}  http://$(hostname):8087/entware-manager/"
echo "${GREEN}  ✓ CGI:${NC}      http://$(hostname):8087/entware-cgi/"
echo ""
echo "  Версия: $(jq -r .version "$TARGET_DIR/version.json" 2>/dev/null || echo 'неизвестна')"
echo ""
IP=$(ip -o -4 addr show br0 2>/dev/null | awk '{print $4}' | cut -d/ -f1)
[ -z "$IP" ] && IP=$(ip route get 8.8.8.8 2>/dev/null | awk '{print $NF}' | head -1)
echo "  Открой в браузере: http://${IP:-<IP-роутера>}:8087/entware-manager/"
echo ""
echo "  Терминал: Настройки → Терминал → Запустить"
echo ""
echo "  Лог установки: $LOG_FILE"
echo ""

if [ -n "$ERRORS" ] || [ -n "$CHECK_ERRS" ]; then
	echo "${YELLOW}  Были ошибки (см. выше). Исправь и запусти заново.${NC}"
	echo "${YELLOW}  Лог: $LOG_FILE${NC}"
fi
