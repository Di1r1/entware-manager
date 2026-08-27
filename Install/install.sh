#!/bin/sh
# shellcheck disable=SC3043,SC3037,SC3057,SC1090,SC1091,SC2034
# Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
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

# --- Лог (единый файл с ротацией по размеру) ---
LOG_DIR="/tmp/entware/install-logs"
LOG_FILE="$LOG_DIR/install.log"
MAX_LOG_SIZE=524288
KEEP_LOG_SIZE=262144
mkdir -p "$LOG_DIR" 2>&1 || {
	echo "Ошибка: не удалось создать $LOG_DIR"
	exit 1
}

# Миграция: старый формат install-*.log больше не используется
find "$LOG_DIR" -maxdepth 1 -type f -name 'install-*.log' -delete 2>/dev/null

# Ротация: если install.log разросся — оставить последние KEEP_LOG_SIZE байт
if [ -f "$LOG_FILE" ] && [ "$(wc -c < "$LOG_FILE" 2>/dev/null || echo 0)" -gt "$MAX_LOG_SIZE" ]; then
	tail -c "$KEEP_LOG_SIZE" "$LOG_FILE" > "$LOG_FILE.tmp" 2>/dev/null && mv -f "$LOG_FILE.tmp" "$LOG_FILE"
fi

ESC=$(printf '\033')
log() {
	echo "$1" | sed "s/${ESC}\[[0-9;]*m//g" >> "$LOG_FILE" 2>/dev/null || true
}
echo "" >> "$LOG_FILE" 2>/dev/null || true
echo "============================================" >> "$LOG_FILE" 2>/dev/null || true
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

# --- Проверка что менеджер отвечает по HTTP ---
lighttpd_http_ok() {
	local port="${1:-8087}"
	command -v curl >/dev/null 2>&1 || return 1
	local code
	# session.cgi открыт без авторизации (в отличие от version.cgi, который
	# после внедрения логина отдаёт 401 без cookie).
	code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 2 "http://127.0.0.1:$port/entware-cgi/session.cgi" 2>/dev/null)
	[ "$code" = "200" ]
}

# --- Есть ли хоть какой-то процесс lighttpd (в т.ч. чужой, напр. zapret) ---
# BusyBox-безопасная проверка по /proc (без pgrep/kill -0, см. RULES §9).
lighttpd_pid() {
	for p in /proc/[0-9]*/cmdline; do
		if tr '\0' ' ' < "$p" 2>/dev/null | grep -q "lighttpd"; then
			echo "$p" | sed 's#/proc/##; s#/cmdline##'
			return 0
		fi
	done
	return 1
}

any_lighttpd_running() {
	[ -n "$(lighttpd_pid)" ]
}

