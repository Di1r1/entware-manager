# ПРОМТ ДЛЯ СОЗДАНИЯ МОДУЛЯ "СЕТЬ"

## Контекст проекта

Entware Manager — веб-интерфейс для управления Entware-пакетами на роутере (OpenWrt/Keenetic).

**Корень проекта:** `/opt/web_entware/`
**Версия:** 0.65
**Дата:** 2026-04-02

### Особенности окружения

- CGI-скрипты: `sh` (Busybox)
- PATH в CGI: `/sbin:/usr/sbin:/bin:/usr/bin` (НЕ включает `/opt/bin`)
- Всегда использовать полные пути: `/opt/bin/jq`, `/opt/bin/sed`, `/opt/bin/cat`
- Все CGI подключают: `. /opt/web_entware/lib/common.sh`

### Существующие аналоги (образцы)

- `cgi-bin/monitor/` — изолированная структура с демоном
- `cgi-bin/logger/` — система логирования
- `monitor.js` — образец JS модуля
- `watchdog.sh` — образец демона
- `lib/common.sh` — общие функции

---

## ЗАДАЧА

Создать модуль "Сеть" с тремя компонентами:

### 1. Sidebar виджет (index.html)

**Назначение:** быстрый просмотр статуса сети без перехода на вкладку

**HTML для вставки в sidebar:**
```html
<div class="network-widget-sidebar" id="networkWidget" style="cursor:pointer;">
    <span id="netLanStatus">LAN: --</span><br>
    <span id="netWifiStatus">WiFi: --</span><br>
    <span id="netWanStatus">WAN: --</span>
</div>
```

**JavaScript:**
```javascript
// Добавить в существующий setInterval или создать новый
setInterval(async () => {
    try {
        const res = await fetch('/entware-cgi/network_status.cgi');
        const data = await res.json();
        document.getElementById('netLanStatus').textContent = 'LAN: ' + (data.lan || '--');
        document.getElementById('netWifiStatus').textContent = 'WiFi: ' + (data.wifi || '--');
        document.getElementById('netWanStatus').textContent = 'WAN: ' + (data.wan || '--');
    } catch (e) { console.error(e); }
}, 30000);

// Загрузить сразу
(async () => {
    try {
        const res = await fetch('/entware-cgi/network_status.cgi');
        const data = await res.json();
        document.getElementById('netLanStatus').textContent = 'LAN: ' + (data.lan || '--');
        document.getElementById('netWifiStatus').textContent = 'WiFi: ' + (data.wifi || '--');
        document.getElementById('netWanStatus').textContent = 'WAN: ' + (data.wan || '--');
    } catch (e) { console.error(e); }
})();
```

**Стили:** аналогично `tempWidget` и `wifiTempWidget` в sidebar

---

### 2. Отдельная вкладка "Сеть"

**Пункт меню (menu.json):**
```json
{ "tab": "network", "icon": "router", "text": "Сеть" }
```

**Роутинг (entware.js):**
```javascript
// Добавить в loadTab()
if (tabName === 'network') { loadNetworkTab(); Menu.setActiveTab(tabName); return; }

// Добавить функцию
function loadNetworkTab() {
    if (typeof initNetworkTab === 'function') {
        initNetworkTab();
        return;
    }
    const script = document.createElement('script');
    script.src = '/entware-manager/network.js?v=2';
    script.onload = () => {
        if (typeof initNetworkTab === 'function') initNetworkTab();
        else document.getElementById('content').innerHTML = '<p class="error">Ошибка загрузки модуля</p>';
    };
    script.onerror = () => document.getElementById('content').innerHTML = '<p class="error">Не удалось загрузить модуль</p>';
    document.head.appendChild(script);
}
```

**Секции вкладки:**
| Секция | Данные | Примечание |
|--------|--------|-----------|
| Интерфейсы | `ip addr` | Список интерфейсов и их состояние |
| Маршруты | `ip route` | Таблица маршрутизации |
| ARP | `ip neigh` | ARP таблица |
| События | лог демона | Последние события мониторинга |

**БЕЗ соединений (ss/netstat)** — не делать!

---

### 3. Автономный демон network_watchdog.sh

**Расположение:** `/opt/web_entware/network_watchdog.sh`
**Init.d:** `/opt/etc/init.d/S98network_watchdog` (симлинк)

