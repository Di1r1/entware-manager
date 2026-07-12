# CGI API

Актуально для версии **1.01** от **2026-04-07**.

Базовый URL (типовой):
- UI: `http://<router-ip>:8087/entware-manager/`
- CGI: `http://<router-ip>:8087/entware-cgi/`

Ниже приведены ключевые эндпоинты. Формат ответа зависит от скрипта: JSON или HTML-фрагмент для вставки в UI.

## 1. Пакеты и обновления

- `GET /entware-cgi/available.cgi`
  - Назначение: список доступных пакетов.
  - Ответ: JSON.
  - Пример: `curl -s http://localhost:8087/entware-cgi/available.cgi | jq .`

- `GET /entware-cgi/packages.cgi`
  - Назначение: список установленных пакетов.
  - Ответ: HTML.

- `GET /entware-cgi/upgradable.cgi`
  - Назначение: список пакетов с доступными обновлениями.
  - Ответ: JSON.
  - Пример: `curl -s http://localhost:8087/entware-cgi/upgradable.cgi | jq .`

- `GET /entware-cgi/api.cgi?action=info&package=<name>`
  - Назначение: подробная информация о пакете.
  - Параметры:
    - `action` — действие (`info`)
    - `package` — имя пакета
  - Ответ: JSON.
  - Пример: `curl -s 'http://localhost:8087/entware-cgi/api.cgi?action=info&package=curl' | jq .`

- `POST /entware-cgi/install.cgi`
  - Назначение: установка пакета.
  - Ответ: HTML (результат операции).

- `POST /entware-cgi/remove.cgi`
  - Назначение: удаление пакета.
  - Ответ: HTML.

- `POST /entware-cgi/upgrade.cgi`
  - Назначение: обновление выбранного пакета.
  - Ответ: HTML.

- `GET /entware-cgi/update.cgi`
  - Назначение: обновление индекса пакетов `opkg`.
  - Ответ: HTML.

## 2. Сервисы и cron

- `GET /entware-cgi/services.cgi`
  - Назначение: список сервисов из `/opt/etc/init.d/`.
  - Ответ: JSON.

- `POST /entware-cgi/service_action.cgi`
  - Назначение: действие над сервисом.
  - Параметры:
    - `name` — имя сервиса
    - `action` — `start|stop|restart|enable|disable`
  - Ответ: JSON.

- `GET /entware-cgi/crontab.cgi?type=system|opt`
  - Назначение: чтение системного/Entware crontab.
  - Ответ: JSON.

- `POST /entware-cgi/crontab_update.cgi`
  - Назначение: сохранение crontab.
  - Ответ: JSON.

## 3. Статистика, tmpfs, файлы

- `GET /entware-cgi/stats.cgi`
  - Назначение: системная статистика.
  - Ответ: HTML.

- `GET /entware-cgi/tmpfs.cgi?path=<path>`
  - Назначение: просмотр содержимого tmpfs-каталога.
  - Ограничение: только `/tmp`, `/dev/shm` и их подкаталоги.
  - Ответ: HTML.

- `POST /entware-cgi/delete_file.cgi`
  - Назначение: удаление файла/папки в разрешенных tmpfs-путях.
  - Ответ: JSON.

## 3.1. Сеть (network)

- `GET /entware-cgi/network_status.cgi`
  - Назначение: JSON для sidebar виджета (LAN, WiFi, WAN, порты).
  - Ответ: JSON.
  - Пример: `curl -s http://localhost:8087/entware-cgi/network_status.cgi | jq .`

- `GET /entware-cgi/network/status.cgi`
  - Назначение: статус демона мониторинга сети.
  - Ответ: JSON: `{"running":true,"pid":12345}`.

- `GET /entware-cgi/network/interfaces.cgi`
  - Назначение: список сетевых интерфейсов.
  - Ответ: JSON.

- `GET /entware-cgi/network/routes.cgi`
  - Назначение: таблица маршрутизации.
  - Ответ: JSON.

- `GET /entware-cgi/network/arp.cgi`
  - Назначение: ARP-таблица.
  - Ответ: JSON.

- `GET /entware-cgi/network/events.cgi?limit=<N>&level=<LEVEL>`
  - Назначение: события от watchdog демона.
  - Параметры: `limit` (по умолчанию 50), `level` (ERROR/WARN/INFO).
  - Ответ: JSON.

