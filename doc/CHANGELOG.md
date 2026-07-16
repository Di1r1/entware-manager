# Изменения проекта

## 1.03.12 (2026-07-07)

### Стандартизация путей и исправление lighttpd

- **install.sh полностью переписан**:
  - Безопасное дополнение конфига lighttpd — `server.modules +=`, `alias.url +=`, `static-file.exclude-extensions += .cgi` (с проверкой через `grep -q`, дубли не создаются)
  - Патч `/opt/etc/lighttpd/conf.d/30-cgi.conf` — `cgi.assign` и `cgi.execute-x-only = "enable"` устанавливаются только здесь
  - Удалён код, удалявший чужие `alias.url`, `mod_alias`, `mod_cgi`
  - Не удаляет чужие настройки

- **Исправление lighttpd**:
  - `cgi.assign` удалён из `main.conf` (дубль валил lighttpd)
  - `static-file.exclude-extensions` содержит `.cgi`
  - `mod_alias`/`mod_cgi` добавляются через `+=` без дублирования

- **Стандартизированы пути директорий**:
  - `logs/` — `/tmp/entware/logs/`
  - `pid/` — `/tmp/entware/pid/`
  - `counters/` — `/tmp/entware/counters/`
  - `counters_ignore/` — `/tmp/entware/counters_ignore/`
  - `temp_history/` — `/tmp/entware/temp_history/`

- **Все watchdog-скрипты**:
  - `mkdir -p` для всех стандартизированных директорий
  - `nohup` заменён на `sh ... &` (BusyBox на Keenetic не содержит `nohup`)

- **Исправление stats.cgi**:
  - `ps -e -o rss,comm` заменён на чтение `/proc/[pid]/status` (VmRSS) — BusyBox `ps` не поддерживает `-e -o`
  - stats.cgi теперь работает на Keenetic

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `install.sh` | Полностью переписан: безопасное дополнение конфига, патч 30-cgi.conf |
| `network_watchdog.sh` | `nohup` → `sh &`, `mkdir -p` для стандартных путей |
| `service_watchdog.sh` | `nohup` → `sh &`, `mkdir -p` для стандартных путей |
| `watchdog.sh` | `mkdir -p` для стандартных путей |
| `cgi-bin/stats.cgi` | Чтение VmRSS из /proc/[pid]/status вместо ps |
| `version.json` | 1.03.12 |

---

## 1.03.14 (2026-07-16)

### Исправление Content-Type в CGI

- **links_load.cgi**: `cat` → `json_out()` — добавлен HTTP-заголовок `Content-Type: application/json`, браузер корректно распознаёт JSON
- **api.cgi:43**: `echo ... | jq` → `json_out()` — исправлен fallthrough без заголовка при успешном ответе

### Правила для LLM-ассистента

- **devlog.md**: обновлены правила до v2.1:
  - Разделение `set -eu` для CGI (запрещён) и демонов/утилит (обязателен)
  - Таблица статусов всех функций из `common.sh` (✅/❌)
  - Добавлены пункты 12 (единая обработка ошибок CGI) и 13 (единый Content-Type)

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `cgi-bin/links_load.cgi` | cat → json_out() |
| `cgi-bin/api.cgi` | echo...jq → json_out() |
| `devlog.md` | Правила v2.1 |
| `version.json` | 1.03.14 |

---

## 1.03.18 (2026-07-16)

### BUGFIX: post_param() терял POST-данные в subshell

- **Критический баг**: `post_param()` кэшировал POST-данные через `_POST_CACHED`, но при вызове через `$(post_param ...)` каждый вызов создаёт новый subshell, где кэш теряется. После первого `post_param` stdin пустел, все последующие возвращали пустоту.
- **Исправление**: `post_param()` переведён на внешнюю переменную `$_POST_BODY`. CGIs с POST должны читать stdin один раз: `_POST_BODY=$(cat); export _POST_BODY`
- **Затронуты**: `ttyd_control.cgi` (починена работа ttyd), `crontab_update.cgi`, `monitor_action.cgi`, `service_action.cgi`, `kill_pid.cgi`
- **`ttyd_control.cgi`**: убран двойной `url_decode` (post_param уже декодирует)
- **`kill_pid.cgi`**: исправлен мёртвый код (GET-парсинг внутри POST-блока)
- **grep**: `\+` → `*` (BusyBox-совместимость)

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `lib/common.sh` | post_param() → $_POST_BODY |
| `cgi-bin/ttyd_control.cgi` | _POST_BODY, убран url_decode, grep \+ → * |
| `cgi-bin/crontab_update.cgi` | _POST_BODY, убран url_decode |
| `cgi-bin/monitor/monitor_action.cgi` | _POST_BODY, убран url_decode |
| `cgi-bin/service_action.cgi` | _POST_BODY |
| `cgi-bin/kill_pid.cgi` | _POST_BODY + исправлен GET/POST |
| `version.json` | 1.03.18 |