# --- Жив ли наш entware-server (по pid-файлу + имени процесса) ---
entware_server_running() {
	[ -f /opt/var/run/entware-server.pid ] || return 1
	pid=$(cat /opt/var/run/entware-server.pid 2>/dev/null | tr -d ' ')
	[ -n "$pid" ] && [ -d "/proc/$pid" ] || return 1
	cmdline=$(tr '\0' ' ' < "/proc/$pid/cmdline" 2>/dev/null)
	echo "$cmdline" | grep -q "entware-server" || return 1
	return 0
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

# --- Умный датированный бэкап (идемпотентный) ---
# Копирует файл в /opt/web_entware/backup/<YYYY-MM-DD>/<путь>, но только
# один раз в сутки (если копия за сегодня уже есть — пропускаем, чтобы не
# копить дубли при повторных установках).
backup_file_dated() {
	local src="$1"
	local label="$2"
	[ ! -f "$src" ] && return
	local today rel dst
	today=$(date '+%Y-%m-%d' 2>/dev/null)
	[ -z "$today" ] && today="unknown"
	rel="${src#/}"
	dst="$TARGET_DIR/backup/$today/$rel"
	[ -f "$dst" ] && { ok "  бэкап $label уже есть за $today (пропуск)"; return; }
	mkdir -p "$(dirname "$dst")" 2>/dev/null
	cp -a "$src" "$dst"
	ok "  бэкап $label → $dst"
}

# --- Наш ли это 30-cgi.conf (точное совпадение с нашим шаблоном)? ---
# Наш файл — ровно 2 строки: cgi.assign = ( ".cgi" => "/bin/sh" )
# и cgi.execute-x-only = "enable". Если там есть что-то ещё (perl/ruby/python/
# php и т.п.) — это ЧУЖОЙ файл, трогать его нельзя.
is_our_cgi_conf() {
	[ ! -f "$1" ] && return 1
	# любая чужая директива (другой интерпретатор или лишние строки) → не наш
	if grep -Eq 'perl|ruby|python|php|\.pl|\.rb|\.erb|\.py|\.php' "$1" 2>/dev/null; then
		return 1
	fi
	local lines
	lines=$(wc -l < "$1" 2>/dev/null || echo 0)
	[ "$lines" -le 3 ] || return 1
	grep -q 'cgi\.assign.*\.cgi.*/bin/sh' "$1" 2>/dev/null || return 1
	return 0
}

SELF_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TARGET_DIR="/opt/web_entware"
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

# --- Lock против параллельного запуска install.sh ---
# Два install.sh одновременно писали бы в один TARGET_DIR (гонка). flock в
# BusyBox-сборках нет, поэтому используем атомарный mkdir + PID-файл:
# если процесс жив — отказываемся, если lock битый (процесс умер) — забираем.
# В postinst-контексте opkg сам держит свой lock — наш не нужен.
LOCK_DIR="/opt/var/run/entware-install.lock.d"
if [ "$OPKG_POSTINST" != "1" ]; then
	mkdir -p /opt/var/run 2>/dev/null
	if ! mkdir "$LOCK_DIR" 2>/dev/null; then
		LOCK_PID=""
		[ -f "$LOCK_DIR/pid" ] && LOCK_PID=$(cat "$LOCK_DIR/pid" 2>/dev/null)
		if [ -n "$LOCK_PID" ] && [ -d "/proc/$LOCK_PID" ]; then
			echo "${RED}  ✗ Установка уже выполняется (PID $LOCK_PID). Дождись завершения.${NC}"
			echo "    Лог: $LOG_FILE"
			exit 1
		fi
		# Lock битый (процесс умер) — забираем
		rm -rf "$LOCK_DIR" 2>/dev/null
		mkdir "$LOCK_DIR" 2>/dev/null || {
			echo "${RED}  ✗ Не удалось создать lock $LOCK_DIR${NC}"
			exit 1
		}
	fi
	echo "$$" > "$LOCK_DIR/pid" 2>/dev/null
	trap 'rm -rf "$LOCK_DIR" 2>/dev/null; [ -n "${STAGE_DIR:-}" ] && rm -rf "$STAGE_DIR" 2>/dev/null' EXIT INT TERM
fi

# --- opkg с таймаутом (защита от зависания при недоступном/медленном feed) ---
# coreutils-timeout уже в зависимостях; если ещё не установлен — без таймаута.
opkg_t() {
	if command -v timeout >/dev/null 2>&1; then
		timeout 60 opkg "$@"
	else
		opkg "$@"
	fi
}

# ========== 0. ОБНОВЛЕНИЕ СПИСКОВ ПАКЕТОВ ==========
# Вынесено в самое начало: это единственная операция, ходящая в интернет.
# Если feed недоступен — install.sh падает/продолжает ДО любых изменений системы.
# В postinst-контексте opkg уже держит lock на текущую установку — вложенный
# opkg update/install привёл бы к self-deadlock, поэтому списки НЕ обновляем
# (они свежие, т.к. только что пришёл ipk, а модули ставятся через Depends).
NET_FAILED=""
if [ "$OPKG_POSTINST" = "1" ]; then
	ok "Списки пакетов уже обновлены (установка через ipk)"
else
	step "Обновление списков пакетов"
	echo "  → opkg update (таймаут 60с)..."
	if opkg_t update >/dev/null 2>&1; then
		ok "Списки пакетов обновлены"
	else
		warn "opkg update не удался или таймаут — пробуем продолжить"
		NET_FAILED="1"
	fi
fi

# ========== 1. ОЧИСТКА СТАРЫХ ВЕРСИЙ ==========
step "Очистка старых версий"
# Удаляем артефакты предыдущих установок старше 1 дня (кроме текущей SELF_DIR и ручного бэкапа)
CLEAN_DIR="/opt/tmp"
if [ -d "$CLEAN_DIR" ]; then
	DELETED=0
	for pattern in 'entware-manager-*.tar.gz' 'entware-manager_*.ipk' 'deploy_old*' 'deploy*'; do
		for f in "$CLEAN_DIR"/$pattern; do
			[ -e "$f" ] || continue
			# не трогаем текущий каталог установки и его содержимое
			case "$f" in
				"$SELF_DIR"|"$SELF_DIR"/*) continue ;;
			esac
			# чистим только то, что старше 1 дня
			if [ "$(find "$f" -maxdepth 0 -mtime +1 | wc -l)" -gt 0 ]; then
				rm -rf "$f" 2>/dev/null && DELETED=$((DELETED + 1)) && ok "  удалено: $f"
			fi
		done
	done
	[ "$DELETED" -eq 0 ] && ok "старые версии не найдены (или свежие, младше 1 дня)"
else
	ok "каталог $CLEAN_DIR не существует — пропуск"
fi

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

# Целостность источника: version.json непустой (битый/обрезанный архив ловим до изменений системы)
if [ ! -s "$SELF_DIR/version.json" ]; then
	fail "version.json отсутствует или пуст — исходные файлы повреждены (битый/обрезанный архив)"
	exit 1
fi
ok "  version.json: $(jq -r .version "$SELF_DIR/version.json" 2>/dev/null || echo '?')"

# ========== 3. ПРОВЕРКА ПАКЕТОВ ==========
step "Проверка установленных пакетов"

PACKAGES="\
lighttpd|/opt/sbin/lighttpd
lighttpd-mod-proxy|/opt/lib/lighttpd/mod_proxy.so
lighttpd-mod-deflate|/opt/lib/lighttpd/mod_deflate.so
lighttpd-mod-access|/opt/etc/lighttpd/conf.d/30-access.conf
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
	if [ "$OPKG_POSTINST" = "1" ]; then
		# postinst выполняется ПОД lock'ом opkg: вложенный opkg install =
		# self-deadlock. Все зависимости должны быть покрыты Depends ipk;
		# если что-то не пришло — только предупреждение, чистку даст
		# повторный install.sh после установки пакета.
		warn "не могу установить в postinst-режиме (opkg держит lock) — проверь Depends, повтори install.sh позже"
	else
		step "Установка отсутствующих пакетов"

		for pkg in $MISSING_PKGS; do
			echo "  → $pkg..."
			if opkg_t install "$pkg"; then
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
fi

# ========== 3. БЭКАП СУЩЕСТВУЮЩИХ ФАЙЛОВ ==========
step "Бэкап существующих файлов"

if [ -f "$LIGHTTPD_CONF" ]; then
	backup_file "$LIGHTTPD_CONF" "lighttpd.conf"
fi
if [ -f "$CGI_CONF" ]; then
	# 30-cgi.conf — общий файл lighttpd (может принадлежать web4static/nfqws2).
	# Бэкапим с датой (идемпотентно), сам файл НЕ перезаписываем — см. ниже.
	backup_file_dated "$CGI_CONF" "30-cgi.conf"
fi

if [ -d "$BACKUP_DIR/opt/etc" ]; then
	ok "Бэкапы сохранены в $BACKUP_DIR"
else
	warn "Нечего бэкапить — чистая установка"
fi

# ========== 4. ОПРЕДЕЛЕНИЕ РЕЖИМА ВЕБ-СЕРВЕРА ==========
# Режимы:
#   go (по умолчанию) — собственный entware-server на 8087 (Variant 1).
#   lighttpd — запасной путь через EWM_MODE=lighttpd (обратная совместимость).
# Миграционные функции порт-хранителя общего lighttpd.
MIGRATE_LIB=""
[ -f "$SELF_DIR/lib/migrate.sh" ] && MIGRATE_LIB="$SELF_DIR/lib/migrate.sh"
[ -z "$MIGRATE_LIB" ] && [ -f "/opt/web_entware/lib/migrate.sh" ] && MIGRATE_LIB="/opt/web_entware/lib/migrate.sh"
if [ -n "$MIGRATE_LIB" ]; then
	. "$MIGRATE_LIB"
else
	warn "lib/migrate.sh не найден — миграция порт-хранителя недоступна"
fi
step "Определение режима веб-сервера"

if [ "${EWM_MODE:-go}" = "lighttpd" ]; then
	WEB_PATH="lighttpd"
	ok "EWM_MODE=lighttpd — принудительно lighttpd (запасной путь)"
elif lighttpd_http_ok 8087; then
	if entware_server_running; then
		WEB_PATH="go"
		ok "entware-server уже отвечает на 127.0.0.1:8087"
	else
		WEB_PATH="go"
		ok "обнаружен прежний lighttpd-режим — переходим на entware-server (Variant 1)"
	fi
else
	WEB_PATH="go"
	ok "использую собственный entware-server (порт 8087)"
fi
echo "  ${YELLOW}→ режим: $WEB_PATH${NC}"

# Удаляем старый init-скрипт entware-lighttpd (заменён entware-server).
rm -f /opt/etc/init.d/S80entware-lighttpd 2>/dev/null

# ========== 5. НАСТРОЙКА ВЕБ-СЕРВЕРА ==========
step "Настройка веб-сервера"

LIGHTTPD_ERR=""

if [ "$WEB_PATH" = "lighttpd" ]; then

if [ ! -f "/opt/lib/lighttpd/mod_cgi.so" ]; then
	if [ "$OPKG_POSTINST" = "1" ]; then
		warn "lighttpd-mod-cgi отсутствует — в postinst не ставлю (opkg lock), проверь Depends"
		LIGHTTPD_ERR="$LIGHTTPD_ERR lighttpd-mod-cgi"
	else
		echo "  → mod_cgi.so нет, устанавливаю lighttpd-mod-cgi..."
		opkg_t install lighttpd-mod-cgi 2>/dev/null && ok "lighttpd-mod-cgi установлен" || LIGHTTPD_ERR="$LIGHTTPD_ERR lighttpd-mod-cgi"
	fi
fi

if [ ! -f "/opt/lib/lighttpd/mod_proxy.so" ]; then
	if [ "$OPKG_POSTINST" = "1" ]; then
		warn "lighttpd-mod-proxy отсутствует — в postinst не ставлю (opkg lock), проверь Depends"
		LIGHTTPD_ERR="$LIGHTTPD_ERR lighttpd-mod-proxy"
	else
		echo "  → mod_proxy.so нет, устанавливаю lighttpd-mod-proxy..."
		opkg_t install lighttpd-mod-proxy 2>/dev/null && ok "lighttpd-mod-proxy установлен" || LIGHTTPD_ERR="$LIGHTTPD_ERR lighttpd-mod-proxy"
	fi
fi

# Удаляем старые строки entware из lighttpd.conf (совместимость с предыдущими версиями)
sed -i '\|"/entware-manager/"|d' "$LIGHTTPD_CONF" 2>/dev/null
sed -i '\|"/entware-cgi/"|d' "$LIGHTTPD_CONF" 2>/dev/null
sed -i '/^[[:space:]]*server\.port[[:space:]]*=.*/d' "$LIGHTTPD_CONF"
sed -i '/cgi\.execute-x-only/d' "$LIGHTTPD_CONF" 2>/dev/null
# Удаляем пустые alias.url блоки (одно- и многострочные), оставшиеся от старых версий.
# Непустые пользовательские alias.url не трогаем.
awk '
/^alias\.url (=|\+=) \([[:space:]]*\)$/ { next }
/^alias\.url (=|\+=) \($/ {
    start = $0
    inb = 1
    empty = 1
    next
}
inb {
    if ($0 ~ /^\)$/) {
        if (empty) { inb = 0; next }
        if (start != "") { print start; start = "" }
        inb = 0
        print
        next
    }
    if (empty) {
        print start
        start = ""
        empty = 0
    }
    print
    next
}
{ print }
' "$LIGHTTPD_CONF" > "$LIGHTTPD_CONF.ewm" 2>/dev/null && mv "$LIGHTTPD_CONF.ewm" "$LIGHTTPD_CONF"

# Отдельный конфиг — не конфликтует с nfqws и другими пакетами
CONF_FILE="/opt/etc/lighttpd/conf.d/90-entware-manager.conf"
mkdir -p "$(dirname "$CONF_FILE")" 2>/dev/null
cat > "$CONF_FILE" <<'EOF'
server.port = 8087
server.modules += ( "mod_alias" )
server.modules += ( "mod_cgi" )
server.modules += ( "mod_proxy" )
server.modules += ( "mod_access" )

alias.url += (
    "/entware-manager/" => "/opt/web_entware/",
    "/entware-cgi/" => "/opt/web_entware/cgi-bin/"
)

# Защита статики: нельзя скачивать скрипты/секретные конфиги из /entware-manager/.
# rdp_config.json, version.json, menu.json, system_sources.json отдаются (публичные).
# (В go-режиме это обеспечивает whitelist.) /entware-cgi/*.cgi НЕ блокируем — это эндпоинты.
$HTTP["url"] =~ "^/entware-manager/" {
    url.access-deny = ( "~", ".sh", ".conf", ".md", ".cgi" )
    # Go-бинарники (без расширения) — не отдаём как статику: иначе их можно скачать.
    $HTTP["url"] =~ "^/entware-manager/cgi-bin/go/" {
        url.access-deny = ( "" )
    }
    $HTTP["url"] =~ "^/entware-manager/(auth_config|server_config|service_config|monitor_config|network_config|links|logger/config|telegram_config|rdp_config)\.json$" {
        url.access-deny = ( "" )
    }
    # Мост сервисов: манифесты и СЕКРЕТНЫЕ файлы (auth.json/prefs/state) —
    # не отдаём как статику вообще.
    $HTTP["url"] =~ "^/entware-manager/bridge/" {
        url.access-deny = ( "" )
    }
    # Init-скрипты (S90grdp-proxy, S80entware-server, install.sh) — не статика.
    $HTTP["url"] =~ "^/entware-manager/Install/" {
        url.access-deny = ( "" )
    }
}

# Второй alias "/entware-cgi/" => cgi-bin/: Go-бинарники (без расширения)
# не попадают в static-file.exclude-extensions и отдавались бы как статика.
$HTTP["url"] =~ "^/entware-cgi/go/" {
    url.access-deny = ( "" )
}

# CGI-диспетчер Entware Manager исполняется ТОЛЬКО в /entware-cgi/ через
# ЛОКАЛЬНЫЙ cgi.assign (пустой extension = любой файл по shebang). Глобальный
# файл /opt/etc/lighttpd/conf.d/30-cgi.conf не трогаем — он может принадлежать
# web4static/nfqws2 (perl/ruby/py/php) и его перезапись их ломает.
$HTTP["url"] =~ "^/entware-cgi/" {
    cgi.assign = ( "" => "" )
}

# Встроенные сервисы через тот же origin (удалённый доступ + HTTPS)
# Порт RDP-прокси берём из rdp_config.json (плейсхолдер заменяется sed ниже),
# чтобы не конфликтовать с другими сервисами (например, AWG на 9099).
proxy.header = ( "upgrade" => "enable" )
$HTTP["url"] =~ "^/terminal/" {
    proxy.server = ( "" => ( ( "host" => "127.0.0.1", "port" => 9089 ) ) )
}
$HTTP["url"] =~ "^/htop/" {
    proxy.server = ( "" => ( ( "host" => "127.0.0.1", "port" => 8089 ) ) )
}
$HTTP["url"] =~ "^/rdp/" {
    proxy.server = ( "" => ( ( "host" => "127.0.0.1", "port" => __RDP_PORT__ ) ) )
}
$HTTP["url"] =~ "^/ws" {
    proxy.server = ( "" => ( ( "host" => "127.0.0.1", "port" => __RDP_PORT__ ) ) )
}
EOF
if [ -f "$CONF_FILE" ]; then
	ok "90-entware-manager.conf: port, modules, alias"
else
	LIGHTTPD_ERR="$LIGHTTPD_ERR 90-entware-manager.conf"
	fail "90-entware-manager.conf не создался"
fi

# Подставляем RDP-порт СРАЗУ после записи конфига и ДО валидации lighttpd.
# Иначе lighttpd -t падает: «Undefined config variable: var.__RDP_PORT__»
# (MAJOR, найден полным циклом go↔lighttpd: EWM_MODE=lighttpd давал нерабочую
# панель на 8087). Источник порта — существующий rdp_config.json (переживает
# установку), иначе дефолт 9099; блок ШАГ 8 ниже повторяет подстановку
# идемпотентно (для свежей установки порт совпадает с дефолтом).
RDP_CFG_EARLY="$TARGET_DIR/rdp_config.json"
RP_EARLY=$(jq -r '.proxy_port // 9099' "$RDP_CFG_EARLY" 2>/dev/null || echo 9099)
case "$RP_EARLY" in ''|*[!0-9]*) RP_EARLY=9099 ;; esac
sed -i "s/__RDP_PORT__/$RP_EARLY/g" "$CONF_FILE" 2>/dev/null
ok "lighttpd-conf: RDP-порт $RP_EARLY подставлен до валидации"

