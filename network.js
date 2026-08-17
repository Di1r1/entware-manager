// Entware Manager - модуль Сеть
// Версия: 1.5
// Дата: 2026-04-04

const NETWORK = {
    intervalId: null,
    currentTab: 'interfaces',

    async init() {
        this.renderHTML();
        await this.loadData();
        this.attachEvents();
        this.enableTableSorting();
        this.loadConfig();
    },

    renderHTML() {
        const content = document.getElementById('content');
        content.innerHTML = `
            <h2 style="display: flex; align-items: center; gap: 8px;">
                <span class="stat-icon" style="width: 28px; height: 28px;">
                    <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=2#icon-router"/></svg>
                </span>
                Сеть
            </h2>
            <div id="network-status-panel" style="background: var(--command-block-bg); padding: 1rem; border-radius: 12px; margin-bottom: 1rem;">
                <div style="display: flex; gap: 1rem; flex-wrap: wrap; align-items: center;">
                    <span><strong>Демон:</strong> <span id="daemon-status" class="stat-value-normal">загрузка...</span></span>
                    <button id="network-start" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-play"/></svg> Запустить</button>
                    <button id="network-stop" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-stop"/></svg> Остановить</button>
                    <button id="network-restart" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Перезапустить</button>
                    <label class="service-watch-toggle" title="Запускать демон при загрузке роутера">
                        <input type="checkbox" id="network-autostart" style="display: none;">
                        <span class="toggle-slider"></span>
                        <span>Автозапуск при загрузке</span>
                    </label>
                </div>
            </div>
            <div id="network-tabs">
                <button class="tab-button active" data-tab="interfaces" id="tab-btn-interfaces">Интерфейсы</button>
                <button class="tab-button" data-tab="routes" id="tab-btn-routes">Маршруты</button>
                <button class="tab-button" data-tab="arp" id="tab-btn-arp">ARP</button>
                <button class="tab-button" data-tab="events" id="tab-btn-events">События</button>
                <label class="tab-toggle" title="Скрыть неизвестные интерфейсы">
                    <input type="checkbox" id="hide-unknown-ifaces" style="display: none;">
                    <span class="toggle-slider"></span>
                </label>
            </div>
            <div id="network-content">
                <div id="tab-interfaces" class="tab-content active">
                    <div class="packages-table-wrapper">
                        <table class="packages-table">
                            <thead>
                                <tr><th>Интерфейс</th><th>Состояние</th><th>IP адрес</th><th>MAC</th><th>Тип</th><th>Скорость</th></tr>
                            </thead>
                            <tbody id="interfaces-tbody">
                                <tr><td colspan="6">Загрузка...</td></tr>
                            </tbody>
                        </table>
                    </div>
                </div>
                <div id="tab-routes" class="tab-content">
                    <div class="packages-table-wrapper">
                        <table class="packages-table">
                            <thead>
                                <tr><th>Назначение</th><th>Шлюз</th><th>Интерфейс</th><th>Метрика</th></tr>
                            </thead>
                            <tbody id="routes-tbody">
                                <tr><td colspan="4">Загрузка...</td></tr>
                            </tbody>
                        </table>
                    </div>
                </div>
                <div id="tab-arp" class="tab-content">
                    <div class="packages-table-wrapper">
                        <table class="packages-table">
                            <thead>
                                <tr><th>IP адрес</th><th>Имя</th><th>MAC адрес</th><th>Интерфейс</th><th>Состояние</th></tr>
                            </thead>
                            <tbody id="arp-tbody">
                                <tr><td colspan="5">Загрузка...</td></tr>
                            </tbody>
                        </table>
                    </div>
                </div>
                <div id="tab-events" class="tab-content">
                    <div id="events-list" style="background: var(--pre-bg); padding: 1rem; border-radius: 8px; max-height: 400px; overflow-y: auto;">
                        <p>Загрузка событий...</p>
                    </div>
                    <button id="refresh-events" class="packages-delete-btn" style="margin-top: 1rem;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Обновить</button>
                </div>
            </div>
        `;
    },

    async loadData() {
        await this.loadStatus();
        await this.loadInterfaces();
        await this.loadRoutes();
        await this.loadArp();
        await this.loadEvents();
    },

    async loadStatus() {
        const statusSpan = document.getElementById('daemon-status');
        if (!statusSpan) return;
        
        try {
            const data = await apiGet('/network/status.cgi');
            
            if (data.running) {
                statusSpan.textContent = `Работает (PID: ${data.pid || '?'})`;
                statusSpan.className = 'stat-value-normal';
            } else {
                statusSpan.textContent = 'Остановлен';
                statusSpan.className = 'stat-value-warning';
            }
        } catch (e) {
            statusSpan.textContent = 'Ошибка';
            statusSpan.className = 'stat-value-critical';
        }
    },

    async loadInterfaces() {
        const tbody = document.getElementById('interfaces-tbody');
        if (!tbody) return;
        
        const hideUnknown = document.getElementById('hide-unknown-ifaces')?.checked;
        
        try {
            const data = await apiGet('/network/interfaces.cgi');
            
            if (data.interfaces && data.interfaces.length > 0) {
                let ifaces = data.interfaces;
                if (hideUnknown) {
                    ifaces = ifaces.filter(iface => iface.state !== 'UNKNOWN');
                }
                if (ifaces.length > 0) {
                    tbody.innerHTML = ifaces.map(iface => `
                        <tr>
                            <td><strong>${escapeHtml(iface.name)}</strong></td>
                            <td><span class="${iface.state === 'UP' ? 'stat-value-normal' : 'stat-value-critical'}">${escapeHtml(iface.state)}</span></td>
                            <td>${escapeHtml(iface.ip)}</td>
                            <td><code>${escapeHtml(iface.mac || '-')}</code></td>
                            <td>${escapeHtml(iface.type || '-')}${iface.ssid ? ` (${escapeHtml(iface.ssid)})` : ''}</td>
                            <td>${escapeHtml(iface.speed || '-')}</td>
                        </tr>
                    `).join('');
                } else {
                    tbody.innerHTML = '<tr><td colspan="6">Нет активных интерфейсов</td></tr>';
                }
            } else {
                tbody.innerHTML = '<tr><td colspan="6">Нет данных</td></tr>';
            }
        } catch (e) {
            tbody.innerHTML = '<tr><td colspan="6">Ошибка загрузки</td></tr>';
        }
    },

    async loadRoutes() {
        const tbody = document.getElementById('routes-tbody');
        if (!tbody) return;
        
        try {
            const data = await apiGet('/network/routes.cgi');
            
            if (data.routes && data.routes.length > 0) {
                tbody.innerHTML = data.routes.map(route => `
                    <tr>
                        <td><code>${escapeHtml(route.destination)}</code></td>
                        <td><code>${escapeHtml(route.gateway)}</code></td>
                        <td>${escapeHtml(route.interface)}</td>
                        <td>${route.metric ? escapeHtml(route.metric) : '-'}</td>
                    </tr>
                `).join('');
            } else {
                tbody.innerHTML = '<tr><td colspan="4">Нет данных</td></tr>';
            }
        } catch (e) {
            tbody.innerHTML = '<tr><td colspan="4">Ошибка загрузки</td></tr>';
        }
    },

    async loadArp() {
        const tbody = document.getElementById('arp-tbody');
        if (!tbody) return;
        
        try {
            const data = await apiGet('/network/arp.cgi');
            
            if (data.entries && data.entries.length > 0) {
                tbody.innerHTML = data.entries.map(entry => `
                    <tr>
                        <td><code>${escapeHtml(entry.ip)}</code></td>
                        <td>${escapeHtml(entry.name || '-')}</td>
                        <td><code>${escapeHtml(entry.mac)}</code></td>
                        <td>${escapeHtml(entry.interface)}</td>
                        <td>${escapeHtml(entry.state)}</td>
                    </tr>
                `).join('');
            } else {
                tbody.innerHTML = '<tr><td colspan="5">Таблица пуста</td></tr>';
            }
        } catch (e) {
            tbody.innerHTML = '<tr><td colspan="5">Ошибка загрузки</td></tr>';
        }
    },

    async loadEvents() {
        const container = document.getElementById('events-list');
        if (!container) return;
        
        try {
            const data = await apiGet('/network/events.cgi?limit=20');
            
            if (data.events && data.events.length > 0) {
                container.innerHTML = data.events.map(event => `
                    <div style="padding: 0.5rem; border-bottom: 1px solid var(--border-color);">
                        <span style="color: var(--text-muted);">${escapeHtml(event.timestamp)}</span>
                        <span style="padding: 0.1rem 0.3rem; border-radius: 3px; font-size: 0.75rem; margin-left: 0.5rem;
                            ${event.level === 'ERROR' ? 'background: #fee2e2; color: #b91c1c;' : ''}
                            ${event.level === 'WARN' ? 'background: #fef3c7; color: #92400e;' : ''}
                            ${event.level === 'INFO' ? 'background: #dcfce7; color: #166534;' : ''}
                        ">${escapeHtml(event.level)}</span>
                        <br>
                        <strong>${escapeHtml(event.service)}</strong>: ${escapeHtml(event.event)} ${escapeHtml(event.details)}
                    </div>
                `).join('');
            } else {
                container.innerHTML = '<p>Нет событий</p>';
            }
        } catch (e) {
            container.innerHTML = '<p>Ошибка загрузки событий</p>';
        }
    },

    async loadConfig() {
        try {
            const cfg = await apiGet('/network/config.cgi');
            this.config = cfg || {};
            const box = document.getElementById('network-autostart');
            if (box) box.checked = this.config.autostart === true;
        } catch(e) {
            console.error(e);
        }
    },

    async saveAutostart(checked) {
        if (!this.config) this.config = {};
        this.config.autostart = !!checked;
        try {
            const res = await apiPostJSON('/network/config.cgi', this.config);
            if (res && res.status === 'ok') {
                Toast.show(checked ? 'Автозапуск включён' : 'Автозапуск выключен');
            } else {
                Toast.show('Ошибка сохранения автозапуска', true);
            }
        } catch (err) {
            Toast.show('Ошибка соединения: ' + err.message, true);
        }
    },

    attachEvents() {
        document.getElementById('network-start')?.addEventListener('click', () => this.doAction('start'));
        document.getElementById('network-stop')?.addEventListener('click', () => this.doAction('stop'));
        document.getElementById('network-restart')?.addEventListener('click', () => this.doAction('restart'));
        document.getElementById('network-autostart')?.addEventListener('change', (e) => this.saveAutostart(e.target.checked));
        document.getElementById('refresh-events')?.addEventListener('click', () => this.loadEvents());
        
        document.getElementById('hide-unknown-ifaces')?.addEventListener('change', () => {
            this.loadInterfaces();
        });
        
        ['interfaces', 'routes', 'arp', 'events'].forEach(tab => {
            const btn = document.getElementById('tab-btn-' + tab);
            if (btn) {
                btn.onclick = (e) => {
                    e.preventDefault();
                    switchNetworkTab(btn, tab);
                };
            }
        });
    },

    enableTableSorting() {
        const tables = [
            { tbody: 'interfaces-tbody', types: ['string', 'string', 'ip', 'string', 'string', 'speed'] },
            { tbody: 'routes-tbody', types: ['ip', 'ip', 'string', 'number'] },
            { tbody: 'arp-tbody', types: ['ip', 'string', 'string', 'string', 'string'] },
        ];

        tables.forEach(cfg => {
            const tbody = document.getElementById(cfg.tbody);
            if (!tbody) return;
            const table = tbody.closest('table');
            if (!table || table.dataset.sortable) return;
            table.dataset.sortable = 'true';

            const headers = table.querySelectorAll('thead th');
            headers.forEach((th, idx) => {
                th.style.cursor = 'pointer';
                th.addEventListener('click', () => {
                    sortTable(table, idx, cfg.types[idx] || 'string');
                });
            });
        });
    },

    async doAction(action) {
        console.log('doAction called with:', action);
        const btn = document.getElementById(`network-${action}`);
        if (btn) btn.disabled = true;
        
        try {
            const data = await apiPost('/network/action.cgi', 'action=' + action);
            
            if (data.status === 'ok') {
                Toast.show(data.message);
                await this.loadStatus();
                await this.loadEvents();
            } else {
                Toast.show(data.message || 'Ошибка', true);
            }
        } catch (e) {
            console.error('Error in doAction:', e);
            Toast.show('Ошибка: ' + e.message, true);
        }
        
        if (btn) btn.disabled = false;
    },
};

function initNetworkTab() {
    NETWORK.init();
}

function switchNetworkTab(btn, tabName) {
    console.log('Tab clicked:', tabName);
    document.querySelectorAll('#network-tabs .tab-button').forEach(b => b.classList.remove('active'));
    document.querySelectorAll('#network-content .tab-content').forEach(c => c.classList.remove('active'));
    
    btn.classList.add('active');
    const contentEl = document.getElementById('tab-' + tabName);
    if (contentEl) {
        contentEl.classList.add('active');
        console.log('Activated tab:', 'tab-' + tabName);
    } else {
        console.error('Tab not found:', 'tab-' + tabName);
    }
    NETWORK.currentTab = tabName;
}
