#!/bin/sh
# ==============================================
# Полное удаление Entware Manager с роутера
# ==============================================

# shellcheck disable=SC2034,SC3043,SC1090
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

# Все пути — переменные в начале (RULES §4) и переопределяются из
# окружения (EWM_*): так uninstall.sh можно прогнать в песочнице из
# test/uninstall_backup.sh, не трогая реальный роутер.
TARGET_DIR="${EWM_TARGET_DIR:-/opt/web_entware}"
BACKUP_DIR="$TARGET_DIR/backup"
# Каталог для бэкапов, переживающих удаление панели (ВНЕ $TARGET_DIR).
KEEP_DIR="${EWM_KEEP_DIR:-/opt/var/lib/entware-manager/backup}"
LIGHTTPD_CONF="${EWM_LIGHTTPD_CONF:-/opt/etc/lighttpd/lighttpd.conf}"
CGI_CONF="${EWM_CGI_CONF:-/opt/etc/lighttpd/conf.d/30-cgi.conf}"
LOG_DIR="${EWM_LOG_DIR:-/tmp/entware}"
SUDOERS_FILE="${EWM_SUDOERS_FILE:-/opt/etc/sudoers.d/entware-smartctl}"
CONFD_OUR="${EWM_CONFD_OUR:-/opt/etc/lighttpd/conf.d/90-entware-manager.conf}"
S85_INIT="${EWM_S85_INIT:-/opt/etc/init.d/S85entware-watchdogs}"
LIGHTTPD_INIT="${EWM_LIGHTTPD_INIT:-/opt/etc/init.d/S80lighttpd}"

# Миграционные функции порт-хранителя. Source ДО удаления $TARGET_DIR.
MIGRATE_LIB=""
[ -f "$TARGET_DIR/lib/migrate.sh" ] && MIGRATE_LIB="$TARGET_DIR/lib/migrate.sh"
[ -z "$MIGRATE_LIB" ] && [ -f "/opt/web_entware/lib/migrate.sh" ] && MIGRATE_LIB="/opt/web_entware/lib/migrate.sh"
[ -n "$MIGRATE_LIB" ] && . "$MIGRATE_LIB"
# Fallback для установки ДО v1.11 (migrate.sh отсутствует): не удаляем порт-хранитель,
# т.к. иначе общий lighttpd может упасть на занятый порт 80.
if [ -z "$MIGRATE_LIB" ]; then
	migrate_is_portkeeper() { return 1; }
	migrate_choose_portkeeper() { echo ""; }
	migrate_write_portkeeper() { return 0; }
fi

# Наш ли это 30-cgi.conf (ровно наш шаблон: cgi.assign .cgi=>/bin/sh + execute-x-only)?
# Чужой файл (web4static/nfqws2: perl/ruby/py/php) не удаляем.
is_our_cgi_conf() {
	[ ! -f "$1" ] && return 1
	if grep -Eq 'perl|ruby|python|php|\.pl|\.rb|\.erb|\.py|\.php' "$1" 2>/dev/null; then
		return 1
	fi
	local lines
	lines=$(wc -l < "$1" 2>/dev/null || echo 0)
	[ "$lines" -le 3 ] || return 1
	grep -q 'cgi\.assign.*\.cgi.*/bin/sh' "$1" 2>/dev/null || return 1
	return 0
}

echo "${BOLD}========================================"
echo " Удаление Entware Manager"
echo "========================================${NC}"
echo ""

OK=0
WARN=0

ok()   { echo "${GREEN}  ✓ $1${NC}"; OK=$((OK+1)); }
warn() { echo "${YELLOW}  ⚠ $1${NC}"; WARN=$((WARN+1)); }

# ========== 1. ПРОВЕРКА ==========
echo "${BOLD}[1/7] Проверка установки${NC}"
echo "────────────────────────────────────────"

if [ -d "$TARGET_DIR" ]; then
	SIZE=$(du -sh "$TARGET_DIR" 2>/dev/null | cut -f1)
	FILES=$(find "$TARGET_DIR" -type f | wc -l)
	ok "Найдена установка: $TARGET_DIR ($SIZE, $FILES файлов)"
else
	warn "Установка не найдена ($TARGET_DIR отсутствует)"
fi

# ========== 2. ОСТАНОВКА СЕРВИСОВ ==========
echo ""
echo "${BOLD}[2/8] Остановка сервисов${NC}"
echo "────────────────────────────────────────"

# Останавливаем наш entware-server (если установлен)
EWM_SERVER_INIT="${EWM_SERVER_INIT:-/opt/etc/init.d/S80entware-server}"
EWM_SERVER_RAN=0
if [ -x "$EWM_SERVER_INIT" ]; then
	if $EWM_SERVER_INIT stop 2>/dev/null; then
		EWM_SERVER_RAN=1
		ok "entware-server остановлен (S80entware-server)"
	else
		warn "Не удалось остановить entware-server"
	fi
else
	warn "entware-server не установлен (S80entware-server нет)"
fi

