# Установка и обновление

## 1. Системные требования

- Прошивка/среда: OpenWrt/Entware.
- Веб-сервер: `lighttpd` с модулями `mod_cgi`, `mod_alias` (опционально `mod_rewrite`).
- Пакеты:
  - `lighttpd` — веб-сервер
  - `ttyd` — веб-терминал и монитор процессов
  - `htop` — интерактивный монитор процессов
  - `jq` — работа с JSON
  - `coreutils-base` — базовые системные утилиты
  - `procps-ng` — утилиты процессов (pgrep, ps)
  - `bridge-utils` — отображение мостов (brctl) на вкладке статистики
  - `ip-full` — расширенный ip (маршруты, интерфейсы)
- Рекомендуемые порты:
  - `8087` — веб-интерфейс
  - `8089` — ttyd + htop
  - `9089` — ttyd + shell

Установка зависимостей:

```sh
opkg update
opkg install lighttpd ttyd htop jq coreutils-base procps-ng bridge-utils ip-full
```

После установки пакетов перезапустите lighttpd:

```sh
/opt/etc/init.d/S80lighttpd restart
```

## 2. Структура размещения

```
/opt/web_entware/
├── cgi-bin/               # CGI-скрипты
│   ├── monitor/           #   демон защиты процессов
│   ├── logger/            #   логирование
│   ├── network/           #   мониторинг сети
│   └── service_watchdog/  #   отслеживание служб
├── lib/                   # общие функции (common.sh)
├── logger/
│   ├── lib/               #   библиотека логирования (logging.sh)
│   └── scripts/           #   скрипты (rotate.sh)
├── menu/                  # конфигурация меню
├── doc/                   # документация
├── Install/               # установщик
├── version.json           # версия проекта
├── entware.js             # основной JS-файл
├── style.css              # стили
├── index.html             # главная страница
├── watchdog.sh            # демон защиты процессов
├── network_watchdog.sh    # демон мониторинга сети
├── service_watchdog.sh    # демон отслеживания служб
└── backup.sh              # резервное копирование
```

## 3. Установка из временного каталога

Если файлы распакованы в `/opt/temp/web_entware`, можно выполнить установщик:

```sh
chmod +x /opt/temp/web_entware/Install/install.sh
/opt/temp/web_entware/Install/install.sh
```

Скрипт:
- копирует файлы в `/opt/web_entware`
- выставляет права для CGI и shell-скриптов
- перезапускает `lighttpd` (если процесс запущен)

## 4. Ручная установка

1. Создайте каталог:

```sh
mkdir -p /opt/web_entware
```

2. Скопируйте все файлы проекта.

3. Выставьте права:

```sh
chmod 755 /opt/web_entware/cgi-bin/*.cgi
[ -d /opt/web_entware/cgi-bin/monitor ] && chmod 755 /opt/web_entware/cgi-bin/monitor/*.cgi
[ -d /opt/web_entware/cgi-bin/logger ] && chmod 755 /opt/web_entware/cgi-bin/logger/*.cgi
[ -d /opt/web_entware/cgi-bin/network ] && chmod 755 /opt/web_entware/cgi-bin/network/*.cgi
[ -d /opt/web_entware/cgi-bin/service_watchdog ] && chmod 755 /opt/web_entware/cgi-bin/service_watchdog/*.cgi
chmod 755 /opt/web_entware/watchdog.sh
chmod 755 /opt/web_entware/network_watchdog.sh
chmod 755 /opt/web_entware/service_watchdog.sh
chmod 755 /opt/web_entware/backup.sh
[ -f /opt/web_entware/logger/lib/logging.sh ] && chmod 755 /opt/web_entware/logger/lib/*.sh
[ -d /opt/web_entware/logger/scripts ] && chmod 755 /opt/web_entware/logger/scripts/*.sh
find /opt/web_entware -type f -name "*.sh" -exec chmod 755 {} \;
find /opt/web_entware -type f \( -name "*.js" -o -name "*.css" -o -name "*.html" -o -name "*.json" -o -name "*.svg" \) -exec chmod 644 {} \;
find /opt/web_entware/cgi-bin -type d -exec chmod 755 {} \;
```

Важно: каталоги, по которым CGI должны проходить, должны иметь `x`-бит (обычно `755`).

## 5. Настройка lighttpd

В `lighttpd.conf` добавьте алиасы:

```conf
alias.url = (
    "/entware-manager/" => "/opt/web_entware/",
    "/entware-cgi/" => "/opt/web_entware/cgi-bin/"
)
```

Убедитесь, что подключены модули `mod_alias` и `mod_cgi`:

```conf
server.modules += ( "mod_alias", "mod_cgi" )
```

Перезапуск:

```sh
/opt/etc/init.d/S80lighttpd restart
```

Проверьте, что lighttpd запущен:

```sh
pgrep -f lighttpd
```

## 6. Первичная проверка

1. Откройте `http://<router-ip>:8087/entware-manager/`.
2. Проверьте вкладки: статистика, пакеты, службы, cron, настройки, защита, логи.
3. Проверьте `version.json` и отображаемую версию в UI.
4. Если вкладка «Статистика» не показывает сетевые данные — проверьте установку `bridge-utils` и `ip-full`.

## 7. Обновление

Рекомендуемый порядок:

1. Выполнить бэкап:

```sh
/opt/web_entware/backup.sh
```

2. Скопировать новые файлы поверх старых (повторно выполнить `/opt/temp/web_entware/Install/install.sh`).
3. Проверить права на CGI.
4. Перезапустить `lighttpd`.
5. Принудительно обновить страницу в браузере (`Ctrl+F5`) из-за кеша JS/CSS/SVG.

Конфигурации (links.json, monitor_config.json, network_config.json, service_config.json)
при обновлении сохраняются (они не входят в дистрибутив).

## 8. Удаление

```sh
rm -rf /opt/web_entware
```

Дополнительно (по необходимости):
- удалить cron-задания, связанные с проектом
- остановить ttyd-процессы (если запускались вручную)
- остановить watchdog.sh, network_watchdog.sh, service_watchdog.sh
- удалить файлы логов и временной истории: `/tmp/entware/logs/`, `/tmp/temp_history/`, `/opt/temp/logs/`
