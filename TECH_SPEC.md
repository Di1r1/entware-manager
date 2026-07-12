# Entware Manager — техническое описание (актуально на 2026-05-03)

отщяться на русском языке
Все файлы в кодировке UTF-8. Не изменяй кодировку.
Не удалять файлы, а складывать в E:\Stol\smart\web_entware_v0.65_20260402\tmp если они не нужны (спрашивать разрешение)
для роутера \\KEENETIC-9989\Entware_USB\web_entware\tmp
добавлять изменения в журнал версий и в документацию

## 1. Общие сведения
Entware Manager — веб-интерфейс для управления пакетами Entware, системными службами, мониторинга процессов и логирования на роутерах под управлением OpenWrt / Entware.

- **Версия интерфейса**: 1.03 (хранится в `/opt/web_entware/version.json`)
- **Веб-сервер**: lighttpd, порт 8087
- **Документация**: данная спецификация
- **Оптимизация**: v1.01 включает значительные улучшения производительности (services.cgi однократный ps), исправления парсинга (upgradable.cgi), добавлена кнопка "Обновить все пакеты"
- **Новое в v1.03**: Модуль проверки системных зависимостей (check_deps.cgi, check_deps_logic в common.sh)

---

## 2. Структура каталогов и файлов (полные пути)

### 2.1. Корневой каталог `/opt/web_entware/`

| Файл | Назначение |
|------|------------|
| `/opt/web_entware/backup.sh` | Скрипт резервного копирования (версия 2.2, адаптирован под новую структуру) |
| `/opt/web_entware/entware.js` | Основной JavaScript код приложения (версия 0.61) |
| `/opt/web_entware/modal.js` | Модуль модальных окон и тостов (вынесен для чистоты кода) |
| `/opt/web_entware/monitor.js` | Модуль защиты (клиентская часть) |
| `/opt/web_entware/watchdog.sh` | Демон защиты (фоновый процесс мониторинга) |
| `/opt/web_entware/icons.svg` | SVG-спрайт всех иконок |
| `/opt/web_entware/index.html` | Главная страница (подключает modal.js) |
| `/opt/web_entware/style.css` | Основной файл стилей (темы, адаптивность) |
| `/opt/web_entware/version.json` | Версия и дата интерфейса |
| `/opt/web_entware/links.json` | Пользовательские ссылки (сохраняются через интерфейс) |
| `/opt/web_entware/monitor_config.json` | Конфигурация демона защиты |

### 2.2. Подкаталог `lib/` — общие функции

| Файл | Назначение |
|------|------------|
| `/opt/web_entware/lib/common.sh` | Библиотека общих функций для CGI (версия 2.6, включает PID в логи, декодирование URL, санитайз, вывод JSON/HTML, get_version, check_deps_logic) |

**Новые функции в common.sh (v2.6):**
- `check_deps_logic()`: Проверка системных зависимостей (opkg, lighttpd, cron, jq, iproute2, утилиты BusyBox). Возвращает JSON со статусом базы, зависимостей и разделов.

### 2.3. Подкаталог `logger/` — модуль логирования

| Файл | Назначение |
|------|------------|
| `/opt/web_entware/logger/config.json` | Включение/отключение записи логов действий менеджера |
| `/opt/web_entware/logger/style.css` | Стили для вкладки «Логи» |
| `/opt/web_entware/logger/system_sources.json` | Статические источники системных логов |
| `/opt/web_entware/logger/lib/logging.sh` | Библиотека функций логирования (версия 1.1, добавлен PID) |
| `/opt/web_entware/logger/scripts/rotate.sh` | Скрипт ежедневной ротации логов (12:00) |

### 2.4. Подкаталог `menu/` — динамическое меню

| Файл | Назначение |
|------|------------|
| `/opt/web_entware/menu/menu.json` | Конфигурация пунктов меню |
| `/opt/web_entware/menu/menu.js` | Модуль загрузки и отрисовки меню |

### 2.5. Подкаталог `cgi-bin/` — все CGI-скрипты

#### 2.5.1. Основные CGI-скрипты (обновлены, используют common.sh)