**Принцип работы:**
```sh
#!/bin/sh
# Читает конфиг из /opt/web_entware/network_config.json
# Цикл с интервалом (по умолчанию 30 сек)
# Проверяет условия мониторинга
# При срабатывании — записывает событие в JSON лог
# PID хранится в /tmp/network_watchdog.pid
```

**Init.d команды:**
- `start` — запуск демона
- `stop` — остановка демона
- `restart` — перезапуск
- `status` — проверка статуса

**Что мониторит:**
| Событие | Условие | Критичность |
|---------|---------|-------------|
| Интерфейс DOWN | `ip link show eth0` state DOWN | ERROR |
| Нет интернета | `ping -c 1 -W 5 8.8.8.8` не проходит | WARN |
| Изменение IP | адрес интерфейса изменился | INFO |

**Логирование:**
```json
{
  "events": [
    {
      "timestamp": "2026-04-02 10:30:00",
      "level": "ERROR",
      "type": "interface_down",
      "message": "Интерфейс eth0 перешёл в состояние DOWN",
      "details": {"interface": "eth0"}
    }
  ]
}
```

**Файлы:**
- `/tmp/network_watchdog.pid` — PID процесса
- `/opt/var/log/entware/network_events.json` — лог событий
- `/tmp/network_watchdog_state.json` — состояние (для детектиции изменений)

---

## АРХИТЕКТУРА CGI

### Структура папки `cgi-bin/network/`

```
cgi-bin/network/
├── config.cgi      # GET конфиг, POST сохранить
├── status.cgi      # Статус демона (pgrep)
├── action.cgi      # start/stop/restart демон
├── interfaces.cgi  # ip addr (JSON)
├── routes.cgi      # ip route (JSON)
├── arp.cgi        # ip neigh (JSON)
└── events.cgi     # Последние события (JSON)
```

### API эндпоинты

| URL | Метод | Описание |
|-----|-------|---------|
| `/entware-cgi/network_status.cgi` | GET | JSON для sidebar |
| `/entware-cgi/network/config.cgi` | GET/POST | Конфиг демона |
| `/entware-cgi/network/status.cgi` | GET | Статус демона |
| `/entware-cgi/network/action.cgi` | POST | Управление демоном |
| `/entware-cgi/network/interfaces.cgi` | GET | Интерфейсы |
| `/entware-cgi/network/routes.cgi` | GET | Маршруты |
| `/entware-cgi/network/arp.cgi` | GET | ARP таблица |
| `/entware-cgi/network/events.cgi` | GET | События |

### JSON форматы

**network_status.cgi (sidebar):**
```json
{"lan": "192.168.1.1", "wifi": "подключено", "wan": "active", "speed": "1000Mbps"}
```

**interfaces.cgi:**
```json
{
  "interfaces": [
    {
      "name": "eth0",
      "state": "UP",
      "ip": "192.168.1.1",
      "mac": "AA:BB:CC:DD:EE:FF",
      "speed": "1000Mbps"
    }
  ]
}
```

**events.cgi:**
```json
{
  "events": [
    {
      "timestamp": "2026-04-02 10:30:00",
      "level": "ERROR",
      "type": "interface_down",
      "message": "Интерфейс eth0 перешёл в состояние DOWN"
    }
  ]
}
```

---

## ФАЙЛЫ ДЛЯ СОЗДАНИЯ

### Новые файлы (11)

1. `cgi-bin/network_status.cgi` — JSON для sidebar
2. `cgi-bin/network/config.cgi` — конфиг демона
3. `cgi-bin/network/status.cgi` — статус демона
4. `cgi-bin/network/action.cgi` — управление демоном
5. `cgi-bin/network/interfaces.cgi` — интерфейсы
6. `cgi-bin/network/routes.cgi` — маршруты
7. `cgi-bin/network/arp.cgi` — ARP таблица
8. `cgi-bin/network/events.cgi` — события
9. `network.js` — JS модуль
10. `network_config.json` — конфигурация
11. `network_watchdog.sh` — демон

### Изменяемые файлы (5)

1. `index.html` — +sidebar виджет
2. `entware.js` — +tab routing
3. `menu.json` — +menu item
4. `icons.svg` — +icon-network
5. `version.json` — обновить версию

---

## ТРЕБОВАНИЯ К КОДУ

### CGI скрипты