# Сжатие статики (mod_deflate): WASM-клиент RDP ~10МБ → ~3МБ при передаче.
# Расширяем deflate.mimetypes в системном 30-deflate.conf (если есть).
DEFLATE_CONF="/opt/etc/lighttpd/conf.d/30-deflate.conf"
if [ -f "$DEFLATE_CONF" ] && [ -f "/opt/lib/lighttpd/mod_deflate.so" ]; then
	if grep -q "^deflate.mimetypes" "$DEFLATE_CONF"; then
		sed -i 's#^deflate.mimetypes.*#deflate.mimetypes         = ("text/plain", "text/html", "text/css", "application/javascript", "application/json", "image/svg+xml", "application/wasm")#' "$DEFLATE_CONF" 2>/dev/null
	else
		echo 'deflate.mimetypes = ("text/plain", "text/html", "text/css", "application/javascript", "application/json", "image/svg+xml", "application/wasm")' >> "$DEFLATE_CONF"
	fi
	ok "mod_deflate: сжатие статики/WASM включено"
else
	ok "mod_deflate не установлен — сжатие статики пропущено"
fi

# Корень сервера: meta-refresh на панель (для Keenetic Remote/KeenDNS,
# которые публикуют корневой URL и отдают 403 на пустой document-root).
DOC_ROOT=$(grep -o "var.server_root = \"[^\"]*\"" "$LIGHTTPD_CONF" 2>/dev/null | cut -d'"' -f2)
if [ -z "$DOC_ROOT" ]; then
	DOC_ROOT="/opt/share/www"
