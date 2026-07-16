# Журнал разработки

Правила проекта: [`RULES.md`](RULES.md)

## [2026-07-16] Исправление #1 и #2 (Content-Type в CGI)

- **Что сделано**: Исправлены два CGI без Content-Type:
  - `links_load.cgi`: `cat` → `json_out()` — добавлен заголовок JSON
  - `api.cgi:43`: `echo...jq` → `json_out()` — заголовок на успешном ответе
- **Затронутые модули**: `cgi-bin/links_load.cgi`, `cgi-bin/api.cgi`
- **Версия**: 1.03.14
- **Статус**: проверено на роутере, работает

## [2026-07-16] #5 — Удаление хардкода IP 192.168.3.1

- **Что сделано**: Все жёстко заданные IP 192.168.3.1 заменены на динамическое определение:
  - `entware.js`: `BASE_URL = window.location.hostname`, `getDefaultLinks()` вместо `DEFAULT_LINKS`
  - `links_load.cgi`: `ROUTER_IP = hostname -I` с fallback
  - `install.sh`, `backup.sh`: динамический IP в сообщениях
  - `TECH_SPEC.md`, `Install/Install.txt`: IP → `<IP_роутера>`
- **Затронутые модули**: `entware.js`, `cgi-bin/links_load.cgi`, `Install/install.sh`, `backup.sh`, `TECH_SPEC.md`, `Install/Install.txt`
- **Версия**: 1.03.15

## [2026-07-16] #4 + #18 — Удаление дубликата escapeHTML

- **Что сделано**: Удалён дубликат `escapeHTML()` из `entware.js` (не экранировал `'`). Все вызовы переведены на `escapeHtml()` из `lib/utils.js`. `network.js`: `this.escapeHtml()` делегирует глобальной `escapeHtml()`.
- **Затронутые модули**: `entware.js`, `network.js`, `lib/utils.js`
- **Версия**: 1.03.16

## [2026-07-16] Этап 1 — #3, #12, #13, #14, #16

- **Что сделано**:
  - #3: удалён дубликат `get_wifi_status()` в network_status.cgi
  - #12: `printf '%b'` → `url_decode()` в 13 местах
  - #13: ручной POST-парсинг → `post_param()` в 5 CGI
  - #14: crontab.cgi подключён к common.sh
  - #16: ручной Content-Type → `json_out()`/`html_header()` в 17 CGI
- **Затронутые модули**: ~30 CGI файлов
- **Версия**: 1.03.17

## [2026-07-16] BUGFIX: post_param() терял POST-данные в subshell

- **Что сделано**: Исправлен критический баг в `post_param()` — при вызове через `$(post_param ...)` каждый вызов создавал новый subshell, кэш `_POST_CACHED` терялся, все последующие вызовы возвращали пустоту. `post_param()` переведён на внешнюю переменную `$_POST_BODY`. Все CGIs с POST читают stdin один раз. `ttyd_control.cgi` починен (работал "некорректный порт"). Убран двойной `url_decode`. grep `\+` → `*` для BusyBox. `kill_pid.cgi` исправлен GET/POST.
- **Затронутые модули**: `lib/common.sh`, `cgi-bin/ttyd_control.cgi`, `cgi-bin/crontab_update.cgi`, `cgi-bin/monitor/monitor_action.cgi`, `cgi-bin/service_action.cgi`, `cgi-bin/kill_pid.cgi`
- **Версия**: 1.03.18

## [2026-07-16] #6 — cache-buster в network.js

- **Что сделано**: Добавлен `?_=Date.now()` во все 6 fetch-запросов network.js, чтобы браузер не кэшировал ответы CGI.
- **Затронутые модули**: `network.js`
- **Версия**: 1.03.19

## [2026-07-16] #19–23 — JS-рефакторинг

