// Entware Manager - модуль защиты (мониторинг процессов)
// Версия: 0.12 (убран group, используется utils.js)
// Дата: 2026-04-01

const MONITOR = {
    intervalId: null,
    logIntervalId: null,

    async init() {
        this.renderHTML();
        await this.loadConfig();
        this.startStatusUpdates();
        this.attachEvents();
    },

    stopUpdates() {
        if (this.intervalId) {
            clearInterval(this.intervalId);
            this.intervalId = null;
        }
        if (this.logIntervalId) {
            clearInterval(this.logIntervalId);
            this.logIntervalId = null;
        }
    },

    renderHTML() {
        const content = document.getElementById('content');
        content.innerHTML = `
            <h2 style="display: flex; align-items: center; gap: 8px;">
                <span class="stat-icon" style="width: 28px; height: 28px;">
                    <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=3#icon-shield"/></svg>
                </span>
                Защита от зависших процессов
            </h2>
            <div id="monitor-status-panel" style="background: var(--command-block-bg); padding: 1rem; border-radius: 12px; margin-bottom: 1rem;">
                <div style="display: flex; gap: 1rem; flex-wrap: wrap; align-items: center;">
                    <span><strong>Статус демона:</strong> <span id="daemon-status" class="stat-value-normal">загрузка...</span></span>
                    <button id="monitor-start" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=3#icon-play"/></svg> Запустить</button>
                    <button id="monitor-stop" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=3#icon-stop"/></svg> Остановить</button>
                    <button id="monitor-restart" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=3#icon-refresh"/></svg> Перезапустить</button>
                    <label class="monitor-toggle" title="Запускать демон при загрузке роутера">
                        <input type="checkbox" id="monitor-autostart" style="display: none;">
                        <span class="toggle-slider"></span>
                        <span>Автозапуск при загрузке</span>
                    </label>
                </div>
            </div>
            <div id="monitor-processes">
                <h3>Топ процессов по нагрузке CPU</h3>
                <div class="packages-table-wrapper">
                    <table class="packages-table">
                        <thead>
                            <tr><th>PID</th><th>%CPU</th><th>CPU время</th><th>Команда</th><th>Действие</th></tr>
                        </thead>
                        <tbody id="processes-tbody">
                            <tr><td colspan="5">Загрузка...</td></tr>
                        </tbody>
                    </table>
                </div>
            </div>
                <div id="monitor-settings">
                <h3>Настройки защиты</h3>
                <form id="monitor-settings-form">
                    <label class="monitor-toggle" title="Включить защиту (глобально)">
                        <input type="checkbox" id="settings-enabled" style="display: none;">
                        <span class="toggle-slider"></span>
                        <span>Включить защиту (глобально)</span>
                    </label>
                    <div><label>Интервал сканирования (сек): <select id="settings-interval" class="settings-input">${this.generateOptionsList([10,30,60,120,300])}</select></label></div>
                    <hr>
                    <h4>Индивидуальный режим</h4>
                    <label class="monitor-toggle" title="Включить индивидуальный режим">
                        <input type="checkbox" id="settings-individual-enabled" style="display: none;">
                        <span class="toggle-slider"></span>
                        <span>Включить</span>
                    </label>
                    <div><label>Порог CPU (%): <select id="settings-individual-cpu" class="settings-input">${this.generateOptions(10, 100, 10)}</select></label></div>
                    <div><label>Время непрерывной нагрузки (мин): <select id="settings-individual-time" class="settings-input">${this.generateOptionsList([1,2,3,5,10,15,20,30])}</select></label></div>
                    <hr>
                    <div><label>Игнорируемые процессы (имена, через запятую): <input type="text" id="settings-ignore" class="settings-input"></label></div>
                    <label class="monitor-toggle" title="Исключать ps из мониторинга">
                        <input type="checkbox" id="settings-ignore-ps" style="display: none;">
                        <span class="toggle-slider"></span>
                        <span>Исключать ps из мониторинга (убирает ложные предупреждения)</span>
                    </label>
                    <hr>
                    <h4>Дополнительные настройки</h4>
                    <div><label>Максимум процессов для сканирования: <input type="number" id="settings-max-processes" min="10" max="1000" value="200" class="settings-input" style="width:80px;"> (10-1000)</label></div>
                    <button type="submit" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=3#icon-disk"/></svg> Сохранить настройки</button>
                </form>
            </div>
            <div id="monitor-log">
                <h3>Лог событий</h3>
                <pre id="log-content" style="background: var(--pre-bg); padding: 0.5rem; height: 200px; overflow-y: auto;">Загрузка...</pre>
                <button id="clear-log" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=3#icon-trash"/></svg> Очистить лог</button>
            </div>
        `;
    },

    generateOptions(start, end, step) {
        let html = '';
        for (let i = start; i <= end; i += step) {
            html += `<option value="${i}">${i}%</option>`;
        }
        return html;
    },

    generateOptionsList(values) {
        return values.map(v => `<option value="${v}">${v}</option>`).join('');
    },

    async updateStatus() {
        const statusSpan = document.getElementById('daemon-status');
        const tbody = document.getElementById('processes-tbody');
        if (!statusSpan || !tbody) return;

        try {
            const data = await apiGet('/monitor/status.cgi');
            if (data.daemon_status === 'running') {
                statusSpan.textContent = 'активен (PID ' + data.daemon_pid + ')';
                statusSpan.className = 'stat-value-normal';
            } else {
                statusSpan.textContent = 'остановлен';
                statusSpan.className = 'stat-value-critical';
            }

            if (data.processes && data.processes.length) {
                tbody.innerHTML = data.processes.map(p => `
                    <tr>
                        <td>${escapeHtml(p.pid)}</td>
                        <td>${escapeHtml(p.pcpu)}</td>
                        <td>${escapeHtml(p.time)}</td>
                        <td title="${escapeHtml(p.command)}">${escapeHtml(p.command).substring(0, 50)}</td>
                        <td><button class="packages-delete-btn" data-pid="${escapeHtml(p.pid)}">Убить</button></td>
                    </tr>
                `).join('');
                document.querySelectorAll('#processes-tbody button[data-pid]').forEach(btn => {
                    btn.onclick = () => this.killProcess(parseInt(btn.dataset.pid));
                });
            } else {
                tbody.innerHTML = '<tr><td colspan="5">Нет данных</td></tr>';
            }
        } catch(e) {
            console.error(e);
            tbody.innerHTML = '<tr><td colspan="5">Ошибка загрузки</td></tr>';
        }
    },

    async loadConfig() {
        try {
            const cfg = await apiGet('/monitor/config.cgi');
            document.getElementById('settings-enabled').checked = cfg.enabled;
            document.getElementById('settings-interval').value = cfg.interval;
            document.getElementById('settings-individual-enabled').checked = cfg.individual.enabled;
            document.getElementById('settings-individual-cpu').value = cfg.individual.threshold_cpu;
            document.getElementById('settings-individual-time').value = Math.floor(cfg.individual.threshold_time / 60);
            document.getElementById('settings-ignore').value = (cfg.ignore || []).join(', ');
            document.getElementById('settings-ignore-ps').checked = cfg.ignore_ps === true;
            document.getElementById('settings-max-processes').value = cfg.max_processes || 200;
            document.getElementById('monitor-autostart').checked = cfg.autostart === true;
        } catch(e) {
            console.error(e);
        }
    },

    async saveConfig(e) {
        e.preventDefault();
        const maxProcesses = parseInt(document.getElementById('settings-max-processes').value) || 200;
        const config = {
            enabled: document.getElementById('settings-enabled').checked,
            interval: parseInt(document.getElementById('settings-interval').value),
            individual: {
                enabled: document.getElementById('settings-individual-enabled').checked,
                threshold_cpu: parseInt(document.getElementById('settings-individual-cpu').value),
                threshold_time: parseInt(document.getElementById('settings-individual-time').value) * 60
            },
            ignore: document.getElementById('settings-ignore').value.split(',').map(s => s.trim()).filter(s => s),
            ignore_ps: document.getElementById('settings-ignore-ps').checked,
            max_processes: maxProcesses,
            autostart: document.getElementById('monitor-autostart').checked
        };
        try {
            var result = await apiPostJSON('/monitor/config.cgi', config);
            if (result.status === 'ok') {
                Toast.show('Настройки сохранены');
                await apiPost('/monitor/action.cgi', 'action=restart');
            } else {
                Toast.show('Ошибка: ' + result.message, true);
            }
        } catch (err) {
            Toast.show('Ошибка соединения: ' + err.message, true);
        }
    },

    async sendAction(action) {
        try {
            var res = await apiPost('/monitor/action.cgi', 'action=' + action);
            Toast.show(res.message);
            this.updateStatus();
        } catch (err) {
            Toast.show('Ошибка: ' + err.message, true);
        }
    },

    async killProcess(pid) {
        if (confirm('Убить процесс ' + pid + '?')) {
            try {
                var res = await apiPost('/monitor/action.cgi', 'action=kill&pid=' + pid);
                Toast.show(res.message);
                this.updateStatus();
            } catch (err) {
                Toast.show('Ошибка: ' + err.message, true);
            }
        }
    },

    async loadLog() {
        const logPre = document.getElementById('log-content');
        if (!logPre) return;

        try {
            const resp = await apiFetch('/monitor/log.cgi');
            const text = await resp.text();
            logPre.innerText = text;
        } catch(e) {
            logPre.innerText = 'Ошибка загрузки лога';
        }
    },

    startStatusUpdates() {
        if (this.intervalId) clearInterval(this.intervalId);
        if (this.logIntervalId) clearInterval(this.logIntervalId);
        this.updateStatus();
        this.intervalId = setInterval(() => this.updateStatus(), 5000);
        this.loadLog();
        this.logIntervalId = setInterval(() => this.loadLog(), 10000);
    },

    attachEvents() {
        document.getElementById('monitor-start').onclick = () => this.sendAction('start');
        document.getElementById('monitor-stop').onclick = () => this.sendAction('stop');
        document.getElementById('monitor-restart').onclick = () => this.sendAction('restart');
        document.getElementById('clear-log').onclick = () => this.sendAction('clearlog');
        document.getElementById('monitor-settings-form').onsubmit = (e) => this.saveConfig(e);
    }
};

function initMonitorTab() {
    MONITOR.init();
}