| Файл | Метод | Формат | Назначение |
|------|-------|--------|------------|
| `/opt/web_entware/cgi-bin/api.cgi` | GET | JSON | Информация о пакете |
| `/opt/web_entware/cgi-bin/check_deps.cgi` | GET | JSON | Проверка системных зависимостей (v1.0, использует check_deps_logic из common.sh) |
| `/opt/web_entware/cgi-bin/check_syntax.cgi` | GET | JSON | Проверка синтаксиса всех CGI и библиотек (v1.1) |
| `/opt/web_entware/cgi-bin/available.cgi` | GET | JSON | Список доступных пакетов |
| `/opt/web_entware/cgi-bin/crontab.cgi` | GET | JSON | Чтение crontab (system / opt) |
| `/opt/web_entware/cgi-bin/crontab_update.cgi` | POST | JSON | Сохранение crontab (оригинальная логика) |
| `/opt/web_entware/cgi-bin/delete_file.cgi` | POST | JSON | Удаление файла/папки в tmpfs (v0.06, без realpath) |
| `/opt/web_entware/cgi-bin/help.cgi` | GET | HTML | Справка |
| `/opt/web_entware/cgi-bin/install.cgi` | POST | HTML | Установка пакета |
| `/opt/web_entware/cgi-bin/links_load.cgi` | GET | JSON | Загрузка пользовательских ссылок |
| `/opt/web_entware/cgi-bin/links_save.cgi` | POST | JSON | Сохранение пользовательских ссылок (v0.03, валидация JSON) |
| `/opt/web_entware/cgi-bin/packages.cgi` | GET | HTML | Список установленных пакетов |
| `/opt/web_entware/cgi-bin/remove.cgi` | POST | HTML | Удаление пакета |
| `/opt/web_entware/cgi-bin/services.cgi` | GET | JSON | Список служб (init.d) (v3.7-fast, однократный ps, поддержка PIDFILE) |
| `/opt/web_entware/cgi-bin/service_action.cgi` | POST | JSON | Управление службой (start/stop/restart/enable/disable) |
| `/opt/web_entware/cgi-bin/stats.cgi` | GET | HTML | Статистика системы |
| `/opt/web_entware/cgi-bin/temperature.cgi` | GET | JSON | Температура CPU |
| `/opt/web_entware/cgi-bin/tmpfs.cgi` | GET | HTML | Файловый менеджер tmpfs (v1.9, без realpath) |
| `/opt/web_entware/cgi-bin/ttyd_control.cgi` | GET/POST | JSON | Управление ttyd (исправлен парсинг POST) |
| `/opt/web_entware/cgi-bin/update.cgi` | GET | HTML | Обновление списков пакетов |
| `/opt/web_entware/cgi-bin/upgradable.cgi` | GET | JSON | Список пакетов с доступными обновлениями (v0.13, исправленный парсинг) |
| `/opt/web_entware/cgi-bin/upgrade.cgi` | POST | HTML | Обновление пакета или всех пакетов (v0.03) |
| `/opt/web_entware/cgi-bin/wifi_temp.cgi` | GET | JSON | Температура WiFi |
| `/opt/web_entware/cgi-bin/temp_history.cgi` | GET/POST | JSON | История температуры CPU (7 дней) |
| `/opt/web_entware/cgi-bin/wifi_temp_history.cgi` | GET/POST | JSON | История температуры WiFi (7 дней) |

#### 2.5.2. Подкаталог `cgi-bin/network/` — мониторинг сети

| Файл | Метод | Формат | Назначение |
|------|-------|--------|------------|
| `/opt/web_entware/cgi-bin/network/status.cgi` | GET | JSON | Статус демона сети (v1.7, uptime, last_check из state file) |
| `/opt/web_entware/cgi-bin/network/action.cgi` | POST | JSON | Управление демоном (start/stop/restart) |
| `/opt/web_entware/cgi-bin/network/config.cgi` | GET/POST | JSON | Чтение/сохранение конфига демона |
| `/opt/web_entware/cgi-bin/network/events.cgi` | GET | JSON | События демона (limit=N) |
| `/opt/web_entware/cgi-bin/network/interfaces.cgi` | GET | JSON | Список сетевых интерфейсов |
| `/opt/web_entware/cgi-bin/network/routes.cgi` | GET | JSON | Таблица маршрутизации |
| `/opt/web_entware/cgi-bin/network/arp.cgi` | GET | JSON | ARP-таблица |

