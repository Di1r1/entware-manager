#!/bin/sh
# ==============================================
# Entware Manager - скрипт резервного копирования
# Версия: 2.6 (добавлена архивация в tar.gz)
# Дата: 2026-05-04
# ==============================================

set -e

BACKUP_ROOT="/opt/temp/backup"

# Получаем версию из version.json (до создания папки!)
if [ -f /opt/web_entware/version.json ] && command -v /opt/bin/jq >/dev/null 2>&1; then
    APP_VERSION=$(/opt/bin/jq -r '.version' /opt/web_entware/version.json 2>/dev/null)
    APP_DATE=$(/opt/bin/jq -r '.date' /opt/web_entware/version.json 2>/dev/null)
elif [ -f /opt/web_entware/version.json ]; then
    APP_VERSION=$(grep '"version"' /opt/web_entware/version.json 2>/dev/null | cut -d'"' -f4 || echo "unknown")
    APP_DATE=$(grep '"date"' /opt/web_entware/version.json 2>/dev/null | cut -d'"' -f4 || echo "unknown")
else
    APP_VERSION="unknown"
    APP_DATE="unknown"
fi

BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_BASENAME="EntwareManager_backup_v${APP_VERSION}_${BACKUP_DATE}"
BACKUP_DIR="$BACKUP_ROOT/$BACKUP_BASENAME"

mkdir -p "$BACKUP_ROOT"

echo "=== Создание резервной копии Entware Manager ==="
echo "📌 Версия: $APP_VERSION ($APP_DATE)"
echo "📁 Временный каталог: $BACKUP_DIR"

mkdir -p "$BACKUP_DIR"

# 1. Сохраняем список установленных пакетов
echo "💾 Сохранение списка пакетов..."
opkg list-installed > "$BACKUP_DIR/EntwareManager_packages_$APP_VERSION.txt"

# 2. Сохраняем важные конфиги
echo "💾 Сохранение конфигураций lighttpd и rc.local..."
[ -f /opt/etc/lighttpd/lighttpd.conf ] && cp /opt/etc/lighttpd/lighttpd.conf "$BACKUP_DIR/lighttpd.conf_$APP_VERSION"
[ -f /opt/etc/rc.local ] && cp /opt/etc/rc.local "$BACKUP_DIR/rc.local_$APP_VERSION"

# 3. Копируем весь каталог веб-интерфейса
echo "📂 Копирование /opt/web_entware ..."
cp -a /opt/web_entware "$BACKUP_DIR/web_entware"

# Проверяем наличие ключевых файлов
for file in modal.js entware.js style.css index.html icons.svg lib/common.sh lib/utils.js lib/smart.sh menu/menu.js menu/menu.json smart.js cgi-bin/smart.cgi cgi-bin/help.cgi cgi-bin/check_deps.cgi cgi-bin/go/entware-net cgi-bin/go/entware-pkg monitor_config.json; do
    if [ -f "$BACKUP_DIR/web_entware/$file" ]; then
        echo "✅ $file скопирован."
    else
        echo "⚠️ ВНИМАНИЕ: $file отсутствует в копии!"
    fi
done

# 4. Создаём файл с историей изменений
cat > "$BACKUP_DIR/CHANGELOG_$APP_VERSION.md" << 'CHANGELOG'
# История изменений Entware Manager

## Версия $APP_VERSION ($APP_DATE)

### Исправления безопасности
- **Исправлена критическая проблема**: логирование НЕ отключалось при `enabled=false`
  - Причина: `jq '// true'` трактует `false` как "falsy", возвращая `true`
  - Решение: используется явная проверка `if [ "$ENABLED_VALUE" = "true" ]`
- **Исправлена проблема с PATH в CGI контексте lighttpd**:
  - CGI получал PATH: `/sbin:/usr/sbin:/bin:/usr/bin` (без `/opt/bin`)
  - Утилиты `jq`, `cat` не находились
  - Решение: использован полный путь `/opt/bin/jq`, `/opt/bin/cat`

