# Модуль "Сеть" — документация

Актуально для версии `0.77` от `2026-04-05`.

## Содержание

1. [Назначение](#1-назначение)
2. [Структура файлов](#2-структура-файлов)
3. [Компоненты](#3-компоненты)
4. [API](#4-api)
5. [Конфигурация](#5-конфигурация)
6. [Эксплуатация](#6-эксплуатация)
7. [Troubleshooting](#7-troubleshooting)

---

## 1. Назначение

Модуль "Сеть" предоставляет:
- **Sidebar виджет** — быстрый просмотр статуса сети без перехода на вкладку
- **Отдельную вкладку** — детальная информация об интерфейсах, маршрутах и ARP
- **Демон мониторинга** — автоматическое отслеживание состояния сети и уведомления

### Что мониторит демон

| Событие | Условие | Критичность | Действие |
|---------|---------|-------------|----------|
| Интерфейс DOWN | `ip link show` — state DOWN | ERROR | Записать событие |
| Нет интернета | `ping -c 1 8.8.8.8` не проходит | WARN | Записать событие |
| Изменение IP | Адрес интерфейса изменился | INFO | Записать событие |
| Интерфейс UP | Восстановление после DOWN | INFO | Записать событие |

---

## 2. Структура файлов

```
/opt/web_entware/
├── cgi-bin/
│   ├── network_status.cgi          # JSON для sidebar виджета
│   └── network/                   # Изолированная папка вкладки
│       ├── config.cgi             # Конфигурация демона
│       ├── status.cgi             # Статус демона
│       ├── action.cgi             # Управление демоном
│       ├── interfaces.cgi         # Данные интерфейсов
│       ├── routes.cgi             # Таблица маршрутизации
│       ├── arp.cgi                # ARP таблица
│       └── events.cgi             # События из лога
├── network.js                     # JS модуль (NETWORK)
├── network_config.json            # Конфигурация модуля
├── network_watchdog.sh           # Демон мониторинга
└── version.json                   # Версия (обновляется)
```

---

## 3. Компоненты

### 3.1. Sidebar виджет

**Файл:** `index.html`

**HTML:**
```html
<div class="network-widget-sidebar" id="networkWidget" style="cursor:pointer;">
    <span id="netLanStatus">LAN: --</span><br>
    <span id="netWifiStatus">WiFi: --</span><br>
    <span id="netWanStatus">WAN: --</span>
</div>
```

**JavaScript (index.html):**
```javascript
// Обновление каждые 30 секунд
setInterval(async () => {
    const res = await fetch('/entware-cgi/network_status.cgi');
    const data = await res.json();
    document.getElementById('netLanStatus').textContent = 'LAN: ' + (data.lan || '--');
    document.getElementById('netWifiStatus').textContent = 'WiFi: ' + (data.wifi || '--');
    document.getElementById('netWanStatus').textContent = 'WAN: ' + (data.wan || '--');
}, 30000);
```

**Данные:** `/entware-cgi/network_status.cgi` → JSON

---

### 3.2. Вкладка "Сеть"

**Пункт меню (menu.json):**
```json
{ "tab": "network", "icon": "router", "text": "Сеть" }
```

**Роутинг (entware.js):**
```javascript
if (tabName === 'network') { loadNetworkTab(); Menu.setActiveTab(tabName); return; }

function loadNetworkTab() {
    if (typeof initNetworkTab === 'function') {
        initNetworkTab();
        return;
    }
    const script = document.createElement('script');
    script.src = '/entware-manager/network.js';
    script.onload = () => { if (typeof initNetworkTab === 'function') initNetworkTab(); };
    document.head.appendChild(script);
}
```

**Секции вкладки:**
| Секция | Описание | Данные |
|--------|---------|--------|
| Интерфейсы | Список интерфейсов и их состояние | `ip addr` |
| Маршруты | Таблица маршрутизации | `ip route` |
| ARP | ARP таблица | `ip neigh` |
| События | Последние события демона | лог JSON |

**Фильтр интерфейсов:**
В панели управления демоном есть переключатель (toggle) "Скрыть неизвестные". При включении:
- Скрываются интерфейсы со состоянием `UNKNOWN`
- Полезно на устройствах с большим количеством виртуальных интерфейсов (OpenWrt/Keenetic)

---

### 3.3. Демон network_watchdog.sh

**Расположение:** `/opt/etc/init.d/S98network_watchdog`

**Принцип работы:**
1. Запускается через init.d
2. Цикл с интервалом (по умолчанию 30 сек)
3. Проверяет условия мониторинга
4. При срабатывании — записывает событие в лог
5. PID хранится в `/tmp/network_watchdog.pid`

**Управление:**
```sh
/opt/etc/init.d/S98network_watchdog start
/opt/etc/init.d/S98network_watchdog stop
/opt/etc/init.d/S98network_watchdog restart
/opt/etc/init.d/S98network_watchdog status
```

---

## 4. API

### 4.1. network_status.cgi

**Метод:** GET

**Описание:** JSON для sidebar виджета на странице статистики

**Ответ:**
```json
{
  "interfaces": [{"iface": "eth0", "ip": "192.168.1.1"}],
  "lan": "192.168.1.1",
  "wifi": "подключено",
  "wifi_info": [
    {"name": "LAN", "2g": "ra0", "5g": "rai0", "rx": "15 MB", "tx": "8 MB"}
  ],
  "wan": "ppp0 (up)",
  "ports": [
    {"port": "0", "iface": "ppp0"},
    {"port": "1", "iface": "eth2.1 - 1000Mbps"}
  ]
}
```

**Пример использования:**
```javascript
fetch('/entware-cgi/network_status.cgi')
  .then(r => r.json())
  .then(d => console.log(d.lan, d.wifi));
```

**Пример curl:**
```sh
curl -s http://localhost:8087/entware-cgi/network_status.cgi | jq .
```

---

### 4.2. network/config.cgi

**Методы:** GET, POST

**Описание:** Конфигурация демона

**GET — получить конфиг:**
```
/entware-cgi/network/config.cgi
```
```json
{
  "enabled": true,
  "interval": 30,
  "watch_interfaces": ["eth0", "wlan0"],
  "watch_internet": true,
  "ping_host": "8.8.8.8",
  "notify_on": ["interface_down", "no_internet", "ip_changed"]
}
```

**POST — сохранить конфиг:**
```
POST /entware-cgi/network/config.cgi
Content-Type: application/json

{"enabled": true, "interval": 30}
```

**Ответ:**
```json
{"status": "ok"}
```

**Пример curl:**
```sh
# Получить конфиг
curl -s http://localhost:8087/entware-cgi/network/config.cgi | jq .

# Сохранить конфиг
curl -X POST -H 'Content-Type: application/json' \
  -d '{"enabled":true,"interval":60}' \
  http://localhost:8087/entware-cgi/network/config.cgi
```

---

### 4.3. network/status.cgi

**Метод:** GET

**Описание:** Статус демона

**Ответ:**
```json
{
  "running": true,
  "pid": 12345,
  "uptime": 3600,
  "last_check": "2026-04-02 10:30:00"
}
```

**Пример curl:**
```sh
curl -s http://localhost:8087/entware-cgi/network/status.cgi | jq .
```

---

### 4.4. network/action.cgi

**Метод:** POST

**Описание:** Управление демоном

**Параметры:**
| Параметр | Описание |
|----------|----------|
| `action` | `start`, `stop`, `restart` |

**Пример:**
```
POST /entware-cgi/network/action.cgi
Content-Type: application/x-www-form-urlencoded

action=start
```

**Ответ:**
```json
{"status": "ok", "message": "Демон запущен"}
```

**Пример curl:**
```sh
curl -X POST 'http://localhost:8087/entware-cgi/network/action.cgi?action=start'
curl -X POST 'http://localhost:8087/entware-cgi/network/action.cgi?action=stop'
curl -X POST 'http://localhost:8087/entware-cgi/network/action.cgi?action=restart'
```

---

### 4.5. network/interfaces.cgi

**Метод:** GET

**Описание:** Данные интерфейсов

**Ответ:**
```json
{
  "interfaces": [
    {
      "name": "eth0",
      "state": "UP",
      "mtu": 1500,
      "ip": "192.168.1.1",
      "mac": "AA:BB:CC:DD:EE:FF",
      "speed": "1000Mbps",
      "type": "ethernet"
    }
  ]
}
```

**Пример curl:**
```sh
curl -s http://localhost:8087/entware-cgi/network/interfaces.cgi | jq .
```

---

### 4.6. network/routes.cgi

**Метод:** GET

**Описание:** Таблица маршрутизации

**Ответ:**
```json
{
  "routes": [
    {
      "destination": "default",
      "gateway": "192.168.1.1",
      "interface": "eth0",
      "metric": 10
    }
  ]
}
```

**Пример curl:**
```sh
curl -s http://localhost:8087/entware-cgi/network/routes.cgi | jq .
```

---

### 4.7. network/arp.cgi

**Метод:** GET

**Описание:** ARP таблица

**Ответ:**
```json
{
  "entries": [
    {
      "ip": "192.168.1.100",
      "mac": "AA:BB:CC:DD:EE:FF",
      "interface": "eth0",
      "state": "REACHABLE"
    }
  ]
}
```

**Пример curl:**
```sh
curl -s http://localhost:8087/entware-cgi/network/arp.cgi | jq .
```

---

### 4.8. network/events.cgi

**Метод:** GET

**Описание:** Последние события из лога

**Параметры:**
| Параметр | Описание | По умолчанию |
|----------|----------|--------------|
| `limit` | Количество событий | 50 |
| `level` | Фильтр по уровню | все |

**Пример:**
```
/entware-cgi/network/events.cgi?limit=10&level=WARN
```

**Ответ:**
```json
{
  "events": [
    {
      "timestamp": "2026-04-02 10:30:00",
      "level": "ERROR",
      "type": "interface_down",
      "message": "Интерфейс eth0 перешёл в состояние DOWN",
      "details": {}
    }
  ]
}
```

**Пример curl:**
```sh
# Последние 10 событий
curl -s 'http://localhost:8087/entware-cgi/network/events.cgi?limit=10' | jq .

# Только warnings
curl -s 'http://localhost:8087/entware-cgi/network/events.cgi?level=WARN&limit=20' | jq .
```

---

## 5. Конфигурация

### network_config.json

```json
{
  "version": "1.0",
  "sidebar": {
    "update_interval": 30,
    "show_lan": true,
    "show_wifi": true,
    "show_wan": true
  },
  "watchdog": {
    "enabled": true,
    "interval": 30,
    "watch_interfaces": ["eth0", "wlan0"],
    "watch_internet": true,
    "ping_host": "8.8.8.8",
    "ping_timeout": 5,
    "notify_on": ["interface_down", "no_internet", "ip_changed"]
  },
  "paths": {
    "pid_file": "/tmp/network_watchdog.pid",
    "log_file": "/tmp/entware/logs/network_events.log",
    "state_file": "/tmp/network_watchdog_state.json"
  }
}
```

### Файлы состояния

| Файл | Описание |
|------|---------|
| `/tmp/network_watchdog.pid` | PID процесса демона |
| `/tmp/entware/logs/network_events.log` | Лог событий (текстовый, ротация при >1MB) |
| `/tmp/network_watchdog_state.json` | Последнее состояние интерфейсов |

---

## 6. Эксплуатация

### 6.1. Управление демоном

**Через UI (вкладка "Сеть"):**
- Кнопки Start/Stop/Restart в панели управления

**Через CLI на роутере:**
```sh
# Запуск
/opt/etc/init.d/S98network_watchdog start

# Остановка
/opt/etc/init.d/S98network_watchdog stop

# Проверка статуса
/opt/etc/init.d/S98network_watchdog status

# Просмотр лога
cat /opt/var/log/entware/network_events.json | /opt/bin/jq .
```

### 6.2. Файлы

| Путь | Описание |
|------|---------|
| `/opt/web_entware/network_watchdog.sh` | Скрипт демона |
| `/tmp/network_watchdog.pid` | PID демона |
| `/tmp/entware/logs/network_events.log` | Лог событий (ротация при >1MB) |
| `/tmp/network_watchdog_state.json` | Состояние |

### 6.3. Рекомендуемые параметры

| Параметр | Значение | Описание |
|----------|---------|---------|
| `interval` | 30-60 сек | Интервал проверки |
| `ping_timeout` | 3-5 сек | Таймаут пинга |
| `watch_interfaces` | ["eth0"] | Интерфейсы для мониторинга |

---

## 7. Troubleshooting

### Демон не запускается

```sh
# Проверить зависимости
which ip
which ping
/opt/bin/jq --version

# Запустить вручную с выводом
sh -x /opt/web_entware/network_watchdog.sh
```

### Sidebar виджет не обновляется

1. Проверить `/entware-cgi/network_status.cgi` в браузере
2. Проверить консоль на ошибки JavaScript
3. Проверить `network_status.cgi` на роутере:
   ```sh
   /opt/web_entware/cgi-bin/network_status.cgi
   ```

### Нет данных в секциях

1. Проверить права на `/opt/bin/ip`, `/opt/bin/jq`
2. Проверить вывод команд:
   ```sh
   /opt/bin/ip addr
   /opt/bin/ip route
   /opt/bin/ip neigh
   ```

### Ошибки в логе

```sh
# Просмотр последних событий
tail -20 /opt/var/log/entware/network_events.json | /opt/bin/jq .

# Проверка формата лога
/opt/bin/jq . /opt/var/log/entware/network_events.json
```

---

## changelog

| Версия | Дата | Изменения |
|--------|------|-----------|
| 1.6 | 2026-04-05 | Изменён формат лога на текстовый, путь /tmp/entware/logs/, ротация при >1MB, добавлено логирование в основной лог |
| 1.5 | 2026-04-04 | Обновлена документация help.cgi, добавлены примеры curl |
| 1.0 | 2026-04-02 | Начальная версия |