---

## 1.03.17 (2026-07-16)

- **#3**: Удалён дубликат `get_wifi_status()` в `network_status.cgi` (строки 282–288)
- **#12**: `printf '%b'` → `url_decode()` из common.sh в 13 местах (tmpfs.cgi, ttyd_control.cgi, monitor_action.cgi, crontab.cgi/update.cgi, logger/*.cgi)
- **#13**: Ручной POST-парсинг (`cat | sed`) → `post_param()` в 5 CGI (crontab_update.cgi, ttyd_control.cgi, monitor_action.cgi, service_action.cgi, kill_pid.cgi)
- **#14**: `crontab.cgi` подключён к common.sh (добавлен `. /opt/web_entware/lib/common.sh`, использует `get_param`/`json_out`)
- **#16**: Ручные `echo "Content-type:..."` заменены на `json_out()` / `html_header()` в 17 CGI

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `cgi-bin/network_status.cgi` | #3: удалён дубликат |
| 8 файлов | #12: printf '%b' → url_decode |
| 5 файлов | #13: POST_DATA → post_param |
| `cgi-bin/crontab.cgi` | #14: подключен common.sh |
| 17 файлов | #16: ручной Content-Type → json_out/html_header |
| `version.json` | 1.03.17 |

---

## 1.03.16 (2026-07-16)

- **entware.js**: удалена `escapeHTML()` — не экранировала `'`, создавала XSS-уязвимость. Все вызовы переведены на `escapeHtml()` из `lib/utils.js`
- **network.js**: `this.escapeHtml()` теперь делегирует глобальной `escapeHtml()` из `lib/utils.js`

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `entware.js` | Удалена escapeHTML(), escapeHTML(...) → escapeHtml(...) |
| `network.js` | this.escapeHtml() → делегат на глобальную escapeHtml() |
| `version.json` | 1.03.16 |

---

## 1.03.15 (2026-07-16)

### Удаление хардкода IP 192.168.3.1

- **entware.js**: добавлена константа `BASE_URL = window.location.protocol + '//' + window.location.hostname`; все жёсткие IP заменены на `BASE_URL`; `DEFAULT_LINKS` → функция `getDefaultLinks()`
- **links_load.cgi**: IP определяется через `hostname -I`, fallback 192.168.3.1
- **install.sh, backup.sh**: сообщения с IP генерируются динамически
- **TECH_SPEC.md, Install/Install.txt**: IP заменён на плейсхолдер `<IP_роутера>`

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `entware.js` | BASE_URL, getDefaultLinks() вместо DEFAULT_LINKS |
| `cgi-bin/links_load.cgi` | ROUTER_IP динамический |
| `Install/install.sh` | IP через hostname -I |
| `backup.sh` | IP через hostname -I |
| `TECH_SPEC.md` | IP → `<IP_роутера>` |
| `Install/Install.txt` | IP → `<IP_роутера>` |
| `version.json` | 1.03.15 |

---

## 1.03.14 (2026-07-16)

### Исправление Content-Type в CGI

- **links_load.cgi**: `cat` → `json_out()` — добавлен HTTP-заголовок `Content-Type: application/json`, браузер корректно распознаёт JSON
- **api.cgi:43**: `echo ... | jq` → `json_out()` — исправлен fallthrough без заголовка при успешном ответе

### Правила для LLM-ассистента

- **devlog.md**: обновлены правила до v2.1:
  - Разделение `set -eu` для CGI (запрещён) и демонов/утилит (обязателен)
  - Таблица статусов всех функций из `common.sh` (✅/❌)
  - Добавлены пункты 12 (единая обработка ошибок CGI) и 13 (единый Content-Type)

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `cgi-bin/links_load.cgi` | cat → json_out() |
| `cgi-bin/api.cgi` | echo...jq → json_out() |
| `devlog.md` | Правила v2.1 |
| `version.json` | 1.03.14 |

---

## 1.03.13 (2026-07-07)

