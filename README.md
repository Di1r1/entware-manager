# Entware Manager

[![ShellCheck](https://github.com/Di1r1/entware-manager/actions/workflows/shellcheck.yml/badge.svg)](https://github.com/Di1r1/entware-manager/actions/workflows/shellcheck.yml)
[![Version](https://img.shields.io/badge/version-1.06.1-blue)](version.json)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Веб-панель управления Entware на роутерах Keenetic. Управление пакетами, сервисами, логами, файлами, мониторинг WiFi/температуры, веб-терминал — всё в браузере.

## Быстрый старт

### Установка с GitHub Releases

```sh
# На роутере (определите архитектуру):
uname -m
# → aarch64 → arm64
# → armv7l/armv6l/armv5tejl/armv5tel → arm
# → mips    → mips
# → mipsel  → mipsel
# → x86_64  → amd64
# → i686    → 386

# Если нет curl: opkg update && opkg install curl

# Скачайте и установите (пример для arm64):
cd /opt/tmp
curl -LO https://github.com/Di1r1/entware-manager/releases/download/v1.06.1/entware-manager-v1.06.1-arm64.tar.gz
tar -xzf entware-manager-v1.06.1-arm64.tar.gz
cd deploy && sh Install/install.sh
```

После установки открой: `http://<ip-роутера>:8087/entware-manager/`

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
index.html (SPA)  ──fetch──▶  cgi-bin/*.cgi  ──. lib/common.sh──▶  система / Entware
     │                         │                    │
  static                    lighttpd              общие функции:
  файлы                   mod_cgi (.cgi→/bin/sh)  - auth, params, json
                           port 8087              - html_escape, logging
                                                - daemon helpers
                                                - human_size, version
```

**Технологии:** POSIX `sh` (BusyBox ash), `lighttpd` + `mod_cgi`, `jq` для JSON, `ttyd` для терминала.

## Компоненты

| CGI | Назначение | Основные действия |
|-----|------------|-------------------|
| `packages.cgi` | Пакеты Entware | `list`, `install`, `remove`, `update`, `upgrade`, `upgradable` |
| `services.cgi` | Системные сервисы | `list`, `start`, `stop`, `restart`, `enable`, `disable`, `status` |
| `system_info.cgi` | Инфо о системе | CPU, RAM, диск, uptime, load, версия |
| `log_viewer.cgi` | Логи | `list`, `read`, `tail`, `clear`, `download` |
| `file_manager.cgi` | Файловый менеджер | `list`, `read`, `write`, `delete`, `mkdir`, `upload`, `download` |
| `wifi_temp.cgi` | WiFi + температура | Текущие значения с датчиков |
| `temp_history.cgi` | История температур | `hours=1..24` — JSON массив точек |
| `ttyd_control.cgi` | Веб-терминал | `start`, `stop`, `status`, `restart` |
| `auth_config.cgi` | Настройки пароля | `GET` — статус, `POST` — включить/сменить пароль |
| `view_file.cgi` | Просмотр файлов | `?path=...` — JSON (XHR) или HTML |

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
- lighttpd: `/opt/var/log/lighttpd/error.log`
- Entware Manager: `/tmp/entware/logs/YYYY-MM-DD.log`

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