fi
mkdir -p "$DOC_ROOT" 2>/dev/null
if [ ! -f "$DOC_ROOT/index.html" ] || ! grep -q "/entware-manager/" "$DOC_ROOT/index.html" 2>/dev/null; then
	cat > "$DOC_ROOT/index.html" <<'INDEXEOF'
<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta http-equiv="refresh" content="0; url=/entware-manager/">
<title>Entware Manager</title>
</head>
<body>
<p>Перенаправление... <a href="/entware-manager/">Открыть Entware Manager</a></p>
</body>
</html>
INDEXEOF
	ok "корень $DOC_ROOT/index.html → редирект на панель"
fi

# CGI-диспетчер исполняется через локальный блок в 90-entware-manager.conf
# ($HTTP["url"] =~ "^/entware-cgi/" { cgi.assign = ( "" => "" ) }).
# Глобальный /opt/etc/lighttpd/conf.d/30-cgi.conf НЕ перезаписываем и НЕ удаляем —
# это общий файл lighttpd, он может принадлежать web4static/nfqws2 (perl/ruby/py/php).
if grep -q 'cgi\.assign = ( "" => "" )' "$CONF_FILE" 2>/dev/null; then
	ok "локальный cgi.assign для /entware-cgi/ (30-cgi.conf не трогаем)"
else
	LIGHTTPD_ERR="$LIGHTTPD_ERR 90-entware-manager.conf"
	fail "не найден локальный cgi.assign в $CONF_FILE"
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

else
	# ---- Режим "go": свой entware-server, общий lighttpd не трогаем ----

	# --- Миграция lighttpd → go (Variant 1) ---
	# Панель переезжает на entware-server:8087. Общий lighttpd получает
	# стабильный порт-хранитель (8086), если необходимо: освобождаем 8087,
	# не трогая чужие конфиги (koffe/web4static/nfqws2) и чужой server.port.

	# S4: если 8087 занят ЧУЖИМ процессом (не нашим lighttpd и не entware-server)
	# — ничего не трогаем (порт/lighttpd), только предупреждаем.
	PORT_SKIP=0
	if ! migrate_port_free 8087; then
		if entware_server_running; then
			ok "entware-server уже работает на 8087 — миграция порта не нужна"
			PORT_SKIP=1
		elif any_lighttpd_running; then
			ok "8087 держит lighttpd — будет освобождён порт-хранителем"
		else
			warn "порт 8087 занят чужим процессом — не трогаю lighttpd/порт (задай .port в server_config.json или освободи порт)"
			PORT_SKIP=1
		fi
	fi

	if [ "$PORT_SKIP" = "0" ]; then
		# Бэкап ДО перезаписи — для rollback.
		[ -f "$EWM_PORT_KEEPER" ] && backup_file "$EWM_PORT_KEEPER" "90-entware-manager.conf"

		PK=$(migrate_choose_portkeeper)
		if [ -z "$PK" ]; then
			# Эффективный порт свободен и чужих сервисов нет — можно удалить.
			if [ -f "$EWM_PORT_KEEPER" ]; then
				rm -f "$EWM_PORT_KEEPER" 2>/dev/null
				ok "90-entware-manager.conf удалён (порт свободен, чужих conf.d нет)"
			else
				ok "наших lighttpd-конфигов не найдено"
			fi
		else
			if migrate_is_portkeeper "$EWM_PORT_KEEPER"; then
				ok "90-entware-manager.conf уже порт-хранитель (:$PK)"
			else
				migrate_write_portkeeper "$PK"
				ok "90-entware-manager.conf → порт-хранитель :$PK (общий lighttpd переезжает)"
			fi
			# Освобождаем 8087: перезагрузка lighttpd (SIGHUP → restart fallback).
			migrate_reload_lighttpd 8087 || warn "не удалось освободить 8087 — проверь lighttpd вручную"

			# Сервисы общего lighttpd (koffe и др.) в links.json: :8087 → :$PK
			# (панель EM сама остаётся на 8087 — в links.json её нет как абсолютной ссылки).
			if [ "$PK" != "8087" ] && [ -f "$TARGET_DIR/links.json" ] && grep -q ':8087/' "$TARGET_DIR/links.json" 2>/dev/null; then
				backup_file "$TARGET_DIR/links.json" "links.json"
				sed -i "s#:8087/#:$PK/#g" "$TARGET_DIR/links.json" 2>/dev/null
				ok "links.json: сервисы общего lighttpd :8087 → :$PK (миграция)"
			fi
		fi
	fi

	# 30-cgi.conf — общий файл lighttpd (может принадлежать web4static/nfqws2).
	# Удаляем только наш устаревший артефакт (ровно наш шаблон); чужой не трогаем.
	if [ -f /opt/etc/lighttpd/conf.d/30-cgi.conf ] && is_our_cgi_conf /opt/etc/lighttpd/conf.d/30-cgi.conf; then
		rm -f /opt/etc/lighttpd/conf.d/30-cgi.conf 2>/dev/null
		ok "удалён наш 30-cgi.conf (устаревший артефакт)"
	elif [ -f /opt/etc/lighttpd/conf.d/30-cgi.conf ]; then
		ok "30-cgi.conf — чужой (web4static/nfqws2), не трогаем"
	fi

	# Конфиг сервера: порт (создаём только если нет — не перетираем настройку пользователя)
	SERVER_CFG="$TARGET_DIR/server_config.json"
	if [ ! -f "$SERVER_CFG" ]; then
		echo '{"port":8087}' > "$SERVER_CFG"
		ok "server_config.json создан (порт 8087)"
	else
		SERVER_PORT=$(jq -r '.port // 8087' "$SERVER_CFG" 2>/dev/null || echo 8087)
		ok "server_config.json уже есть (порт $SERVER_PORT)"
	fi

	# Проверяем, что порт не занят чужим сервисом
	SERVER_PORT=$(jq -r '.port // 8087' "$SERVER_CFG" 2>/dev/null || echo 8087)
	if [ -n "$SERVER_PORT" ] && command -v ss >/dev/null 2>&1; then
		if ss -tln 2>/dev/null | grep -q ":$SERVER_PORT "; then
			warn "порт $SERVER_PORT уже занят — поменяй .port в $SERVER_CFG"
		fi
	fi

	ok "общий lighttpd не тронут — конфликтов с nfqws/zapret нет"
