// Entware Manager - SMART мониторинг дисков
// Версия: 1.0 (унифицированный стиль)
// Дата: 2026-07-17

const SMART = {
    intervalId: null,
    currentTestDevice: null,
    testPollInterval: null,

    async init() {
        this.stopUpdates();
        this.renderHTML();
        await this.loadDisks();
    },

    stopUpdates() {
        if (this.intervalId) {
            clearInterval(this.intervalId);
            this.intervalId = null;
        }
        if (this.testPollInterval) {
            clearInterval(this.testPollInterval);
            this.testPollInterval = null;
        }
    },

    renderHTML() {
        contentDiv.innerHTML = `
            <h2 style="display: flex; align-items: center; gap: 8px;">
                <span class="stat-icon" style="width: 28px; height: 28px;">
                    <svg class="icon" width="28" height="28">
                        <use href="/entware-manager/icons.svg?v=2#icon-hdd"/>
                    </svg>
                </span>
                SMART дисков
            </h2>
            <div style="display: flex; gap: 8px; align-items: center; margin-bottom: 20px;">
                <div class="search-container" style="display: flex; gap: 8px; align-items: center; flex: 1; background: var(--input-bg); border: 2px solid var(--input-border); border-radius: 40px; padding: 0 12px; transition: border-color 0.3s ease, box-shadow 0.3s ease;">
                    <svg class="icon" width="18" height="18" style="color: var(--text-muted);"><use href="/entware-manager/icons.svg?v=2#icon-search"/></svg>
                    <input type="text" id="searchSmart" placeholder="Поиск по модели/серийнику..." style="flex: 1; background: transparent; border: none; outline: none; padding: 14px 0; font-size: 16px; color: var(--text-primary);">
                </div>
                <button id="refreshSmart" class="packages-delete-btn" style="background: #4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Обновить</button>
            </div>
            <div class="packages-table-wrapper">
                <table class="packages-table" id="smartTable">
                    <thead>
                        <tr>
                            <th>Устройство</th>
                            <th>Модель</th>
                            <th>Серийный №</th>
                            <th>Размер</th>
                            <th>Тип</th>
                            <th>Health</th>
                            <th>Temp</th>
                            <th>Power-On</th>
                            <th>Действия</th>
                        </tr>
                    </thead>
                    <tbody id="smartTableBody">
                        <tr><td colspan="9">Загрузка...</td></tr>
                    </tbody>
                </table>
            </div>
        `;

        document.getElementById('refreshSmart').addEventListener('click', () => this.loadDisks());
        initTableSearch('searchSmart', 'smartTable', 0);
    },

    async loadDisks() {
        const tbody = document.getElementById('smartTableBody');
        if (!tbody) return;
        tbody.innerHTML = '<tr><td colspan="9">Загрузка...</td></tr>';

        try {
            const data = await apiGet('/smart.cgi?action=list');
            this.renderTable(data.disks || []);
        } catch (err) {
            tbody.innerHTML = `<tr><td colspan="9" class="error">Ошибка: ${escapeHtml(err.message)}</td></tr>`;
        }
    },

    renderTable(disks) {
        const tbody = document.getElementById('smartTableBody');
        if (!tbody) return;

        if (disks.length === 0) {
            tbody.innerHTML = '<tr><td colspan="9">Диски не найдены или smartmontools не установлен.</td></tr>';
            return;
        }

        tbody.innerHTML = disks.map(disk => {
            const healthClass = disk.health === 'PASSED' ? 'status-running' : 'status-stopped';
            const healthIcon = disk.health === 'PASSED' ? 'icon-check' : 'icon-cross';
            const temp = disk.temperature != null ? disk.temperature : null;
            const tempClass = temp === null ? '' : (temp > 55 ? 'text-red' : (temp > 45 ? 'text-yellow' : ''));
            const tempText = temp !== null ? `${temp}°C` : '—';
            const powerOn = disk.power_on_hours ? `${disk.power_on_hours} ч` : '—';

            return `
                <tr data-device="${escapeHtml(disk.device)}">
                    <td>${escapeHtml(disk.device)}</td>
                    <td>${escapeHtml(disk.model || '—')}</td>
                    <td>${escapeHtml(disk.serial || '—')}</td>
                    <td>${escapeHtml(disk.size || '—')}</td>
                    <td>${escapeHtml(disk.type || '—')}</td>
                    <td><span class="status-badge ${healthClass}"><svg class="icon" width="12" height="12"><use href="/entware-manager/icons.svg?v=2#${healthIcon}"/></svg> ${escapeHtml(disk.health)}</span></td>
                    <td class="${tempClass}">${tempText}</td>
                    <td>${powerOn}</td>
                    <td>
                        <div style="display: flex; gap: 4px;">
                            <button class="packages-delete-btn smart-attr-btn" data-device="${escapeHtml(disk.device)}" style="padding: 4px 8px; font-size: 12px; background: #3182ce;">Атрибуты</button>
                            <button class="packages-delete-btn smart-health-btn" data-device="${escapeHtml(disk.device)}" style="padding: 4px 8px; font-size: 12px; background: #2c7a7b;">Health</button>
                            <button class="packages-delete-btn smart-test-btn" data-device="${escapeHtml(disk.device)}" style="padding: 4px 8px; font-size: 12px; background: #c05621;">Тест</button>
                        </div>
                    </td>
                </tr>
            `;
        }).join('');

        this.bindButtons();
    },

    bindButtons() {
        document.querySelectorAll('.smart-attr-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.showAttributes(btn.dataset.device);
            });
        });

        document.querySelectorAll('.smart-health-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.showHealth(btn.dataset.device);
            });
        });

        document.querySelectorAll('.smart-test-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.showTestDialog(btn.dataset.device);
            });
        });
    },

    async showAttributes(device) {
        Modal.loading('Загрузка атрибутов...');
        try {
            const data = await apiGet(`/smart.cgi?action=attributes&device=${encodeURIComponent(device)}`);
            if (data.error) { Modal.error(data.error); return; }

            let html = `
                <h3 style="margin-bottom: 12px;">Атрибуты SMART: ${escapeHtml(device)}</h3>
                <div style="overflow-x: auto;">
                    <table class="packages-table" style="width: 100%;">
                        <thead>
                            <tr><th>ID</th><th>Имя атрибута</th><th>Value</th><th>Worst</th><th>Thresh</th><th>Raw</th><th>Статус</th></tr>
                        </thead>
                        <tbody>
            `;

            (data.attributes || []).forEach(attr => {
                const isCritical = attr.value <= attr.threshold && attr.threshold > 0;
                const statusClass = isCritical ? 'status-stopped' : 'status-running';
                const statusIcon = isCritical ? 'icon-cross' : 'icon-check';
                const statusText = isCritical ? 'КРИТИЧНО' : 'OK';

                html += `
                    <tr class="${isCritical ? 'text-red' : ''}">
                        <td>${attr.id}</td>
                        <td>${escapeHtml(attr.name)}</td>
                        <td>${attr.value}</td>
                        <td>${attr.worst}</td>
                        <td>${attr.threshold}</td>
                        <td>${attr.raw}</td>
                        <td><span class="status-badge ${statusClass}"><svg class="icon" width="12" height="12"><use href="/entware-manager/icons.svg?v=2#${statusIcon}"/></svg> ${statusText}</span></td>
                    </tr>
                `;
            });

            html += `</tbody></table></div>`;
            Modal.show(html, false, `SMART атрибуты — ${escapeHtml(device)}`);
        } catch (err) {
            Modal.error('Ошибка: ' + err.message);
        }
    },

    async showHealth(device) {
        Modal.loading('Загрузка health...');
        try {
            const data = await apiGet(`/smart.cgi?action=health&device=${encodeURIComponent(device)}`);
            if (data.error) { Modal.error(data.error); return; }

            let html = `
                <h3 style="margin-bottom: 12px;">Health: ${escapeHtml(device)}</h3>
                <div style="font-family: monospace; white-space: pre-wrap; background: var(--input-bg); padding: 12px; border-radius: 6px;">
${escapeHtml(data.message || data.health || 'Нет данных')}
                </div>
            `;
            Modal.show(html, false, `SMART Health — ${escapeHtml(device)}`);
        } catch (err) {
            Modal.error('Ошибка: ' + err.message);
        }
    },

    showTestDialog(device) {
        let html = `
            <h3 style="margin-bottom: 12px;">Самодиагностика: ${escapeHtml(device)}</h3>
            <p>Выберите тип теста:</p>
            <div style="display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px;">
                <button class="packages-delete-btn" onclick="SMART.runTest('${escapeHtml(device)}', 'short')" style="background: #3182ce;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-play"/></svg> Короткий тест (~2 мин)
                </button>
                <button class="packages-delete-btn" onclick="SMART.runTest('${escapeHtml(device)}', 'long')" style="background: #2c7a7b;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-play"/></svg> Полный тест (~60-120 мин)
                </button>
                <button class="packages-delete-btn" onclick="SMART.runTest('${escapeHtml(device)}', 'conveyance')" style="background: #c05621;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-play"/></svg> Conveyance тест (~5 мин)
                </button>
            </div>
            <div id="smartTestStatus" style="font-family: monospace; background: var(--input-bg); padding: 12px; border-radius: 6px; min-height: 60px;">
                Нажмите кнопку для запуска теста...
            </div>
        `;
        Modal.show(html, false, `SMART тест — ${escapeHtml(device)}`);
    },

    async runTest(device, type) {
        const statusDiv = document.getElementById('smartTestStatus');
        if (!statusDiv) return;
        statusDiv.innerHTML = `Запуск ${type} теста...`;

        try {
            const data = await apiPost('/smart.cgi', `action=selftest&device=${encodeURIComponent(device)}&type=${type}`);
            if (data.error) { statusDiv.innerHTML = `Ошибка: ${escapeHtml(data.error)}`; return; }

            statusDiv.innerHTML = `Тест запущен: ${escapeHtml(data.message || 'OK')}`;

            // Polling статуса
            this.currentTestDevice = device;
            let attempts = 0;
            const maxAttempts = 240;
            const poll = async () => {
                try {
                    const d = await apiGet(`/smart.cgi?action=selftest&device=${encodeURIComponent(device)}`);
                    if (d.status) {
                        statusDiv.innerHTML = `Статус: ${escapeHtml(d.status)}<br>Прогресс: ${d.progress || '?'}%`;
                        if (d.status.includes('Completed') || d.status.includes('Aborted') || d.status.includes('Interrupted')) {
                            clearInterval(this.testPollInterval);
                            this.testPollInterval = null;
                            statusDiv.innerHTML += '<br><span class="status-badge status-running">Тест завершён</span>';
                        }
                    }
                } catch (e) { /* ignore */ }
                if (++attempts >= maxAttempts) { clearInterval(this.testPollInterval); this.testPollInterval = null; }
            };
            if (this.testPollInterval) clearInterval(this.testPollInterval);
            this.testPollInterval = setInterval(poll, 5000);
        } catch (err) {
            statusDiv.innerHTML = `Ошибка: ${err.message}`;
        }
    }
};

// Экспорт для совместимости
window.SMART = SMART;