- `POST /entware-cgi/network/action.cgi?action=<action>`
  - Назначение: управление демоном.
  - Параметры: `action` — `start`, `stop`, `restart`.
  - Ответ: JSON.

- `GET|POST /entware-cgi/network/config.cgi`
  - Назначение: чтение/сохранение конфигурации демона.
  - Ответ: JSON.

## 4. Температура

- `GET /entware-cgi/temperature.cgi`
  - Назначение: текущая температура CPU.
  - Ответ: JSON.

- `GET /entware-cgi/wifi_temp.cgi`
  - Назначение: текущая температура WiFi.
  - Ответ: JSON.

- `GET /entware-cgi/temp_history.cgi?action=history`
  - Назначение: история температуры CPU.
  - Ответ: JSON.

- `POST /entware-cgi/temp_history.cgi?action=save&temp=<value>`
  - Назначение: сохранить точку CPU температуры.
  - Ответ: JSON.

- `GET /entware-cgi/wifi_temp_history.cgi?action=history`
  - Назначение: история температуры WiFi.
  - Ответ: JSON.

- `POST /entware-cgi/wifi_temp_history.cgi?action=save&temp0=<value>&temp1=<value>`
  - Назначение: сохранить точки WiFi температуры.
  - Ответ: JSON.

## 5. Настройки интерфейса

- `GET /entware-cgi/links_load.cgi`
  - Назначение: загрузка пользовательских ссылок.
  - Ответ: JSON.

- `POST /entware-cgi/links_save.cgi`
  - Назначение: сохранение пользовательских ссылок.
  - Ответ: JSON.

- `GET|POST /entware-cgi/ttyd_control.cgi`
  - Назначение: запуск/остановка/статус ttyd-инстансов (htop/terminal).
  - Ответ: JSON.

## 6. Модуль защиты (monitor)

- `GET|POST /entware-cgi/monitor/monitor_action.cgi`
  - Назначение: действия с демоном (`start`, `stop`, `restart`, очистка лога и т.д.).
  - Ответ: JSON.

- `GET|POST /entware-cgi/monitor/monitor_config.cgi`
  - Назначение: чтение/сохранение параметров защиты.
  - Ключевые параметры: `interval`, `threshold_cpu`, `threshold_time`, `max_processes`, `ignore_patterns`.
  - Ответ: JSON.

- `GET /entware-cgi/monitor/monitor_status.cgi`
  - Назначение: статус демона + top процессов по CPU.
  - Ответ: JSON.

- `GET /entware-cgi/monitor/monitor_log.cgi`
  - Назначение: чтение лога демона.
  - Ответ: текст/JSON (зависит от реализации UI-потребления).

## 7. Модуль логирования (logger)

- `GET|POST /entware-cgi/logger/config.cgi`
  - Назначение: чтение/изменение конфигурации логирования действий.
  - Ответ: JSON.

- `GET /entware-cgi/logger/view.cgi?date=<YYYY-MM-DD>&level=<LEVEL>&search=<text>`
  - Назначение: просмотр логов действий с фильтрами.
  - Ответ: JSON.

- `GET /entware-cgi/logger/system_logs.cgi?source=<name>&file=<path>&level=<LEVEL>&search=<text>`
  - Назначение: чтение выбранного системного лога (обычно последние 500 строк + фильтрация).
  - Ответ: JSON.

- `GET /entware-cgi/logger/find_by_name.cgi?q=<mask>`
  - Назначение: поиск лог-файлов в `/tmp` по имени.
  - Ответ: JSON.

- `POST /entware-cgi/logger/rotate.cgi`
  - Назначение: ручной запуск ротации логов.
  - Ответ: JSON.

- `POST /entware-cgi/logger/clear.cgi`
  - Назначение: удаление старых логов.
  - Ответ: JSON.

## 8. Примечания по совместимости

- На слабых устройствах избегается тяжелая обработка и неиспользуемые зависимости в CGI.
- Для shell URL decode и парсинга параметров используется `lib/common.sh`.
- Для некоторых endpoint-ов сохранен ручной парсинг POST для надежности (`crontab_update.cgi`, `ttyd_control.cgi`, части monitor/logger).