LIGHTTPD_PIDF="${EWM_LIGHTTPD_PIDF:-/opt/var/run/lighttpd.pid}"
LIGHTTPD_PID=""
if [ -f "$LIGHTTPD_PIDF" ]; then
	LIGHTTPD_PID=$(cat "$LIGHTTPD_PIDF" 2>/dev/null | tr -d ' ')
	[ -d "/proc/$LIGHTTPD_PID" ] || LIGHTTPD_PID=""
fi

# Если есть наш init-скрипт — останавливаем ТОЛЬКО свой экземпляр (по pid-файлу),
# не трогая чужой lighttpd (например zapret). Иначе — стандартный S80lighttpd.
EWM_INIT="${EWM_INIT:-/opt/etc/init.d/S80entware-lighttpd}"
if [ "$EWM_SERVER_RAN" = "1" ]; then
	# Режим entware-server: lighttpd не наш, чужой (nfqws/zapret) не трогаем
	warn "lighttpd не трогаю (режим entware-server)"
elif [ -n "$LIGHTTPD_PID" ] || [ -x "$EWM_INIT" ]; then
	if [ -x "$EWM_INIT" ]; then
		$EWM_INIT stop 2>/dev/null
		sleep 1
		ok "lighttpd остановлен (S80entware-lighttpd)"
	elif [ -n "$LIGHTTPD_PID" ]; then
		kill "$LIGHTTPD_PID" 2>/dev/null
		sleep 1
		rm -f "$LIGHTTPD_PIDF" 2>/dev/null
		ok "lighttpd остановлен (pid $LIGHTTPD_PID)"
	fi
elif "$LIGHTTPD_INIT" stop 2>/dev/null; then
	sleep 1
	ok "lighttpd остановлен"
else
	warn "lighttpd не запущен"
fi

# ========== 3. УДАЛЕНИЕ ФАЙЛОВ ==========
echo ""
echo "${BOLD}[3/7] Удаление файлов${NC}"
echo "────────────────────────────────────────"

# Откатные копии конфигов лежат ВНУТРИ $TARGET_DIR, поэтому до удаления
# выносим их в $KEEP_DIR — иначе они исчезают вместе с панелью, а итог
# обещал обратное. Правка install.sh здесь не нужна: он эти копии и так
# сохраняет при обновлении, если backup/ лежит рядом с панелью.
BACKUP_SAVED=""
BACKUP_LOST=""
if [ -d "$BACKUP_DIR" ]; then
	# Копируем во временный каталог и подменяем целиком: так в $KEEP_DIR
	# не остаётся огрызков от прошлой копии и не появляется частичный архив.
	KEEP_TMP="$KEEP_DIR.tmp.$$"
	if mkdir -p "$KEEP_TMP" 2>/dev/null && cp -a "$BACKUP_DIR/." "$KEEP_TMP/" 2>/dev/null; then
		if rm -rf "$KEEP_DIR" 2>/dev/null && mv "$KEEP_TMP" "$KEEP_DIR" 2>/dev/null; then
			BACKUP_SAVED="$KEEP_DIR"
			ok "Бэкап конфигов сохранён: $KEEP_DIR"
		else
			rm -rf "$KEEP_TMP" 2>/dev/null
			BACKUP_LOST=1
			warn "Не удалось сохранить бэкап конфигов в $KEEP_DIR — копии будут удалены вместе с панелью"
		fi
	else
		rm -rf "$KEEP_TMP" 2>/dev/null
		BACKUP_LOST=1
		warn "Не удалось сохранить бэкап конфигов в $KEEP_DIR — копии будут удалены вместе с панелью"
	fi
fi
# Конфигурация самой панели (пароль, ссылки, настройки модулей) удаляется
# вместе с панелью всегда — предупреждаем, чтобы владелец скачал бэкап.
if [ -f "$TARGET_DIR/auth_config.json" ]; then
	warn "Конфигурация панели (пароль, ссылки, модули) удаляется вместе с панелью. Если резервная копия не скачана — восстановить будет нечем."
fi

if [ -d "$TARGET_DIR" ]; then
	rm -rf "$TARGET_DIR"
	if [ ! -d "$TARGET_DIR" ]; then
		ok "$TARGET_DIR удалён"
	else
		warn "Не удалось полностью удалить $TARGET_DIR"
	fi
else
	warn "$TARGET_DIR уже удалён"
fi

# ========== 4. УДАЛЕНИЕ КОНФИГОВ LIGHTTPD ==========
echo ""
echo "${BOLD}[4/8] Удаление конфигов lighttpd${NC}"
echo "────────────────────────────────────────"

# 90-entware-manager.conf: порт-хранитель НЕ удаляем — иначе общий lighttpd
# упадёт на занятый порт 80 вместе с koffe/web4static. Полноценную lighttpd-
# панель заменяем порт-хранителем (S9b), чтобы чужие conf.d-сервисы жили.
if migrate_is_portkeeper "$CONFD_OUR"; then
	ok "порт-хранитель сохранён (общий lighttpd остаётся на стабильном порту)"