#### 2.5.3. Подкаталог `cgi-bin/monitor/` — модуль защиты (CGI)

| Файл | Назначение |
|------|------------|
| `/opt/web_entware/cgi-bin/monitor/monitor_action.cgi` | Запуск/остановка/перезапуск демона, убийство процесса, очистка лога (исправлено логирование) |
| `/opt/web_entware/cgi-bin/monitor/monitor_config.cgi` | Чтение/запись конфигурации защиты (добавлен max_processes) |
| `/opt/web_entware/cgi-bin/monitor/monitor_log.cgi` | Просмотр лога демона защиты |
| `/opt/web_entware/cgi-bin/monitor/monitor_status.cgi` | Статус демона и топ-5 процессов по CPU |

#### 2.5.3. Подкаталог `cgi-bin/logger/` — модуль логирования (CGI)

| Файл | Назначение |
|------|------------|
| `/opt/web_entware/cgi-bin/logger/clear.cgi` | Очистка старых логов (>30 дней) |
| `/opt/web_entware/cgi-bin/logger/config.cgi` | Чтение/запись конфигурации логирования (v1.2, системный лог) |
| `/opt/web_entware/cgi-bin/logger/find_by_name.cgi` | Поиск файлов в `/tmp` по имени (JSON) |
| `/opt/web_entware/cgi-bin/logger/rotate.cgi` | Ручная ротация логов |
| `/opt/web_entware/cgi-bin/logger/system_log.cgi` | Просмотр системного лога событий (v1.7) |
| `/opt/web_entware/cgi-bin/logger/system_logs.cgi` | Просмотр выбранного системного лога (последние 500 строк, поиск) |
| `/opt/web_entware/cgi-bin/logger/view.cgi` | Просмотр логов действий менеджера с фильтрацией (v1.4, index() для поиска) |

### 2.6. Подкаталог `doc/` — проектная документация

| Файл | Назначение |
|------|------------|
| `/opt/web_entware/doc/README.md` | Оглавление и быстрый старт |
| `/opt/web_entware/doc/INSTALL.md` | Установка, обновление, удаление |
| `/opt/web_entware/doc/ARCHITECTURE.md` | Архитектура и модули |
| `/opt/web_entware/doc/API.md` | Описание CGI API |
| `/opt/web_entware/doc/OPERATIONS.md` | Эксплуатация и runbook |
| `/opt/web_entware/doc/TROUBLESHOOTING.md` | Диагностика и решения |
| `/opt/web_entware/doc/SECURITY.md` | Безопасность и ограничения |
| `/opt/web_entware/doc/CHANGELOG.md` | История изменений |

---

## 3. Пути и данные (внешние)

- **Логи действий менеджера**:
  - Временное хранилище: `/tmp/entware/logs/YYYY-MM-DD.log`
  - Постоянное хранилище: `/opt/var/log/entware/YYYY-MM-DD.log`
  - Формат записи: `[дата время] [уровень] [IP] [PID] [скрипт] сообщение`
  - Скрипт ротации: `/opt/web_entware/logger/scripts/rotate.sh` (запуск по cron в 12:00)

- **Лог демона защиты**: `/opt/temp/logs/monitor.log` (ротация при превышении 1 МБ)
- **Файл PID демона защиты**: `/tmp/watchdog.pid`
- **Временные счётчики демона**: `/tmp/monitor_counters/` и `/tmp/monitor_ignore_counters/`

- **Резервные копии**: `/opt/temp/backup/EntwareManager_backup_YYYYMMDD_HHMMSS/`

- **Конфигурация веб-сервера**: `/opt/etc/lighttpd/lighttpd.conf` (алиасы `/entware-manager/` и `/entware-cgi/`)

- **Службы Entware**: `/opt/etc/init.d/` (скрипты с префиксами S/K)

---

## 4. Основные модули (краткое описание)

