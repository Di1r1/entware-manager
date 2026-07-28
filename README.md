# Entware Manager

[![ShellCheck](https://github.com/Di1r1/entware-manager/actions/workflows/shellcheck.yml/badge.svg)](https://github.com/Di1r1/entware-manager/actions/workflows/shellcheck.yml)
[![Version](https://img.shields.io/badge/version-1.06.2-blue)](version.json)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

**Entware Manager** — веб-панель управления Entware на роутерах Keenetic и Netcraze с NDMS.
Всё в браузере, без SSH и консоли.

### Возможности

| Раздел | Что умеет |
|--------|-----------|
| **Пакеты** | Установка, удаление, обновление пакетов Entware. Поиск, список установленных, список доступных обновлений |
| **Мониторинг** | CPU/RAM/диск, температура CPU и WiFi, история температур за 7 дней с графиками. Встроенный watchdog (перезапуск упавших процессов) |
| **Сеть** | Интерфейсы, маршруты, ARP-таблица. Мониторинг трафика (real-time). Watchdog: пинг шлюза/8.8.8.8, сброс интерфейса при потере связи |
| **Сервисы** | Управление сервисами (start/stop/restart/enable/disable). Мониторинг процессов, автоперезапуск по PID |
| **SMART** | S.M.A.R.T. атрибуты дисков (HDD/SSD/NVMe), health-статус, самотесты, температура накопителей |
| **Логи** | Просмотр системных логов, поиск, ротация, очистка |
| **Файлы** | Просмотр файлов в `/tmp/`, backup/restore настроек |
| **Терминал** | Встроенный веб-терминал (ttyd) для прямого доступа к shell |
| **Безопасность** | Защита паролем (SHA-256), auth_config.json |

### Интерфейс

- Одна страница (SPA) — всё на index.html
- Тёмная тема, иконки SVG
- Адаптивный дизайн (десктоп + мобильные)
- Клик по температуре в сайдбаре → графики за 7 дней

## Быстрый старт

### Установка с GitHub Releases

На роутере по SSH:

```sh
# Определи архитектуру:
uname -m

# Если нет curl: opkg update && opkg install curl
```

Качай архив под свою архитектуру (стабильная ссылка — всегда последняя версия):

```sh
# arm (armv7l / armv5tel)
curl -LO https://github.com/Di1r1/entware-manager/releases/latest/download/entware-manager-arm.tar.gz

# mips
curl -LO https://github.com/Di1r1/entware-manager/releases/latest/download/entware-manager-mips.tar.gz

# mipsel
curl -LO https://github.com/Di1r1/entware-manager/releases/latest/download/entware-manager-mipsel.tar.gz
```

Распаковывай и ставь:

```sh
tar -xzf entware-manager-*.tar.gz
cd deploy && sh Install/install.sh
```

После установки открой: `http://<ip-роутера>:8087/entware-manager/`

Все архивы: https://github.com/Di1r1/entware-manager/releases

### Установка через ipk

```sh
# arm (armv7l / armv5tel)
curl -LO https://github.com/Di1r1/entware-manager/releases/latest/download/entware-manager_arm.ipk
opkg install entware-manager_arm.ipk

# mips
curl -LO https://github.com/Di1r1/entware-manager/releases/latest/download/entware-manager_mips.ipk
opkg install entware-manager_mips.ipk

# mipsel
curl -LO https://github.com/Di1r1/entware-manager/releases/latest/download/entware-manager_mipsel.ipk
opkg install entware-manager_mipsel.ipk
```

Зависимости (lighttpd, jq, curl, ttyd и др.) opkg установит автоматически.
При установке запускается install.sh с полным цветным выводом и логом.

### Обновление

```sh
# Через tar.gz (стабильная ссылка — всегда последняя версия)
cd /opt/tmp
curl -LO https://github.com/Di1r1/entware-manager/releases/latest/download/entware-manager-mipsel.tar.gz
tar -xzf entware-manager-mipsel.tar.gz
cd deploy && sh Install/install.sh

# Через ipk
curl -LO https://github.com/Di1r1/entware-manager/releases/latest/download/entware-manager_mipsel.ipk
opkg install --force-reinstall entware-manager_mipsel.ipk
```

Все релизы: https://github.com/Di1r1/entware-manager/releases

### Удаление

```sh
# Если ставил вручную
sh /opt/web_entware/Install/uninstall.sh

# Если ставил через ipk
opkg remove entware-manager
```

Удаляет файлы, конфиги lighttpd, sudoers и логи. Пакеты Entware не трогает.

### Лог установки

install.sh пишет лог в `/tmp/entware/install-logs/install-YYYYMMDD-HHMMSS.log`.
При повторной установке создаётся новый файл, старые не затираются.

Финальный шаг проверяет:
- все пакеты и бинарники
- симлинки `.cgi → go.cgi`
- 7 Go-бинарников
- веб-файлы (index.html, style.css, …)
- lighttpd (PID + HTTP 200)

### Бэкап конфигов

Перед изменением `install.sh` сохраняет копии:
- `/opt/etc/lighttpd/lighttpd.conf` → `/opt/web_entware/backup/etc/lighttpd/lighttpd.conf`
- `/opt/etc/lighttpd/conf.d/30-cgi.conf` → `/opt/web_entware/backup/etc/lighttpd/conf.d/30-cgi.conf`

При удалении `uninstall.sh` восстанавливает конфиги из бэкапа. Если бэкапа нет — просто удаляет строки Entware Manager из `lighttpd.conf`.

### Установка через git

```sh
# Если есть git-http:
opkg update && opkg install git-http
git clone https://github.com/Di1r1/entware-manager.git /opt/tmp/entware-manager
cd /opt/tmp/entware-manager/Install
chmod +x install.sh
./install.sh
```

## Архитектура

```
браузер  ──http──▶  lighttpd (порт 8087)
                        │
                    mod_cgi
                        │
                   go.cgi (диспетчер)
                        │
              ┌─────────┼─────────┐
              │         │         │
         entware-*   *.html    *.js
       (7 Go-бинарн.) статика   логика
              │
         система / Entware
```

**Технологии:** Go (7 бинарников, UPX-сжатые), POSIX `sh` (BusyBox ash), `lighttpd` + `mod_cgi`, `jq`, `ttyd`.

## Компоненты

| Бинарник (Go) | Назначение | Основные эндпоинты |
|---------------|------------|-------------------|
| `entware-pkg` | Пакеты Entware | `available`, `packages`, `install`, `remove`, `upgrade`, `update`, `upgradable`, `api` |
| `entware-stats` | Инфо, файлы, ссылки | `stats`, `version`, `help`, `links_load/save`, `tmpfs`, `view_file`, `delete_file`, `auth_config`, `crontab` |
| `entware-net` | Сеть | `network_interfaces`, `routes`, `arp`, `status`, `stats`, `events`, `config`, `action` |
| `entware-services` | Сервисы, watchdog | `check_syntax/deps`, `services`, `service_action`, `ttyd_control`, `debug`, `service_watchdog/*` |
| `entware-monitor` | Мониторинг | `temperature`, `wifi_temp`, `temp_history`, `wifi_temp_history`, `kill_pid`, `monitor_*` |
| `entware-smart` | SMART дисков | `smart` (info, attributes, health, selftest) |
| `entware-logger` | Логи | `logger_*` (config, view, system_logs, rotate, clear)` |

## Конфигурация

| Файл | Описание |
|------|----------|
| `/opt/etc/entware-manager.conf` | Пути (генерируется install.sh): `ENTWARE_MANAGER_ROOT`, `ENTWARE_MANAGER_CGI`, `ENTWARE_MANAGER_LOGS`, `ENTWARE_MANAGER_AUTH`, `ENTWARE_MANAGER_VERSION` |
| `/opt/web_entware/auth_config.json` | Пароль веб-панели: `{"enabled":true,"password_hash":"<sha256>"}` |
| `/opt/web_entware/version.json` | Версия проекта: `{"version":"2.6","date":"2026-07-16"}` |

## Troubleshooting

| Проблема | Решение |
|----------|---------|
| Port 8087 занят | `netstat -tlnp \| grep 8087` → изменить `server.port` в `/opt/etc/lighttpd/lighttpd.conf` |
| `mod_cgi.so` не найден | `opkg install lighttpd-mod-cgi` |
| CGI выдаёт 500 | Проверить `/opt/var/log/lighttpd/error.log` — часто: забыт `chmod +x`, неверный shebang, ошибка в sh |
| Пароль не принимается | Проверить `/opt/web_entware/auth_config.json` — хэш должен быть SHA-256 |
| Файлы не отображаются | Путь должен быть под `/tmp/` (безопасность) |
| lighttpd не стартует | `lighttpd -D -f /opt/etc/lighttpd/lighttpd.conf` — увидишь ошибку конфига |

**Логи:**

```sh
# lighttpd
tail -f /opt/var/log/lighttpd/error.log

# Entware Manager (сегодня)
cat /tmp/entware/logs/$(date +%Y-%m-%d).log
tail -f /tmp/entware/logs/$(date +%Y-%m-%d).log

# Лог установки
cat /tmp/entware/install-logs/install-*.log
```

### Что смотреть при ошибках

| Симптом | Какой лог смотреть | Что искать |
|---------|-------------------|-----------|
| Страница 500 / белый экран | `/opt/var/log/lighttpd/error.log` | `(mod_cgi.c.xxx) write failed`, `(mod_cgi.c.xxx) pipe failed`, `(mod_cgi.c.xxx) response not sent` |
| Go-бинарник не запускается | `/opt/var/log/lighttpd/error.log` | `execve failed: No such file`, signal name (SEGV, ABRT) |
| Не устанавливаются пакеты | `/tmp/entware/logs/$(date +%Y-%m-%d).log` | `opkg returned` с кодом ошибки |
| Температура пустая / null | `/tmp/entware/logs/$(date +%Y-%m-%d).log` | `rci request failed`, `parse error`, `localhost` |
| Сеть не отображается (пусто) | `/tmp/entware/logs/$(date +%Y-%m-%d).log` | `exec: ... failed`, `exit status` |
| Установка прервалась | `/tmp/entware/install-logs/install-*.log` | `✗`, `fail`, `ошибка` |
| После перезагрузки не работает | `/opt/var/log/lighttpd/error.log` | `binding failed` (порт занят другим процессом) |

## Документация

- `doc/INSTALL.md` — подробная установка, обновление, удаление
- `doc/ARCHITECTURE.md` — детальная схема, потоки данных
- `doc/API.md` — полное описание CGI API (параметры, ответы, коды ошибок)
- `doc/OPERATIONS.md` — cron, backup/restore, регламенты
- `doc/SECURITY.md` — ограничения доступа, права, безопасность
- `doc/TROUBLESHOOTING.md` — расширенные кейсы диагностики
- `doc/CHANGELOG.md` — история изменений
- `doc/NETWORK.md` — модуль "Сеть": интерфейсы, маршруты, ARP, мониторинг

## Лицензия

MIT — используй, меняй, распространяй.