1. Всегда использовать shebang: `#!/bin/sh`
2. Всегда подключать: `. /opt/web_entware/lib/common.sh`
3. Использовать полные пути: `/opt/bin/jq`, `/opt/bin/cat`, `/opt/bin/sed`
4. JSON ответы начинать с `Content-type: application/json`
5. Корректный ранний `exit 0` после ответа
6. Обработка ошибок с fallback значениями

**Шаблон:**
```sh
#!/bin/sh
# Комментарий с версией
. /opt/web_entware/lib/common.sh

CONFIG_FILE="/opt/web_entware/network_config.json"
LOG_FILE="/opt/var/log/entware/network_events.json"

if [ "$REQUEST_METHOD" = "GET" ]; then
    echo "Content-type: application/json"
    echo ""
    # ... логика ...
    exit 0
fi

if [ "$REQUEST_METHOD" = "POST" ]; then
    # ... логика ...
    exit 0
fi

json_out '{"error":"Method not allowed"}'
```

### JavaScript

1. Namespace: `NETWORK` (заглавными)
2. Функция инициализации: `initNetworkTab()`
3. Стиль аналогичен `MONITOR` в `monitor.js`
4. Lazy load из отдельного файла

**Шаблон:**
```javascript
// network.js
const NETWORK = {
    intervalId: null,
    
    async init() {
        this.renderHTML();
        this.startUpdates();
        this.attachEvents();
    },
    
    renderHTML() {
        const content = document.getElementById('content');
        content.innerHTML = `...`;
    },
    
    startUpdates() { ... },
    attachEvents() { ... }
};

function initNetworkTab() {
    NETWORK.init();
}
```

### Демон

1. Поддержка команд init.d: start, stop, restart, status
2. Запись PID в файл
3. Цикл с интервалом
4. JSON логирование
5. Корректное завершение при stop

**Шаблон:**
```sh
#!/bin/sh
NAME="network_watchdog"
DAEMON="/opt/web_entware/network_watchdog.sh"
PIDFILE="/tmp/network_watchdog.pid"
LOGFILE="/opt/var/log/entware/network_events.json"

case "$1" in
    start)
        if [ -f "$PIDFILE" ] && kill -0 $(cat $PIDFILE) 2>/dev/null; then
            echo "Already running"
            exit 1
        fi
        nohup $DAEMON >> /tmp/network_watchdog.log 2>&1 &
        echo $! > $PIDFILE
        ;;
    stop)
        [ -f $PIDFILE ] && kill $(cat $PIDFILE) && rm $PIDFILE
        ;;
    status)
        [ -f $PIDFILE ] && kill -0 $(cat $PIDFILE) && echo "Running" || echo "Not running"
        ;;
esac
```

---

## ИКОНКА

Добавить в `icons.svg`:

```svg
<symbol id="icon-network" viewBox="0 0 16 16">
    <circle cx="8" cy="5" r="2.5" fill="none" stroke="currentColor"/>
    <circle cx="3" cy="13" r="2" fill="none" stroke="currentColor"/>
    <circle cx="13" cy="13" r="2" fill="none" stroke="currentColor"/>
    <line x1="8" y1="7.5" x2="8" y2="14" stroke="currentColor"/>
    <line x1="3" y1="11" x2="8" y2="7.5" stroke="currentColor"/>
    <line x1="13" y1="11" x2="8" y2="7.5" stroke="currentColor"/>
</symbol>
```

---

## ПОРЯДОК РАЗРАБОТКИ

1. Создать `cgi-bin/network_status.cgi` — проверить вручную
2. Создать `cgi-bin/network/*.cgi` (8 файлов) — проверить каждый
3. Создать `network_config.json`
4. Создать `network_watchdog.sh` — проверить init.d
5. Создать `network.js` — проверить lazy load
6. Изменить `index.html` — добавить виджет
7. Изменить `entware.js` — добавить роутинг
8. Изменить `menu.json` — добавить пункт
9. Изменить `icons.svg` — добавить иконку
10. Обновить `version.json`
11. Протестировать всё вместе

---

## ПРОВЕРКА

После создания проверить:

1. **Sidebar виджет:**
   - Открыть главную страницу
   - Виджет показывает данные
   - Обновляется каждые 30 сек

2. **Вкладка "Сеть":**
   - Пункт в меню
   - Клик открывает вкладку
   - Все секции загружаются

3. **Демон:**
   - Запуск через UI
   - Остановка через UI
   - События пишутся в лог
   - Init.d команды работают

4. **API:**
   - Все эндпоинты возвращают валидный JSON
   - Нет ошибок 500