elif [ -f "$CONFD_OUR" ]; then
	PK=$(migrate_choose_portkeeper)
	if [ -n "$PK" ]; then
		migrate_write_portkeeper "$PK"
		ok "90-entware-manager.conf заменён порт-хранителем (:$PK)"
	else
		rm -f "$CONFD_OUR" 2>/dev/null
		ok "90-entware-manager.conf удалён (порт свободен)"
	fi
fi

# Наш init-скрипт entware-server
rm -f "$EWM_SERVER_INIT" 2>/dev/null
ok "S80entware-server удалён"

# Старый init-скрипт entware-lighttpd
rm -f "$EWM_INIT" 2>/dev/null
ok "S80entware-lighttpd удалён"

# Автозапуск демонов мониторинга
rm -f "$S85_INIT" 2>/dev/null
ok "S85entware-watchdogs удалён"

# 30-cgi.conf — удаляем ТОЛЬКО если это наш файл (точное совпадение с шаблоном).
# Чужой (web4static/nfqws2) не трогаем.
if [ -f "$CGI_CONF" ] && is_our_cgi_conf "$CGI_CONF"; then
	rm -f "$CGI_CONF" 2>/dev/null
	ok "30-cgi.conf (наш) удалён"
elif [ -f "$CGI_CONF" ]; then
	ok "30-cgi.conf — чужой (web4static/nfqws2), не трогаем"
fi

# Чистим старые строки из lighttpd.conf (на случай обновления с предыдущей версии)
[ -f "$LIGHTTPD_CONF" ] && sed -i '\|"/entware-manager/"|d' "$LIGHTTPD_CONF" || true
[ -f "$LIGHTTPD_CONF" ] && sed -i '\|"/entware-cgi/"|d' "$LIGHTTPD_CONF" || true
sed -i '/^[[:space:]]*server\.port[[:space:]]*=.*8087.*/d' "$LIGHTTPD_CONF" 2>/dev/null
sed -i '/^alias\.url = (\s*)$/d' "$LIGHTTPD_CONF" 2>/dev/null

# ========== 5. УДАЛЕНИЕ SUDOERS ==========
echo ""
echo "${BOLD}[5/8] Удаление sudoers${NC}"
echo "────────────────────────────────────────"

if [ -f "$SUDOERS_FILE" ]; then
	rm -f "$SUDOERS_FILE"
	ok "$SUDOERS_FILE удалён"
else
	warn "sudoers уже удалён"
fi

# ========== 6. УДАЛЕНИЕ ЛОГОВ ==========
echo ""
echo "${BOLD}[6/8] Удаление логов и временных файлов${NC}"
echo "────────────────────────────────────────"

if [ -d "$LOG_DIR" ]; then
	rm -rf "$LOG_DIR"
	ok "$LOG_DIR удалён"
else
	warn "Логи уже удалены"
fi

# ========== 7. ЗАПУСК LIGHTTPD ==========
echo ""
echo "${BOLD}[7/8] Запуск lighttpd${NC}"
echo "────────────────────────────────────────"

if [ "$EWM_SERVER_RAN" = "1" ]; then
	# Режим entware-server: светтитхтd чужой, ничего не перезапускаем
	warn "lighttpd не перезапускаю (режим entware-server, он не наш)"
elif [ -x "$EWM_INIT" ]; then
	if "$EWM_INIT" start 2>/dev/null; then
		sleep 1
		ok "lighttpd запущен (S80entware-lighttpd)"
	else
		warn "Не удалось запустить lighttpd (S80entware-lighttpd)"
	fi
elif [ -f "$LIGHTTPD_INIT" ]; then
	if "$LIGHTTPD_INIT" start 2>/dev/null; then
		sleep 1
		ok "lighttpd запущен"
	else
		warn "Не удалось запустить lighttpd"
	fi
else
	warn "Скрипт запуска lighttpd не найден"
fi

echo ""
echo "${BOLD}[8/8] Итог${NC}"
echo ""
echo "${GREEN}  ✓ Выполнено: $OK${NC}"
if [ $WARN -gt 0 ]; then
	echo "${YELLOW}  ⚠ Предупреждений: $WARN${NC}"
fi
echo ""
echo "  Пакеты Entware (lighttpd, jq, curl и др.) не удалены —"
echo "  они могут использоваться другими программами."
echo "  Если хочешь удалить вручную:"
echo "    opkg remove lighttpd lighttpd-mod-cgi jq curl sudo smartmontools ..."
echo ""
if [ -n "$BACKUP_SAVED" ]; then
	echo "  Бэкап конфигов сохранён: $BACKUP_SAVED"
	echo "  При переустановке положи его обратно в $BACKUP_DIR — install.sh"
	echo "  сохранит эти копии при обновлении."
else
	echo "  Бэкап конфигов: не сохранялся (удалён вместе с панелью)."
fi
echo ""

# Код возврата: панель удалена успешно, но бэкап конфигов пропадёт —
# сообщаем об этом кодом, чтобы вызывающий (человек или скрипт) заметил.
if [ -d "$BACKUP_DIR" ] || [ -n "$BACKUP_LOST" ]; then
	exit 1
fi
exit 0