### Исправление демона защиты от зависших процессов

- **watchdog.sh полностью переписан**:
  - Добавлен единый интерфейс `start|stop|restart|status|daemon` (как у network_watchdog/service_watchdog)
  - `start` читает конфиг ДО запуска — если `enabled: false`, отказывает с ошибкой, не создаёт PID-файл
  - `daemon_loop` убрана проверка `ENABLED` — демон больше не выходит сам после старта
  - PID-файл создаётся в `start`, а не внутри `daemon_loop`
  - Использованы `pid_is_alive()` и `find_pids()` из common.sh

- **monitor_action.cgi**:
  - Вместо `$DEMON_SCRIPT >> log &` вызывает `watchdog.sh start`/`stop`
  - После `start` делает `sleep 1` и проверяет `watchdog.sh status`
  - `kill -0` → `pid_is_alive()`

- **monitor_status.cgi**:
  - `-d "/proc/$pid"` → `pid_is_alive()` (зомби-процессы больше не показывают running)

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `watchdog.sh` | Полностью переписан: интерфейс start/stop/restart/status |
| `cgi-bin/monitor/monitor_action.cgi` | Использует watchdog.sh start/stop |
| `cgi-bin/monitor/monitor_status.cgi` | pid_is_alive вместо -d /proc/$pid |
| `lib/common.sh` | Добавлены pid_is_alive(), find_pids() |
| `version.json` | 1.03.13 |

---

## 1.03.10 (2026-06-02)

### Новые функции

- **Убийство процессов и отображение дублей PID в службах**:
  - `cgi-bin/services.cgi v3.8` — полностью переписан поиск PID, теперь возвращает все PID процесса (поле `pids`). При нескольких процессах у службы показывается первый PID + кликабельный бейдж `+N`
  - `cgi-bin/kill_pid.cgi v1.0` — новый CGI для принудительного завершения процесса по PID (`kill -9`) с валидацией и логированием
  - `entware.js` — `renderServices()` обновлена: отображение дублей, кликабельный PID открывает модалку `showProcessList()` со списком всех процессов и кнопками "Убить"
  - `style.css` — `.pid-link`, `.pid-badge`, `.process-list`, `.process-item`, `.process-kill-btn`

- **Физические порты + сети на главной странице**:
  - `cgi-bin/network_status.cgi` — старый блок "LAN порты" (жёстко зашитые Порт 0/1/2/3/4/5) заменён на "Физические порты" (детектирует `eth*` интерфейсы, читает `carrier` и `speed` из sysfs) + "Сети" (парсит `brctl show` для мостов и определяет WAN)
  - `entware.js` — `loadNetworkStatus()` обновлена под новый формат данных

- **Блочные устройства — кликабельные, открываются в файловом менеджере**:
  - `cgi-bin/stats.cgi` — точки монтирования блочных устройств теперь `<a href>` на `tmpfs.cgi`
  - `cgi-bin/tmpfs.cgi v2.0` — снято ограничение на пути (работало только `/tmp`, `/dev`, `/dev/shm`). Теперь открывается любая существующая директория

- **Топ процессов по памяти в карточке RAM**:
  - `cgi-bin/stats.cgi` — добавлен сбор топ-5 процессов по RSS (через `ps -e -o rss,comm`), вывод компактной таблицы в карточку памяти
  - `style.css` — `.top-mem-wrapper` с hover-эффектом: подъём, тень и подсветка строк цветом, зависящим от загрузки RAM (normal→фиолетовый, warning→янтарный, critical→красный). Уменьшены отступы карточки памяти для компенсации

### Исправления

- **Плавность меню**:
  - `style.css` — `.sidebar .menu li` transition теперь включает `transform` и `box-shadow` (было только `padding`/`margin` из-за более высокой специфичности `.sidebar .menu li`). Ховер на пунктах меню стал плавным
  - `entware.js` — таймаут `menu-animate` увеличен с 500ms до 1400ms, чтобы pop-in анимация успевала завершиться для всех 15 пунктов меню

### Обновлённые файлы