### 4.1. Статистика (`stats.cgi`)
- Показывает модель, версию ядра, память (RAM), пакеты Entware, использование `/opt`, таблицы tmpfs и блочных устройств.
- Кликабельные точки монтирования tmpfs открывают файловый менеджер.
- Сортировка таблиц кликом по заголовку (числовая для размера и загрузки).

### 4.2. Пакеты
- **Установленные** (`packages.cgi` + `entware.js`): таблица с поиском, удаление через модальное окно, клик по строке показывает информацию (API через `api.cgi`).
- **Доступные** (`available.cgi` + `entware.js`): список с кэшированием в localStorage на 1 час.
- **Обновления** (`upgradable.cgi` + `entware.js`): список пакетов с новыми версиями, возможность обновить каждый пакет или все сразу (кнопка "Обновить все пакеты").
- **Обновление списков пакетов**: кнопка "Обновить списки пакетов" вызывает `update.cgi`.

### 4.3. Процессы и терминал
- **Процессы** – iframe с ttyd (порт 8089, htop).
- **Терминал** – iframe с ttyd (порт 9089, bash). Пароль настраивается во вкладке «Настройки».

### 4.4. Службы и Cron
- **Службы** (`services.cgi` + `service_action.cgi`): JSON-список скриптов в `/opt/etc/init.d/` с префиксами S/K. Кнопки управления (start/stop/restart/enable/disable) вызывают `service_action.cgi`.
- **Crontab**: два редактора (системный через `crontab -l`, и файл `/opt/etc/crontab`). Сохранение через `crontab_update.cgi` (оригинальная логика, без зависимости от common.sh).

### 4.5. Настройки
- **Управление ttyd**: статус, запуск/остановка/перезапуск экземпляров htop (порт 8089) и терминала (порт 9089). Пароль для терминала передаётся в `ttyd_control.cgi`.
- **Управление ссылками**: редактирование списка ссылок на главной странице (сохраняется в `links.json`). Иконки выбираются из списка (идентификаторы SVG). Загрузка и сохранение через `links_load.cgi` и `links_save.cgi`.

### 4.6. Защита (мониторинг процессов)
- **Демон** (`watchdog.sh`) – запускается через веб-интерфейс. Проверяет каждые `interval` секунд потребление CPU процессами (через `ps -e -o pid,pcpu,comm`). Если процесс превышает порог `threshold_cpu` в течение `threshold_time` секунд, он убивается (игнорируемые – не убиваются). Количество сканируемых процессов ограничено параметром `max_processes` (по умолчанию 200).
- **Конфигурация** хранится в `monitor_config.json`.
- **Лог** – в `/opt/temp/logs/monitor.log` с уровнями INFO, WARN, ERROR (включает PID процесса-инициатора).
- **Интерфейс** (`monitor.js`) – отображает статус демона, топ-5 процессов (PID, %CPU, время жизни, команду), настройки защиты, лог.

### 4.7. Файловый менеджер tmpfs (`tmpfs.cgi`)
- Доступен по клику на точки монтирования tmpfs из вкладки «Статистика».
- Позволяет навигировать по каталогам, просматривать файлы, удалять файлы и пустые папки.
- Сортировка по размеру (клик по заголовку).
- Удаление работает только в `/tmp`, `/dev/shm` и их подкаталогах.

### 4.8. Логирование
- **Действия менеджера**: записываются через `log_action` (из `logger/lib/logging.sh`). Все CGI-скрипты вызывают её после успешного выполнения. Логируются: установка/удаление/обновление пакетов, управление службами, сохранение crontab, ссылок, запуск/остановка ttyd, удаление файлов, действия с демоном защиты.
- **Системные логи**: можно добавлять источники (файлы из `/tmp`, содержащие `log` в имени) через кнопку «Поиск по имени файла». Найденные файлы добавляются в выпадающий список и сохраняются в `localStorage` браузера. Просмотр – через `system_logs.cgi` (последние 500 строк, поиск по тексту).
- **Ротация**: ежедневно в 12:00 (cron) копирует вчерашний лог из `/tmp/entware/logs/` в `/opt/var/log/entware/` и удаляет исходный. Старые логи (старше 30 дней) автоматически удаляются из постоянного хранилища.
- **Интерфейс** (`loadLogsTab` в `entware.js`): два режима (Действия менеджера / Системные логи), фильтры, кнопки управления.