fi

# Самовосстановление после прерванной установки (kill -9 посреди swap).
# TARGET_DIR модифицируется только атомарными rename, поэтому, когда он
# существует, он всегда целый. Если же его нет (убили между двумя mv),
# возвращаем рабочую версию из .old либо завершаем swap по маркеру
# .stage_complete — повторный запуск гарантированно приводит панель в
# рабочее состояние без ручной чистки.
self_heal_install() {
	if [ -d "$TARGET_DIR" ]; then
		[ -d "$STAGE_DIR" ] && rm -rf "$STAGE_DIR" 2>/dev/null
		return 0
	fi
	if [ -d "$STAGE_DIR" ] && [ -f "$STAGE_DIR/.stage_complete" ]; then
		mv "$STAGE_DIR" "$TARGET_DIR" 2>/dev/null && return 0
	fi
	if [ -d "$OLD_DIR" ]; then
		mv "$OLD_DIR" "$TARGET_DIR" 2>/dev/null && return 0
	fi
	[ -d "$STAGE_DIR" ] && rm -rf "$STAGE_DIR" 2>/dev/null
	return 0
}

# Минимальная проверка целостности собранного staging ДО подмены.
verify_stage() {
	[ -s "$STAGE_DIR/version.json" ] || return 1
	[ -x "$STAGE_DIR/cgi-bin/go.cgi" ] || return 1
	found=0
	for b in "$STAGE_DIR/cgi-bin/go"/*/entware-server; do
		[ -x "$b" ] && found=1 && break
	done
	[ "$found" = "1" ] || return 1
	[ -L "$STAGE_DIR/cgi-bin/session.cgi" ] || return 1
	return 0
}

# ========== 5. КОПИРОВАНИЕ ФАЙЛОВ ==========
step "Копирование файлов"

if [ "$SELF_DIR" = "$TARGET_DIR" ]; then
	ok "Файлы уже на месте (установка через ipk)"
else
	# Копирование в staging-каталог + атомарный swap:
	#  1) новая версия собирается в $TARGET_DIR.new (никаких изменений текущей),
	#  2) после успешной проверки каталоги меняются местами через mv (атомарно
	#     в пределах одной ФС — /opt это одна точка монтирования),
	#  3) старая версия остаётся в $TARGET_DIR.old до конца установки — откат
	#     при любой ошибке: rm -rf $TARGET_DIR && mv $TARGET_DIR.old $TARGET_DIR.
	# Пользовательские конфиги (*_config.json, links.json, .arch, backup/) из
	# старой установки переносятся в новую — их нет в deploy/.
	STAGE_DIR="$TARGET_DIR.new"
	OLD_DIR="$TARGET_DIR.old"

	# Самовосстановление после прерванной установки (kill -9 посреди swap).
	self_heal_install

	mkdir -p "$TARGET_DIR" || {
		fail "Не удалось создать $TARGET_DIR"
		exit 1
	}

	rm -rf "$STAGE_DIR" 2>/dev/null
	mkdir -p "$STAGE_DIR" || {
		fail "Не удалось создать staging ($STAGE_DIR)"
		exit 1
	}
	if ! cp -a "$SELF_DIR"/* "$STAGE_DIR/" 2>/dev/null; then
		fail "Не удалось скопировать файлы в staging ($STAGE_DIR)"
		rm -rf "$STAGE_DIR" 2>/dev/null
		exit 1
	fi

	# Целостность источника: version.json обязан быть непустым
	if [ ! -s "$STAGE_DIR/version.json" ]; then
		fail "Исходные файлы повреждены — version.json отсутствует или пуст"
		rm -rf "$STAGE_DIR" 2>/dev/null
		exit 1
	fi

	# Переносим пользовательские конфиги из старой установки в staging
	for cfg in "$TARGET_DIR"/*_config.json; do
		[ -f "$cfg" ] && cp -a "$cfg" "$STAGE_DIR/" 2>/dev/null
	done
	[ -f "$TARGET_DIR/links.json" ] && cp -a "$TARGET_DIR/links.json" "$STAGE_DIR/" 2>/dev/null
	[ -f "$TARGET_DIR/.arch" ] && cp -a "$TARGET_DIR/.arch" "$STAGE_DIR/" 2>/dev/null
	if [ -d "$TARGET_DIR/backup" ]; then
		rm -rf "$STAGE_DIR/backup" 2>/dev/null
		cp -a "$TARGET_DIR/backup" "$STAGE_DIR/" 2>/dev/null
	fi

	# Пользовательские данные каталогов целиком: пользовательские манифесты
	# моста, секреты *.auth.json (0600, права сохранит cp -a), настройки
	# уведомлений/управления (_prefs.json) и конфиги логгера.
	# Штатные манифесты поставки обновятся ниже блоком «ставим только
	# отсутствующие» (кворум v1.16.4 P1: без этого обновление с 1.15.x+
	# стирало сервисы, пароли приложений и галочки).
	for d in bridge logger; do
		if [ -d "$TARGET_DIR/$d" ]; then
			mkdir -p "$STAGE_DIR/$d"
			cp -a "$TARGET_DIR/$d/." "$STAGE_DIR/$d/" 2>/dev/null
		fi
	done

	# Проверка целостности staging ДО подмены (бинарники/симлинки/go.cgi).
	if ! verify_stage; then
		fail "Staging повреждён — оставляю текущую рабочую версию"
		rm -rf "$STAGE_DIR" 2>/dev/null
		exit 1
	fi
	touch "$STAGE_DIR/.stage_complete" 2>/dev/null

	# Атомарный swap: старая → .old, новая → на место
	rm -rf "$OLD_DIR" 2>/dev/null
	mv "$TARGET_DIR" "$OLD_DIR" 2>/dev/null || {
		fail "Не удалось отложить старую версию ($OLD_DIR)"
		rm -rf "$STAGE_DIR" 2>/dev/null
		exit 1
	}
	if ! mv "$STAGE_DIR" "$TARGET_DIR" 2>/dev/null; then
		fail "Не удалось подменить каталог — восстанавливаю старую версию"
		mv "$OLD_DIR" "$TARGET_DIR" 2>/dev/null
		rm -rf "$STAGE_DIR" 2>/dev/null
		exit 1
	fi
fi
if [ -f "$TARGET_DIR/version.json" ]; then
	VERSION=$(jq -r .version "$TARGET_DIR/version.json" 2>/dev/null || echo '?')
	ok "Файлы скопированы в $TARGET_DIR (версия $VERSION, $(du -sh "$TARGET_DIR" | cut -f1))"
	if [ -d "$OLD_DIR" ]; then
		ok "Предыдущая версия сохранена в $OLD_DIR (удалится при успешной установке)"
	fi
else
	fail "Копирование файлов не удалось — $TARGET_DIR пуст"
	if [ -d "$OLD_DIR" ]; then
		echo "    Восстановление: rm -rf $TARGET_DIR && mv $OLD_DIR $TARGET_DIR"
	fi
	exit 1
fi

rm -f "$TARGET_DIR/README.md" "$TARGET_DIR/LICENSE" "$TARGET_DIR/DEVLOG.md" "$TARGET_DIR/DEVICE.md" "$TARGET_DIR/BUILD.md" "$TARGET_DIR/RULES.md" "$TARGET_DIR/TECH_SPEC.md" "$TARGET_DIR/forum_post.md" "$TARGET_DIR/Makefile" "$TARGET_DIR/build-ipk.sh" "$TARGET_DIR/opencode.json" "$TARGET_DIR/Install/Install.txt" "$TARGET_DIR/doc/NETWORK_PROMPT.md" "$TARGET_DIR/doc/IPK_BUILD.md" "$TARGET_DIR/conffiles" "$TARGET_DIR/control" "$TARGET_DIR/postinst" "$TARGET_DIR/prerm" 2>/dev/null || true

# Миграция links.json: прямые порты ttyd (8089/9089) теперь доступны через панель
# (/htop/, /terminal/ — тот же origin, работает и в LAN, и через KeenDNS/Remote).
LINKS_FILE="$TARGET_DIR/links.json"
if [ -f "$LINKS_FILE" ] && grep -qE ':(8089|9089)' "$LINKS_FILE" 2>/dev/null; then
	backup_file "$LINKS_FILE" "links.json"
	# Было "http://<ip>:8089" или "http://<ip>:9089" → относительный путь панели.
	sed -i 's#http://[^"]*:8089#/htop/#g; s#http://[^"]*:9089#/terminal/#g' "$LINKS_FILE" 2>/dev/null
	ok "links.json: порты ttyd → /htop/, /terminal/ (миграция)"
fi

# Устаревшие процессы ttyd (запущенные до v1.09.2) не знают про --base-path и
# отвечают 404 на /htop/ и /terminal/. Останавливаем их, чтобы после обновления
# панель показала "служба не запущена" (запуск — через Настройки → Терминал).
# Корректные процессы с --base-path не трогаем — активные сессии сохраняются.
for TTYD_CMDLINE in /proc/[0-9]*/cmdline; do
	[ -r "$TTYD_CMDLINE" ] || continue
	TTYD_PID=${TTYD_CMDLINE#/proc/}
	TTYD_PID=${TTYD_PID%/cmdline}
	TTYD_CMD=$(tr '\0' ' ' < "$TTYD_CMDLINE" 2>/dev/null)
	case "$TTYD_CMD" in
		*ttyd*-p\ 8089*|*ttyd*-p\ 9089*)
			case "$TTYD_CMD" in
				*--base-path*) ;;
				*) kill "$TTYD_PID" 2>/dev/null && warn "ttyd: остановлен устаревший процесс (PID $TTYD_PID) — запустите службу в Настройки → Терминал" || true ;;
			esac
			;;
	esac