| Файл | Версия | Описание |
|------|--------|----------|
| `cgi-bin/services.cgi` | 3.8 | Возвращает все PID (поле pids). Поддержка дублей |
| `cgi-bin/kill_pid.cgi` | 1.0 | Новый CGI: убийство процесса по PID |
| `cgi-bin/tmpfs.cgi` | 2.0 | Снято ограничение на пути |
| `cgi-bin/network_status.cgi` | — | Физические порты + сети вместо LAN портов |
| `cgi-bin/stats.cgi` | 0.26 | Кликабельные блочные устройства + топ по памяти |
| `entware.js` | — | PID бейджи, showProcessList, killProcess, обновлён network |
| `style.css` | — | PID badge, процесс-лист, top-mem, исправлен menu hover |
| `version.json` | 1.03.10 | Обновлено |

---

## 1.03 (2026-05-03)

### Новые функции

- **Модуль проверки системных зависимостей**:
  - `lib/common.sh v2.6` — добавлена функция `check_deps_logic()` для проверки всех зависимостей
  - `cgi-bin/check_deps.cgi v1.0` — новый CGI-скрипт проверки системы (cron, jq, iproute2, lighttpd)
  - `cgi-bin/check_syntax.cgi v1.1` — скрипт проверки синтаксиса всех CGI и библиотек
  - `entware.js v0.83` — добавлена функция `checkSystemDeps()` с красивым выводом статуса в модальном окне
  - `index.html` — добавлена кнопка "Проверка системы" в сайдбар

### Что проверяется

- **Базовые компоненты**: opkg, lighttpd (работает ли)
- **Утилиты BusyBox**: sed, awk, grep, ps
- **Пакеты Entware**: cron (установлен/запущен), jq, iproute2 (ip)
- **Статус разделов**: packages, services, monitoring, network, logger

### Интеграция

- Функция `check_deps_logic()` в `common.sh` может быть вызвана любым CGI-скриптом
- Кнопка в сайдбаре открывает модальное окно с цветовой индикацией статуса
- Рекомендации по установке отсутствующих пакетов

---

## 1.02 (2026-04-08)

### Исправления истории температур

