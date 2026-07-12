# Эксплуатация (Runbook)

## 1. Повседневные операции

- Проверка доступности UI: `http://<router-ip>:8087/entware-manager/`
- Проверка версии: `cat /opt/web_entware/version.json`
- Контроль lighttpd:
  - старт: `/opt/etc/init.d/S80lighttpd start`
  - рестарт: `/opt/etc/init.d/S80lighttpd restart`
  - статус: `pgrep -f lighttpd`

## 2. Логирование

### 2.1. Файлы

- Текущие логи действий: `/tmp/entware/logs/YYYY-MM-DD.log`
- Архив логов: `/opt/var/log/entware/YYYY-MM-DD.log`
- Лог монитора процессов: `/opt/temp/logs/monitor.log`

### 2.2. Ротация

Рекомендуемое cron-задание:

```cron
0 12 * * * /opt/web_entware/logger/scripts/rotate.sh >> /opt/tmp/log_rotate.log 2>&1
```

### 2.3. Очистка

- Ручной запуск ротации: через `logger/rotate.cgi` или скрипт `rotate.sh`.
- Очистка старых логов: `logger/clear.cgi`.

## 3. Мониторинг процессов (watchdog)

### 3.1. Управление

- Через UI вкладки «Защита» (предпочтительно).
- Через CGI monitor-модуля:
  - `monitor_action.cgi`
  - `monitor_config.cgi`
  - `monitor_status.cgi`

### 3.2. Состояние

- PID демона: `/tmp/watchdog.pid`
- Временные счетчики: `/tmp/monitor_counters/`, `/tmp/monitor_ignore_counters/`
- Лог: `/opt/temp/logs/monitor.log`

### 3.3. Рекомендованные параметры

- `interval`: 2-5 секунд
- `threshold_cpu`: 80-95
- `threshold_time`: 20-60 секунд
- `max_processes`: 100-300 для слабых роутеров

## 4. Температурные графики

- История CPU: `/tmp/temp_history/cpu.YYYY-MM-DD`
- История WiFi: `/tmp/temp_history/wifi.YYYY-MM-DD`
- Хранение: 7 дней
- Частота записи: обычно раз в 30 секунд

Примечание: очистка старых файлов истории выполняется с оптимизацией (не на каждый запрос, а периодически по маркеру дня).

## 5. Резервное копирование и восстановление

## 5.1. Backup

```sh
/opt/web_entware/backup.sh
```

Результат:
- каталог в `/opt/temp/backup/EntwareManager_backup_YYYYMMDD_HHMMSS/`
- копия `/opt/web_entware`
- дамп списка пакетов
- инструкции по восстановлению
- changelog

## 5.2. Restore (типовой)

```sh
rm -rf /opt/web_entware
cp -a /opt/temp/backup/<backup_dir>/web_entware /opt/
chmod 755 /opt/web_entware/cgi-bin/*.cgi
[ -d /opt/web_entware/cgi-bin/monitor ] && chmod 755 /opt/web_entware/cgi-bin/monitor/*.cgi
[ -d /opt/web_entware/cgi-bin/logger ] && chmod 755 /opt/web_entware/cgi-bin/logger/*.cgi
/opt/etc/init.d/S80lighttpd restart
```

После восстановления проверьте UI и ключевые CGI.

## 6. Плановые проверки (рекомендуемый регламент)

- Ежедневно:
  - доступность `lighttpd`
  - актуальность логов и отсутствие переполнения `/tmp`
- Еженедельно:
  - контроль ротации логов
  - проверка корректности `watchdog` и его PID
  - smoke-тест основных вкладок UI
- Перед обновлением:
  - обязательный `backup.sh`
  - фиксация текущей версии из `version.json`
