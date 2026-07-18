# Entware Manager

[![ShellCheck](https://github.com/Di1r1/entware-manager/actions/workflows/shellcheck.yml/badge.svg)](https://github.com/Di1r1/entware-manager/actions/workflows/shellcheck.yml)
[![Version](https://img.shields.io/badge/version-2.6-blue)](version.json)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Веб-панель управления Entware на роутерах Keenetic. Управление пакетами, сервисами, логами, файлами, мониторинг WiFi/температуры, веб-терминал — всё в браузере.

## Быстрый старт

```sh
# На роутере (Entware уже установлен):
opkg update && opkg install git-http
git clone https://github.com/Di1r1/entware-manager.git /opt/tmp/entware-manager
cd /opt/tmp/entware-manager/Install
chmod +x install.sh
./install.sh
```

После установки открой: `http://<ip-роутера>:8087/entware-manager/`

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

## Разработка

### Сборка deploy-папки

```sh
./build-deploy.sh [--tar]
# → создаёт ./deploy/ (или deploy.tar.gz)
```

### Деплой на роутер (SMB)

```sh
smbclient //192.168.3.1/Entware_USB -U 'USER%PASS' <<'EOF'
put deploy/lib/common.sh web_entware/lib/common.sh
put deploy/cgi-bin/*.cgi web_entware/cgi-bin/
# ... остальные файлы
EOF
```

### Конвенции кода

- **POSIX sh** — никаких bashism (`local` только в функциях, никаких `$'...'`, `[[ ]]`, `==`)
- **Общие функции** — только через `. /opt/web_entware/lib/common.sh`
- **JSON ответ** — `json_out '{"status":"ok"}'` (вызывает `exit 0`)
- **HTML экранирование** — всегда `html_escape "$var"` перед выводом в HTML
- **Auth** — `check_filemgr_auth "password"` в начале CGI, требующих пароль
- **Параметры** — `get_param "key" "default"` (GET), `post_param "key" "default"` (POST)

### Добавить новый CGI

1. Создать `cgi-bin/new_feature.cgi` с shebang `#!/bin/sh`
2. Подключить `lib/common.sh`
3. Использовать `get_param`/`post_param`, `json_out`/`html_header`
4. Добавить в `build-deploy.sh` (автоматически подхватится по маске `cgi-bin/*.cgi`)
5. Пересобрать и задеплоить

## Тестирование

### Smoke-тесты (на роутере)

```sh
# Запуск всех базовых проверок
sh /opt/web_entware/test/smoke.sh

# Или вручную:
curl -s http://localhost:8087/entware-cgi/version.cgi | jq .
curl -s http://localhost:8087/entware-cgi/packages.cgi?action=list | jq .
```

Тесты проверяют: HTTP 200, валидный JSON, работа auth, основные эндпоинты.

### Линтинг (CI)

```sh
shellcheck -x lib/common.sh Install/install.sh cgi-bin/*.cgi *.sh logger/**/*.sh
```

GitHub Actions: `.github/workflows/shellcheck.yml` — запускается на каждый push/PR.

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