### Версии файлов
| Файл | Версия | Описание |
|------|--------|----------|
| lib/common.sh | 2.5 | Чистая версия |
| logger/lib/logging.sh | 1.7 | Исправлен jq |
| logger/config.cgi | 1.5 | Исправлены пути |
| cgi-bin/logger/system_log.cgi | 1.8 | Использованы /opt/bin |
| cgi-bin/crontab_update.cgi | 2.6 | Использует common.sh |
| monitor/*.cgi | разные | Исправлены пути |

## Версия 0.63 (2026-04-01)

### Новая функциональность
- Добавлен системный лог: `/opt/var/log/entware/system.log`
- Создан `logger/system_log.cgi` для просмотра системных событий
- Обновлён `logger/config.cgi` — пишет в системный лог
- Добавлена кнопка "Системные события" в UI

### Исправления
- **lib/common.sh v2.1**: исправлен `return` в `log_action()`
- **tmpfs.cgi v1.9**: убран `realpath`
- **links_save.cgi v0.03**: валидация JSON

## Версия 0.62 (2026-04-01)

### Исправления
- **lib/common.sh v2.0**: Переписан `url_decode()` — корректно декодирует `%2F` и другие URL-коды
- **delete_file.cgi v0.06**: Убран `realpath`, добавлены `exit 0`
- **api.cgi v0.04**: Добавлены `exit 0` после `json_out()`
- **wifi_temp.cgi v0.03**: Исправлено формирование JSON

### Оптимизация
- **История температур CPU/WiFi**: очистка запускается раз в день
- **services.cgi v2.3**: оптимизированы проверки
- **stats.cgi v0.25**: оптимизирован парсинг

### Безопасность
- **packages.cgi v0.21**: XSS-экранирование
- Все CGI возвращают корректные HTTP-коды

---

## Зависимости
- lighttpd (mod_cgi, mod_alias)
- ttyd, htop, jq, coreutils-base, coreutils-timeout, procps-ng
- smartmontools, smartmontools-drivedb
- sudo, bridge-utils, ip-full
CHANGELOG

# 5. Создаём файл с инструкцией по восстановлению
# (текст адаптирован для архива)
cat > "$BACKUP_DIR/EntwareManager_restore_$APP_VERSION.txt" << INSTR
=== Восстановление Entware Manager ===
Бэкап создан: $BACKUP_DATE
Версия: $APP_VERSION ($APP_DATE)
Архив: $BACKUP_ROOT/$BACKUP_BASENAME.tar.gz

=== ШАГ 1. Распакуйте архив ===
   cd /opt/temp/backup
   tar -xzf $BACKUP_BASENAME.tar.gz
   cd $BACKUP_BASENAME

=== ШАГ 2. Установка зависимостей ===
Перед восстановлением убедитесь, что установлены все необходимые пакеты:

   opkg update
   opkg install lighttpd ttyd htop jq coreutils-base coreutils-timeout procps-ng smartmontools smartmontools-drivedb sudo bridge-utils ip-full

Если каких-то пакетов нет в репозитории, установите их вручную.

=== ШАГ 3. Восстановление файлов ===
Скопируйте сохранённый каталог web_entware обратно:

   rm -rf /opt/web_entware
   cp -a web_entware /opt/

Все файлы интерфейса будут восстановлены.

=== ШАГ 4. Восстановление конфигураций (опционально) ===
Если вы хотите вернуть старые настройки lighttpd и автозапуск:

   cp lighttpd.conf_$APP_VERSION /opt/etc/lighttpd/lighttpd.conf
   cp rc.local_$APP_VERSION /opt/etc/rc.local
   chmod +x /opt/etc/rc.local

=== ШАГ 5. Восстановление списка пакетов (опционально) ===
Чтобы переустановить все пакеты:

   opkg install \$(cat EntwareManager_packages_$APP_VERSION.txt | awk '{print \$1}')

Внимание: некоторые пакеты могут отсутствовать в текущих репозиториях.

=== ШАГ 6. Права доступа ===
Убедитесь, что все CGI-скрипты исполняемые:

    chmod 755 /opt/web_entware/cgi-bin/*.cgi
    [ -d /opt/web_entware/cgi-bin/go ] && chmod 755 /opt/web_entware/cgi-bin/go/*
    [ -d /opt/web_entware/cgi-bin/monitor ] && chmod 755 /opt/web_entware/cgi-bin/monitor/*.cgi
    [ -d /opt/web_entware/cgi-bin/logger ] && chmod 755 /opt/web_entware/cgi-bin/logger/*.cgi
    [ -d /opt/web_entware/cgi-bin/network ] && chmod 755 /opt/web_entware/cgi-bin/network/*.cgi
   [ -d /opt/web_entware/cgi-bin/service_watchdog ] && chmod 755 /opt/web_entware/cgi-bin/service_watchdog/*.cgi
   chmod 755 /opt/web_entware/lib/*.sh
   chmod 755 /opt/web_entware/watchdog.sh
   chmod 755 /opt/web_entware/network_watchdog.sh
   chmod 755 /opt/web_entware/service_watchdog.sh
   chmod 755 /opt/web_entware/backup.sh

=== ШАГ 7. Перезапуск lighttpd ===
   /opt/etc/init.d/S80lighttpd restart

=== ШАГ 8. Проверка работы ===
Откройте в браузере:

   http://$(hostname -I | awk '{print $1}'):8087/entware-manager/

Проверьте все вкладки: статистика, пакеты, процессы, терминал, настройки, защита, логи.

=== НЮАНСЫ ===
- **Иконки (SVG спрайт)** – при восстановлении очистите кэш браузера (Ctrl+F5).
- **Пароль терминала** – не сохраняется в бэкапе.
- **Порты** – убедитесь, что порты 8087, 8089, 9089 свободны.
- **Автозапуск** – добавьте команды в /opt/etc/rc.local.
- **Права** – проверьте chmod 755 для CGI и 755 для watchdog.sh.
- **Тёмная тема** – настройки в localStorage браузера.
- **Логи** – хранятся в /tmp/entware/logs/.
- **История температур** – хранится в /tmp/temp_history/.

=== ПОДДЕРЖКА ===
Полная документация: см. CHANGELOG_$APP_VERSION.md и каталог web_entware/doc/
INSTR

# 6. Архивация (tar.gz)
ARCHIVE_NAME="$BACKUP_ROOT/$BACKUP_BASENAME.tar.gz"
if command -v /opt/bin/tar >/dev/null 2>&1 && command -v /opt/bin/gzip >/dev/null 2>&1; then
    echo "📦 Создание архива $ARCHIVE_NAME ..."
    cd "$BACKUP_ROOT"
    /opt/bin/tar -czf "$ARCHIVE_NAME" "$BACKUP_BASENAME"
    if [ $? -eq 0 ]; then
        echo "🗑️  Удаление временного каталога..."
        rm -rf "$BACKUP_DIR"
        BACKUP_FILE="$ARCHIVE_NAME"
    else
        echo "⚠️  Ошибка при создании архива, оставлен каталог: $BACKUP_DIR"
        BACKUP_FILE="$BACKUP_DIR"
    fi
else
    echo "⚠️  tar или gzip не найдены в /opt/bin, архив не создан."
    BACKUP_FILE="$BACKUP_DIR"
fi

# Вычисляем размер конечного файла/каталога
if [ -f "$BACKUP_FILE" ]; then
    SIZE=$(du -sh "$BACKUP_FILE" 2>/dev/null | cut -f1)
    echo "📦 Размер архива: $SIZE"
elif [ -d "$BACKUP_FILE" ]; then
    SIZE=$(du -sh "$BACKUP_FILE" 2>/dev/null | cut -f1)
    echo "📦 Размер каталога: $SIZE"
else
    echo "📦 Размер: неизвестен"
fi

echo ""
echo "✅ Резервная копия создана: $BACKUP_FILE"
if [ -f "$ARCHIVE_NAME" ]; then
    echo "📄 Инструкция (внутри архива): $BACKUP_BASENAME/EntwareManager_restore_$APP_VERSION.txt"
else
    echo "📄 Инструкция: $BACKUP_DIR/EntwareManager_restore_$APP_VERSION.txt"
fi
echo "📝 История изменений (внутри архива): CHANGELOG_$APP_VERSION.md"
echo ""
echo "Для восстановления следуйте шагам из инструкции."