done

# Конфиг RDP-модуля: порт прокси и пути (создаём в обоих режимах — lighttpd и go)
# build-deploy.sh исключает *_config.json из deploy, поэтому файл создаём здесь.
# Манифесты моста сервисов: ставим только отсутствующие —
# пользовательские правки и .auth.json (0600) не трогаем.
BRIDGE_SRC="$SELF_DIR/bridge"
BRIDGE_DST="$TARGET_DIR/bridge"
mkdir -p "$BRIDGE_DST"
if [ -d "$BRIDGE_SRC" ]; then
    for f in "$BRIDGE_SRC"/*.json; do
        [ -f "$f" ] || continue
        bn=$(basename "$f")
        if [ ! -f "$BRIDGE_DST/$bn" ]; then
            cp -a "$f" "$BRIDGE_DST/$bn" && chmod 644 "$BRIDGE_DST/$bn"
        fi
    done
    ok "манифесты моста: установлены отсутствующие (существующие не перезаписаны)"
fi

RDP_CFG="$TARGET_DIR/rdp_config.json"
if [ ! -f "$RDP_CFG" ]; then
	echo '{"proxy_port":9099,"proxy_host":"","bin_path":"/opt/web_entware/cgi-bin/go/grdp-proxy","static_dir":"/opt/web_entware/static/rdp/","enabled":false}' > "$RDP_CFG"
	ok "rdp_config.json создан (порт 9099)"
else
	ok "rdp_config.json уже есть"
fi

# Подставить фактический порт RDP-прокси в lighttpd-conf (плейсхолдер __RDP_PORT__).
if [ -f "$CONF_FILE" ]; then
	RP=$(jq -r '.proxy_port // 9099' "$RDP_CFG" 2>/dev/null || echo 9099)
	case "$RP" in ''|*[!0-9]*) RP=9099 ;; esac
	sed -i "s/__RDP_PORT__/$RP/g" "$CONF_FILE" 2>/dev/null
	ok "lighttpd-conf: RDP-порт $RP подставлен в proxy.server"
fi

# Конфиг Telegram-шлюза (дефолт, без токена; 0600 ставит Go при сохранении).
# *_config.json исключается из deploy, поэтому создаём здесь.
TG_CFG="$TARGET_DIR/telegram_config.json"
if [ ! -f "$TG_CFG" ]; then
	echo '{"enabled":false,"bot_token":"","chat_id":"","level":"ERROR","sources":["system","monitor"],"bot_enabled":false,"autostart":false,"proxy_url":"http://127.0.0.1:10871"}' > "$TG_CFG"
	chmod 600 "$TG_CFG"
	ok "telegram_config.json создан"
else
	ok "telegram_config.json уже есть"
fi

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

# RDP-модуль: grdp-proxy + WASM-клиент + init-скрипт (если присутствуют в поставке).
# Блок после flatten-шага: бинарники уже разложены из cgi-bin/go/<arch>/ в cgi-bin/go/,
# поэтому проверки идут по финальным путям TARGET_DIR (не по SELF_DIR до распаковки).
RDP_PKG_DIR="$TARGET_DIR"
if [ -f "$RDP_PKG_DIR/cgi-bin/go/grdp-proxy" ] || [ -d "$RDP_PKG_DIR/static/rdp" ] || [ -f "$RDP_PKG_DIR/Install/S90grdp-proxy" ]; then
	mkdir -p "$RDP_PKG_DIR/cgi-bin/go" "$RDP_PKG_DIR/static/rdp"
	if [ -f "$RDP_PKG_DIR/cgi-bin/go/grdp-proxy" ]; then
		chmod 755 "$RDP_PKG_DIR/cgi-bin/go/grdp-proxy" 2>/dev/null
		ok "grdp-proxy установлен"
	else
		warn "grdp-proxy не найден в поставке — RDP-клиент будет недоступен"
	fi
	if [ -d "$RDP_PKG_DIR/static/rdp" ]; then
		chmod 644 "$RDP_PKG_DIR/static/rdp"/* 2>/dev/null
		ok "WASM-клиент RDP установлен"
	else
		warn "WASM-клиент RDP не найден в поставке"
	fi
	if [ -f "$RDP_PKG_DIR/Install/S90grdp-proxy" ]; then
		ln -sf "$RDP_PKG_DIR/Install/S90grdp-proxy" "/opt/etc/init.d/S90grdp-proxy" 2>/dev/null
		chmod 755 "$RDP_PKG_DIR/Install/S90grdp-proxy" 2>/dev/null
		ok "S90grdp-proxy установлен (симлинк)"
	else
		warn "S90grdp-proxy не найден в поставке — прокси придётся запускать вручную"
	fi
else
	ok "RDP-артефакты не в поставке — пропускаю установку модуля"
fi

# Форк index.html ttyd (перехват вставки Ctrl+V) — модуль терминала, не RDP,
# поэтому отдельный блок вне условия RDP-артефактов.
if [ -f "$TARGET_DIR/static/ttyd/index.html" ]; then
	chmod 644 "$TARGET_DIR/static/ttyd/index.html" 2>/dev/null
	ok "форк index.html ttyd установлен (вставка Ctrl+V)"
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
# Секретные конфиги — строго 600 (blanket chmod 644 выше перетирает права,
# а в них: хеш пароля и токен Telegram-бота; подтверждено живым тестом).
chmod 600 "$TARGET_DIR/auth_config.json" "$TARGET_DIR/telegram_config.json" 2>/dev/null
# Секреты моста (*.auth.json — пароли приложений) и его настройки (_prefs.json)
# тоже строго 600: контракт Go (manifest.go/prefs.go), кворум v1.16.4 P1 —
# blanket chmod 644 *.json выше их перетирал.
chmod 600 "$TARGET_DIR"/bridge/*.auth.json "$TARGET_DIR"/bridge/_prefs.json 2>/dev/null || true
find "$TARGET_DIR/cgi-bin" -type d -exec chmod 755 {} \; 2>/dev/null
ok "Права доступа установлены"

# ========== 8. ЗАПУСК ВЕБ-СЕРВЕРА ==========
step "Запуск веб-сервера"

if [ "$WEB_PATH" = "lighttpd" ]; then
	# --- Режим lighttpd: чистый роутер, стандартный путь ---
	# Если раньше стоял entware-server (обратный переход) — убираем его
	if [ -x /opt/etc/init.d/S80entware-server ]; then
		/opt/etc/init.d/S80entware-server stop 2>/dev/null
		rm -f /opt/etc/init.d/S80entware-server 2>/dev/null
		ok "предыдущий entware-server остановлен и удалён"
	fi
	if lighttpd_http_ok 8087; then
		ok "lighttpd уже отвечает на 127.0.0.1:8087"
	elif [ -f /opt/etc/init.d/S80lighttpd ]; then
		echo "  → перезапуск (конфиг мог измениться)..."
		# restart, а не start: если lighttpd уже работает на старом конфиге
		# (порт-хранитель 8086), start ничего не сделает и новый конфиг
		# (панель на 8087) не применится (MAJOR полного цикла go↔lighttpd).
		/opt/etc/init.d/S80lighttpd restart 2>&1 | sed 's/^/    /'
		sleep 2
	elif [ -x /opt/sbin/lighttpd ]; then
		warn "S80lighttpd не найден, попробую запустить напрямую"
		/opt/sbin/lighttpd -f "$LIGHTTPD_CONF" >/dev/null 2>&1 &
		sleep 1
	fi
else
	# --- Режим go: собственный entware-server ---
	EWM_SERVER_INIT="/opt/etc/init.d/S80entware-server"
	if [ -f "$SELF_DIR/Install/S80entware-server" ]; then
		ln -sf "$TARGET_DIR/Install/S80entware-server" "$EWM_SERVER_INIT"
		chmod 755 "$TARGET_DIR/Install/S80entware-server"
		ok "S80entware-server установлен (симлинк)"
	else
		warn "шаблон S80entware-server не найден в $SELF_DIR/Install/"
	fi
	if [ -f "$EWM_SERVER_INIT" ]; then
		echo "  → запуск entware-server..."
		$EWM_SERVER_INIT start 2>&1 | sed 's/^/    /'
		if [ "$PORT_SKIP" = "0" ] && ! lighttpd_http_ok 8087; then
			# entware-server не поднялся на 8087 → откат: вернуть прежний 90-conf,
			# перезагрузить lighttpd (панель остаётся жива через lighttpd-режим).
			warn "entware-server не отвечает на 8087 — откат порта-хранителя"
			if [ -f "$BACKUP_DIR/opt/etc/lighttpd/conf.d/90-entware-manager.conf" ]; then
				cp -a "$BACKUP_DIR/opt/etc/lighttpd/conf.d/90-entware-manager.conf" "$EWM_PORT_KEEPER" 2>/dev/null
				if [ -x /opt/etc/init.d/S80lighttpd ]; then
					/opt/etc/init.d/S80lighttpd restart >/dev/null 2>&1
				else
					p=$(lighttpd_pid | head -1)
					[ -n "$p" ] && kill -HUP "$p" 2>/dev/null
				fi
				ok "восстановлен прежний 90-entware-manager.conf"
			fi
		fi
	fi
fi

# --- NDM fs.d хуки для bridge-модулей с init (автозапуск на Keenetic) ---
# На Keenetic автозапуск идёт через /opt/etc/ndm/fs.d/, а не через rc.unslung.
# bridge/*.json с полем "init" → генерируем хук S{N}name.sh в fs.d.
BRIDGE_DIR="$TARGET_DIR/bridge"
NDM_FSD="/opt/etc/ndm/fs.d"
if [ -d "$BRIDGE_DIR" ] && [ -d "$NDM_FSD" ]; then
	for mf in "$BRIDGE_DIR"/*.json; do
		[ -f "$mf" ] || continue
		# извлекаем init и id (BusyBox grep -o)
		init_name=$(grep -o '"init"[[:space:]]*:[[:space:]]*"[^"]*"' "$mf" 2>/dev/null | head -1 | sed 's/.*: *"//;s/"//')
		mod_id=$(grep -o '"id"[[:space:]]*:[[:space:]]*"[^"]*"' "$mf" 2>/dev/null | head -1 | sed 's/.*: *"//;s/"//')
		[ -z "$init_name" ] && continue
		[ -z "$mod_id" ] && continue
		# ищем существующий init-скрипт в /opt/etc/init.d/
		init_script=""
		for candidate in /opt/etc/init.d/S*"${init_name}" ; do
			[ -x "$candidate" ] && init_script="$candidate" && break
		done
		[ -z "$init_script" ] && continue
		hook_file="$NDM_FSD/S50${mod_id}.sh"
		if [ ! -f "$hook_file" ]; then
			cat > "$hook_file" <<HOOKEOF
#!/bin/sh
case "\$1" in
    start|restore)
        $init_script start
        ;;
    stop)
        $init_script stop
        ;;
    restart)
        $init_script restart
        ;;
esac
HOOKEOF
			chmod 755 "$hook_file"
			ok "NDM fs.d хук: $mod_id → $(basename "$init_script")"
		else
			ok "NDM fs.d хук: $mod_id (уже существует)"
		fi
	done
fi

# --- Автозапуск демонов мониторинга (S85entware-watchdogs) ---
if [ -f "$SELF_DIR/Install/S85entware-watchdogs" ]; then
	ln -sf "$TARGET_DIR/Install/S85entware-watchdogs" "/opt/etc/init.d/S85entware-watchdogs" 2>/dev/null
	chmod 755 "$TARGET_DIR/Install/S85entware-watchdogs" 2>/dev/null
	ok "S85entware-watchdogs установлен (автозапуск демонов)"
else
	warn "шаблон S85entware-watchdogs не найден в $SELF_DIR/Install/"
fi

# --- Диагностические утилиты (tools/) доступными в PATH (/opt/bin) ---
if [ -d "$TARGET_DIR/tools" ]; then
	for t in "$TARGET_DIR/tools"/*.sh; do
		[ -f "$t" ] || continue
		name=$(basename "$t" .sh)
		ln -sf "$t" "/opt/bin/$name" 2>/dev/null && chmod 755 "$t"
	done
	ok "Утилиты диагностики установлены в /opt/bin ($(ls "$TARGET_DIR/tools" 2>/dev/null | wc -l) шт.)"
fi

if lighttpd_http_ok 8087; then
	ok "менеджер доступен по HTTP"
else
	fail "менеджер не доступен по HTTP (127.0.0.1:8087)"
	echo "    Для диагностики: tail -20 /opt/var/log/entware/server.log"
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
GO_BINS="entware-bridge entware-logger entware-monitor entware-net entware-pkg entware-rdp entware-server entware-services entware-smart entware-stats entware-telegram"
# Счётчик из самого списка: не отстанет при добавлении нового бинарника
# (кворум v1.16.4: «Найдено 11 из 10» блокировало блок успеха установки).
GO_TOTAL=$(echo $GO_BINS | wc -w)
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
if [ "$GO_OK" -eq "$GO_TOTAL" ]; then
	ok "  Все $GO_TOTAL бинарников"
else
	fail "  Найдено $GO_OK из $GO_TOTAL бинарников"
fi

# Проверка веб-файлов
echo "  ${BOLD}Веб-файлы:${NC}"
for f in index.html style.css entware.js theme.js icons.svg rdp.js version.json; do
	if [ -f "$TARGET_DIR/$f" ]; then
		ok "  $f"
	else
		CHECK_ERRS="$CHECK_ERRS
    ✗ $f — не найден"
		fail "  $f не найден"
	fi
done

# Проверка веб-сервера (по pid-файлу, не по имени процесса — чтобы не путаться с чужим lighttpd)
if [ "$WEB_PATH" = "lighttpd" ]; then
	echo "  ${BOLD}Lighttpd:${NC}"
	LIGHTTPD_PIDF=/opt/var/run/lighttpd.pid
	LIGHTTPD_PID=""
	if [ -f "$LIGHTTPD_PIDF" ]; then
		LIGHTTPD_PID=$(cat "$LIGHTTPD_PIDF" 2>/dev/null | tr -d ' ')
		[ -d "/proc/$LIGHTTPD_PID" ] || LIGHTTPD_PID=""
	fi
	if [ -n "$LIGHTTPD_PID" ]; then
		ok "  lighttpd (PID $LIGHTTPD_PID)"
	elif lighttpd_http_ok 8087; then
		ok "  lighttpd работает (порт 8087, pid-файл отсутствует)"
	else
		CHECK_ERRS="$CHECK_ERRS
    ✗ lighttpd не отвечает на 8087"
		fail "  lighttpd не отвечает на 8087"
	fi
else
	echo "  ${BOLD}Сервер:${NC}"
	EWM_PIDF=/opt/var/run/entware-server.pid
	EWM_PID=""
	if [ -f "$EWM_PIDF" ]; then
		EWM_PID=$(cat "$EWM_PIDF" 2>/dev/null | tr -d ' ')
		[ -d "/proc/$EWM_PID" ] || EWM_PID=""
	fi
	if [ -n "$EWM_PID" ]; then
		ok "  entware-server (PID $EWM_PID)"
	elif lighttpd_http_ok 8087; then
		ok "  entware-server работает (порт 8087, pid-файл отсутствует)"
	else
		CHECK_ERRS="$CHECK_ERRS
    ✗ entware-server не отвечает на 8087"
		fail "  entware-server не отвечает на 8087"
	fi
fi

# Проверка HTTP-ответа (фикс бага двойного «000»: -w уже даёт "000" при ошибке, без || echo)
if [ "$WEB_PATH" = "lighttpd" ]; then
	WEB_PORT=$(grep 'server\.port' "$LIGHTTPD_CONF" 2>/dev/null | grep -o '[0-9]*' | head -1)
	WEB_PORT=${WEB_PORT:-8087}
else
	WEB_PORT=$(jq -r '.port // 8087' "$TARGET_DIR/server_config.json" 2>/dev/null || echo 8087)
fi
echo "  ${BOLD}HTTP-ответ:${NC}"
if lighttpd_http_ok "$WEB_PORT"; then
	ok "  HTTP 200 (127.0.0.1:$WEB_PORT/entware-cgi/session.cgi)"
else
	HTTP_CODE="000"
	CHECK_ERRS="$CHECK_ERRS
    ✗ HTTP $HTTP_CODE (127.0.0.1:$WEB_PORT/entware-cgi/session.cgi)"
	fail "  HTTP $HTTP_CODE (127.0.0.1:$WEB_PORT/entware-cgi/session.cgi)"
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

if [ "$NET_FAILED" = "1" ]; then
	echo ""
	echo "${YELLOW}  ⚠ ВНИМАНИЕ: opkg update не выполнился (сеть/feed недоступны).${NC}"
	echo "${YELLOW}    Ошибки ниже могут быть следствием устаревших списков пакетов.${NC}"
	echo "${YELLOW}    Повтори установку при доступном bin.entware.net.${NC}"
fi

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
	# Удаляем отложенную предыдущую версию (успех — откат не нужен)
	if [ -n "${OLD_DIR:-}" ] && [ -d "$OLD_DIR" ]; then
		rm -rf "$OLD_DIR" 2>/dev/null
		ok "Предыдущая версия $OLD_DIR удалена"
	fi
fi
echo ""
echo "${GREEN}  ✓ Архитектура:${NC} $ROUTER_ARCH"
echo "${GREEN}  ✓ Файлы:${NC}    $TARGET_DIR"
echo "${GREEN}  ✓ Режим:${NC}    $WEB_PATH"
echo "${GREEN}  ✓ Статика:${NC}  http://$(hostname):${WEB_PORT:-8087}/entware-manager/"
echo "${GREEN}  ✓ CGI:${NC}      http://$(hostname):${WEB_PORT:-8087}/entware-cgi/"
echo ""
echo "  Версия: $(jq -r .version "$TARGET_DIR/version.json" 2>/dev/null || echo 'неизвестна')"
echo ""
IP=$(ip -o -4 addr show br0 2>/dev/null | awk '{print $4}' | cut -d/ -f1)
[ -z "$IP" ] && IP=$(ip route get 8.8.8.8 2>/dev/null | awk '{print $NF}' | head -1)
echo "  Открой в браузере: http://${IP:-<IP-роутера>}:${WEB_PORT:-8087}/entware-manager/"
echo ""
echo "  Терминал: Настройки → Терминал → Запустить"
echo ""
echo "  Лог установки: $LOG_FILE"
echo ""

if [ -n "$ERRORS" ] || [ -n "$CHECK_ERRS" ]; then
	echo "${YELLOW}  Были ошибки (см. выше). Исправь и запусти заново.${NC}"
	echo "${YELLOW}  Лог: $LOG_FILE${NC}"
fi