- **temp_history.cgi v1.8** — исправлен парсинг:
  - Проблема: "temp":62},{"time":"23:59:33","temp":6200 - лишние данные из-за переноса строк между файлами
  - Решение: объединение файлов через $'\n' + надёжный awk с split()

- **wifi_temp_history.cgi v1.5** — аналогичное исправление:
  - Проблема: "temp0":71,"temp1":5700:43:38 - склейка данных из разных дней
  - Решение: такой же подход с объединением файлов

### Исправления интерфейса

- **index.html** — убран gradient с графика температуры CPU:
  - Удалена полупрозрачная заливка под графиком
  - Оставлена только чистая линия без "запыления"

---

## 1.01 (2026-04-08)

### Оптимизация производительности

- **services.cgi v3.7-fast** — полностью переписан для оптимизации:
  - Однократный сбор данных процессов (`ps` один раз в переменную)
  - Поддержка PIDFILE из скриптов init.d
  - Функция `get_var()` для извлечения PROCS/NAME/DAEMON/SCRIPT из скриптов
  - Проверка cmdline для избежания ложных срабатываний (zombie/мертвые процессы)
  - **Выигрыш: 94%** (с 3.2 сек до 0.37 сек)

- **network/status.cgi v1.7** — исправлен парсинг uptime и state file:
  - Заменён heavy `ps -o pid,etime | grep | awk` на `/proc/uptime` + `/proc/PID/stat`
  - Исправлен парсинг `last_check` из state file (`/tmp/network_watchdog_state.json`)
  - Защита от переводов строк в JSON (`tr -d '\n\r'`)
  - **Выигрыш: 60%**

- **upgradable.cgi v0.13** — исправлен парсинг:
  - Использует `$1`, `$3`, `$5` для правильного парсинга вывода `opkg list-upgradable`
  - Фильтрация строк с пустыми версиями и "-"
  - Клиентская фильтрация в `fetchUpgradable()` для отсеивания undefined

### Новые функции

- **Кнопка "Обновить все пакеты"**:
  - Новая кнопка в интерфейсе обновлений (оранжевый цвет)
  - Функция `upgradeAll()` в entware.js с подтверждением
  - Поддержка `upgrade_all=1` в upgrade.cgi
  - Запускает `opkg upgrade` без аргументов

### Исправления Service Watchdog

- **service_watchdog.sh v1.13** — улучшенная проверка PID:
  - Проверка cmdline для избежания ложных PID (kernel boot params)
  - Проверка zombie процессов через /proc/PID/status
  - Pattern для shadowsocks изменён на `ss-redir ss-local ss-server`

- **service_watchdog/events.cgi v2.4** — исправлен парсинг:
  - `tr -d '()'` удаляет все скобки из details
  - Раньше показывало `skipped (cooldown 60s))`, теперь `skipped cooldown 60s`

### Изменения интерфейса

- **entware.js** — убрано автообновление таблицы служб:
  - servicesInterval setInterval удалён
  - Таблица обновляется только при открытии вкладки и после действий пользователя
  - Значительно снижена нагрузка на роутер

### Обновлённые файлы

| Файл | Версия | Описание |
|------|--------|----------|
| `services.cgi` | 3.7-fast | Однократный ps, поддержка PIDFILE |
| `network/status.cgi` | 1.7 | Исправлен uptime + state file |
| `upgradable.cgi` | 0.13 | Исправленный парсинг $1,$3,$5 |
| `upgrade.cgi` | 0.03 | Поддержка upgrade all |
| `service_watchdog.sh` | 1.13 | Проверка cmdline + zombie |
| `service_watchdog/events.cgi` | 2.4 | tr -d '()' |
| `entware.js` | — | Кнопка "Обновить все", фильтрация undefined |
| `version.json` | 1.01 | Обновлено |

### Итог производительности

| Метрика | До | После | Улучшение |
|---------|-----|-------|-----------|
| services.cgi | 3.2 сек | 0.37 сек | **8.6x** |
| Общая нагрузка | ~42 сек/мин | ~4 сек/мин | **10x** |

---

## 0.68 (2026-04-06)

### Новые функции
- **Мониторинг служб (service watchdog)** — обновлённый функционал
  - **Автоперезапуск**: при падении службы демон автоматически перезапускает её через init.d скрипт
  - **Список исключений**: позволяет исключить определённые службы из мониторинга (по умолчанию: dropbear, kvas-ws, service_watchdog)
  - **Cooldown**: защита от повторного автоперезапуска (60 секунд), чтобы не мешать ручному управлению
  - **Режим custom**: переименован из "all", отслеживает только процессы из списка watch_list
  - **Логирование**: все изменения конфигурации и действия (start/stop/restart) записываются в основной лог

### Изменения интерфейса
- **Переключатели мониторинга**:
  - "Кастомный список" — включает режим custom с полем ввода списка процессов
  - "Исключения" — включает отображение списка исключений
  - "Автоперезапуск" — отдельный переключатель для включения автоперезапуска

### Исправления
- Исправлен парсинг JSON массивов в config.cgi (извлечение watch_list и exclude_list)
- Исправлена проблема с переключателем "Исключения" (сбрасывался при загрузке)
- Исправлена проблема с автоперезапуском (找到 неправильный init.d скрипт)
- Демон перезагружает конфигурацию при получении сигнала HUP
- service_watchdog исключает сам себя из мониторинга

### Обновлённые файлы
| Файл | Версия | Описание |
|------|--------|----------|
| `service_watchdog.sh` | 1.4 | Автоперезапуск, exclude_list, cooldown, HUP перезагрузка |
| `cgi-bin/service_watchdog/config.cgi` | 1.4 | Исправлен парсинг JSON, поддержка exclude_list |
| `cgi-bin/service_watchdog/status.cgi` | 1.2 | Возвращает exclude_list |
| `cgi-bin/service_watchdog/action.cgi` | 1.3 | Логирование всех действий |
| `cgi-bin/service_watchdog/events.cgi` | 2.0 | Исправлен парсинг details (убраны скобки) |
| `entware.js` | — | Новый UI: переключатели, поля ввода |

## 0.65 (2026-04-04)

### Новые функции
- **Модуль "Сеть" (Network)** — новая вкладка для мониторинга сети
  - Sidebar виджет статуса сети
  - Вкладка "Интерфейсы" — список сетевых интерфейсов
  - Вкладка "Маршруты" — таблица маршрутизации
  - Вкладка "ARP" — ARP таблица с именами хостов
  - Вкладка "События" — лог событий watchdog демона
  - Демон мониторинга network_watchdog.sh

### Улучшения интерфейса
- **Фильтр интерфейсов**: добавлен toggle "Скрыть неизвестные"
  - Скрывает интерфейсы со состоянием UNKNOWN
  - Полезно на устройствах с множеством виртуальных интерфейсов

### Исправления
- **Вкладки Network**: исправлена работа переключения вкладок (JavaScript)
- **action.cgi**: исправлен парсинг POST данных для управления демоном
- **ARP таблица**: добавлено определение имени хоста через getent hosts

## 0.64 (2026-04-01)

### Исправления безопасности
- **Исправлена критическая проблема**: логирование НЕ отключалось при `enabled=false`
  - Причина: `jq '// true'` трактует `false` как "falsy", возвращая `true`
  - Решение: используется явная проверка `if [ "$ENABLED_VALUE" = "true" ]`
  - Затронутые файлы: `logger/lib/logging.sh`, `lib/common.sh`
- **Исправлена проблема с PATH в CGI контексте lighttpd**:
  - CGI получал PATH: `/sbin:/usr/sbin:/bin:/usr/bin` (без `/opt/bin`)
  - Утилиты `jq`, `cat` не находились
  - Решение: использован полный путь `/opt/bin/jq`, `/opt/bin/cat`
  - Затронутые файлы: `common.sh`, `config.cgi`, `crontab_update.cgi`, `links_save.cgi`, `monitor/*.cgi`, и др.

### Версии после исправлений
| Файл | Версия | Описание |
|------|--------|----------|
| `lib/common.sh` | 2.5 | Чистая версия, использует logging.sh |
| `logger/lib/logging.sh` | 1.7 | Исправлен jq для false, чистая версия |
| `logger/config.cgi` | 1.5 | Исправлены пути |
| `cgi-bin/logger/system_log.cgi` | 1.8 | Использованы /opt/bin/cat, /opt/bin/sed |
| `cgi-bin/crontab_update.cgi` | 2.6 | Использует log_action из common.sh |
| `cgi-bin/links_save.cgi` | 0.04 | Исправлены пути |
| `monitor/*.cgi` | разные | Исправлены пути |
| `api.cgi` | 0.05 | Исправлены пути |
| `wifi_temp.cgi` | 0.04 | Исправлены пути |
| `ttyd_control.cgi` | 0.07 | Исправлены пути |
| `service_action.cgi` | 0.04 | Исправлены пути |

## 0.63 (2026-04-01)

### Новая функциональность
- Добавлен системный лог событий: `/opt/var/log/entware/system.log`
- Создан `logger/system_log.cgi` (v1.7) для просмотра системных событий
- Обновлён `logger/config.cgi` (v1.2) — теперь пишет в системный лог при включении/выключении логирования, показывает красивый вывод конфига
- Добавлена кнопка "Системные события" в UI (вкладка Логи)
- Обновлена справка `help.cgi` (v0.55) — добавлено описание системных событий

### Исправления
- **lib/common.sh v2.1**: исправлен `return` в `log_action()` — теперь не падает в fallback при выключенном логировании.
- `tmpfs.cgi v1.9`: убран `realpath`, заменён на встроенную нормализацию пути.
- `links_save.cgi v0.03`: добавлена валидация JSON через `jq empty`.
- `logger/view.cgi v1.4`: используется `index()` для поиска.

### Рефакторинг
- Создан общий модуль `lib/utils.js` с функцией `escapeHtml`.
- Удалён мёртвый код: `version.cgi`, `logger/status.cgi`.
- Удалено неиспользуемое поле `group` из конфига монитора.

### Улучшения UI
- Добавлен `?v=` к `monitor.js`.
- Удалена внешняя загрузка Google Fonts.

## 0.62 (2026-04-01)

### Исправления
- Исправлено URL-декодирование в `lib/common.sh` (`url_decode`) для корректной обработки `%2F` и других кодов.
- В ряде CGI добавлены явные завершения после отдачи JSON/ответа, чтобы исключить лишнее выполнение.
- `delete_file.cgi`: улучшены проверки и оптимизирована логика под слабые устройства.
- `wifi_temp.cgi`: исправлено формирование ответа.

### Оптимизация
- История температур CPU/WiFi: очистка старых данных запускается не на каждый запрос, а периодически.
- Улучшены отдельные участки обработки в `services.cgi`, `stats.cgi`, `upgradable.cgi`.

### Новые возможности
- Добавлены `temp_history.cgi` и `wifi_temp_history.cgi` для хранения истории температур (до 7 дней).
- В UI добавлены графики температуры CPU/WiFi.

## 0.57 (2026-03-29)

- Выделена общая библиотека CGI: `lib/common.sh`.
- Улучшена структура frontend-кода (`modal.js`, улучшение обработки ошибок).
- Доработаны `crontab_update.cgi`, `ttyd_control.cgi`.
- Обновлен `backup.sh`.

## Примечание

Каноническая текущая версия хранится в `version.json`.