- **Что сделано**:
  - `lib/utils.js`: добавлены константы `API_BASE` (`/entware-cgi`), `UI_BASE` (`/entware-manager`), `ICONS`. Функция `initTableSearch(inputId, tableId, cellIndex)`.
  - `cgi-bin/monitor/monitor_status.cgi`: `demon_*` → `daemon_*` (опечатка)
  - `monitor.js`: `demon` → `daemon`, все fetch через `API_BASE`, удалён хардкод `log_file/log_max_size` из saveConfig
  - `network.js`: удалён `escapeHtml()` wrapper (делегировал глобальной), все fetch через `API_BASE`
  - `entware.js`: все fetch через `API_BASE`, дубликаты `initPackagesSearch`/`renderAvailableTable` заменены на `initTableSearch()` (+40 строк → +10)
  - `menu/menu.js`: fetch через `UI_BASE`
- **Затронутые модули**: `lib/utils.js`, `monitor.js`, `network.js`, `entware.js`, `menu/menu.js`, `cgi-bin/monitor/monitor_status.cgi`
- **Версия**: 1.03.25

- **Что сделано**: Добавлена функция `parse_log_events(tag, limit)` в `common.sh` — читает дневной лог, фильтрует по тегу, парсит строки в JSON-массив событий (timestamp, level, service, event, details). JSON-экранирование через sed.
- `network/events.cgi` и `service_watchdog/events.cgi` сокращены с ~60 строк до 10 строк (один вызов `parse_log_events`)
- **Затронутые модули**: `lib/common.sh`, `cgi-bin/network/events.cgi`, `cgi-bin/service_watchdog/events.cgi`
- **Версия**: 1.03.23

## [2026-07-16] #9 fix — undefined log(), log-viewer CGIs

- **Что сделано**:
  - watchdog.sh: `log "INFO" "Лог ротирован"` → `log_message "INFO" "[monitor] Лог ротирован"` (вызывал "not found" при ротации)
  - `monitor_log.cgi`: теперь читает из `/tmp/entware/logs/YYYY-MM-DD.log`, фильтр `[monitor]`
  - `network/events.cgi`, `service_watchdog/events.cgi`: переведены на дневной лог, grep -i, регистронезависимый парсинг
- **Затронутые модули**: `watchdog.sh`, `cgi-bin/monitor/monitor_log.cgi`, `cgi-bin/network/events.cgi`, `cgi-bin/service_watchdog/events.cgi`
- **Версия**: 1.03.22

- **Что сделано**:
  - `log()` в watchdog.sh → `log_message()` (удалена локальная функция)
  - `log_event()` в network_watchdog.sh → `log_message()` (удалена локальная функция)
  - `log_service()` в service_watchdog.sh → `log_message()` (удалена локальная функция)
  - `log_action()` в common.sh: fallback теперь делегирует `log_message()` вместо дублирования mkdir/echo
  - Все `log_action()` в start/stop/restart всех 3 демонов → `log_message()`
  - Формат сообщений: `[модуль] подсистема: событие (детали)`
  - Все пишут в `/tmp/entware/logs/YYYY-MM-DD.log` через единый `log_message()`
- **Затронутые модули**: `lib/common.sh`, `watchdog.sh`, `network_watchdog.sh`, `service_watchdog.sh`
- **Версия**: 1.03.21

## [2026-07-17] Модуль SMART — Этап 1+2 (UI + бэкенд)

- **Что сделано**: Добавлен полноценный модуль SMART-мониторинга дисков:
  - `lib/smart.sh` — библиотека: обнаружение дисков через `/proc/partitions`, вызов `smartctl` через `sudo`, парсинг атрибутов/health/температуры/самотестов
  - `cgi-bin/smart.cgi` — API: `list`, `info`, `attributes`, `health`, `selftest` (GET/POST)
  - `smart.js` — UI-таб: таблица дисков, модалки атрибутов/health, запуск тестов (поллинг статуса)
  - `icons.svg` — иконка `#icon-hdd`
  - `lib/utils.js` — функция `loadScript()` для динамической загрузки модулей
  - `build-deploy.sh` — копирование `lib/*.sh` в deploy
  - `Install/install.sh` — добавлены `sudo`, `smartmontools`, `smartmontools-drivedb` + sudoers для `nobody`
- **Затронутые модули**: `lib/smart.sh` (новый), `cgi-bin/smart.cgi` (переписан), `smart.js` (новый), `icons.svg`, `menu/menu.json`, `entware.js`, `lib/utils.js`, `build-deploy.sh`, `Install/install.sh`, `version.json`, `doc/CHANGELOG.md`
- **Версия**: 1.04.00