### 4.9. Меню
- Динамическое, загружается из `menu/menu.json`.
- Активный пункт синхронизируется через `Menu.setActiveTab()`.

### 4.10. Справка (`help.cgi`)
- HTML-страница с подробным описанием всех вкладок, команд и настроек.
- Версия подставляется из `version.json`.

### 4.11. История температур
- **CPU температура**: данные сохраняются в `/tmp/temp_history/cpu.YYYY-MM-DD`, хранение 7 дней, точки каждые 30 секунд.
- **WiFi температура**: данные сохраняются в `/tmp/temp_history/wifi.YYYY-MM-DD`, хранение 7 дней.
- **Графики**: клик на виджет температуры CPU или WiFi в сайдбаре открывает модальное окно с графиком (Canvas).
- **API**: `temp_history.cgi` и `wifi_temp_history.cgi` (GET history, POST save).

---

## 5. Зависимости (opkg)
- lighttpd (с mod_cgi, mod_alias, mod_rewrite)
- ttyd
- htop
- jq
- coreutils-base (для базовых утилит)
- bash (опционально, для терминала)
- procps-ng (для `pgrep`, `ps`)

---

## 6. Безопасность
- Все CGI-скрипты выполняются от пользователя `nobody` (lighttpd). Доступ к файловой системе ограничен.
- Проверка путей в `delete_file.cgi` и `tmpfs.cgi` – разрешены только `/tmp` и `/dev/shm`.
- Пароль терминала не сохраняется, передаётся при запуске ttyd.
- Логи не содержат конфиденциальных данных (пароли), но содержат PID процессов.

---

## 7. Кэширование и версионирование
- **CSS / JS / спрайт** – в URL добавлен параметр `?v=2`, чтобы при обновлениях клиенты загружали новые версии.
- **Логи действий** – кэшируются на клиенте только для доступных пакетов (1 час).
- **Динамические источники системных логов** – сохраняются в `localStorage` (ключ `entware_dynamic_sources`).

---

## 8. Инструкции по установке (кратко)
1. Установить Entware на внешний носитель.
2. Установить пакеты: `opkg update && opkg install lighttpd ttyd htop jq coreutils-base procps-ng`
3. Скопировать файлы проекта в `/opt/web_entware/`
4. Настроить lighttpd (алиасы) – пример конфигурации в `/opt/web_entware/lighttpd.conf.example`
5. Запустить lighttpd: `/opt/etc/init.d/S80lighttpd start`
6. Открыть браузер: `http://192.168.3.1:8087/entware-manager/`
7. Настроить cron для ротации логов: добавить строку в `/opt/etc/crontab`:
0 12 * * * /opt/web_entware/logger/scripts/rotate.sh >> /opt/tmp/log_rotate.log 2>&1

---

## 9. Изменения в рефакторинге (пункты 1 и 3)

- **Вынесен общий код** в `/opt/web_entware/lib/common.sh`. Все CGI-скрипты используют единые функции для парсинга параметров, санитайза, вывода JSON/HTML и логирования.
- **Улучшена обработка ошибок** – добавлен модуль уведомлений `modal.js` (модальные окна и тосты), все CGI возвращают осмысленные сообщения об ошибках, проверяют коды возврата команд.
- **Удалены ссылки `javascript:window.close()`** из CGI-ответов, чтобы избавиться от предупреждений в консоли браузера.
- **Логирование дополнено PID** – теперь каждая запись в логах содержит идентификатор процесса CGI или демона.
- **Исправлен `crontab_update.cgi`** – возвращена оригинальная логика прямого парсинга POST (без зависимости от `common.sh`) для надёжности.
- **Исправлен `ttyd_control.cgi`** – также возвращён прямой парсинг POST.
- **Обновлён `backup.sh`** до версии 2.2 – копирует все новые файлы (lib, modal.js и др.), проверяет их наличие и содержит актуальный changelog.
- **Модуль защиты** (`monitor_action.cgi`) исправлен для корректного логирования в `/opt/temp/logs/monitor.log`.

---

*Документ актуален на 2026-04-08. Версия интерфейса: 1.02*
