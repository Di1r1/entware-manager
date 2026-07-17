// Entware Manager - SMART мониторинг дисков
// Версия: 1.1 (кликабельные зоны, цветные типы, подсветка здоровья)
// Дата: 2026-07-17

function formatSize(bytes) {
    const n = parseInt(bytes);
    if (!n || isNaN(n)) return '—';
    const GB = 1024 * 1024 * 1024;
    const TB = GB * 1024;
    if (n >= TB) return (n / TB).toFixed(1) + ' TB';
    if (n >= GB) return Math.round(n / GB) + ' GB';
    return Math.round(n / (1024 * 1024)) + ' MB';
}

const SMART = {
    intervalId: null,
    currentTestDevice: null,
    testPollInterval: null,

    // Описания атрибутов для всплывающих подсказок
    ATTR_DESC: {
        1:   'Read Error Rate — частота ошибок чтения с головок.',
        3:   'Spin-Up Time — время выхода шпинделя на рабочую скорость.',
        4:   'Start/Stop Count — количество циклов запуска/останова.',
        5:   'Reallocated Sectors Count — переназначенные сектора. Критический атрибут!',
        7:   'Seek Error Rate — частота ошибок позиционирования головок.',
        9:   'Power-On Hours — общее время работы в часах.',
        10:  'Spin Retry Count — попытки повторного раскрута шпинделя.',
        12:  'Power Cycle Count — количество циклов подачи питания.',
        184: 'End-to-End Error — ошибки на шине передачи данных.',
        187: 'Reported Uncorrectable Errors — неисправимые ошибки.',
        188: 'Command Timeout — число таймаутов выполнения команд.',
        189: 'High Fly Writes — количество записей на неоптимальной высоте.',
        190: 'Airflow Temperature — температура воздуха внутри корпуса.',
        193: 'Load Cycle Count — количество циклов парковки головок.',
        194: 'Temperature — температура диска.',
        196: 'Reallocation Event Count — количество операций переназначения.',
        197: 'Current Pending Sector Count — количество нестабильных секторов, ожидающих переназначения.',
        198: 'Uncorrectable Sector Count — количество неисправимых секторов.',
        199: 'UDMA CRC Error Count — количество ошибок CRC на интерфейсе.',
    },

    init() {
        this.stopUpdates();
        this.renderHTML();
        this.loadDisks();
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
                        </tr>
                    </thead>
                    <tbody id="smartTableBody">
                        <tr><td colspan="8">Загрузка...</td></tr>
                    </tbody>
                </table>
            </div>
        `;

        document.getElementById('refreshSmart').addEventListener('click', () => this.loadDisks());
        initTableSearch('searchSmart', 'smartTable', -1);
    },

    async loadDisks() {
        const tbody = document.getElementById('smartTableBody');
        if (!tbody) return;
        tbody.innerHTML = '<tr><td colspan="8">Загрузка...</td></tr>';

        try {
            const data = await apiGet('/smart.cgi?action=list');
            this.renderTable(data.disks || []);
        } catch (err) {
            tbody.innerHTML = `<tr><td colspan="8" class="error">Ошибка: ${escapeHtml(err.message)}</td></tr>`;
        }
    },

    renderTable(disks) {
        const tbody = document.getElementById('smartTableBody');
        if (!tbody) return;

        if (disks.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8">Диски не найдены или smartmontools не установлен.</td></tr>';
            return;
        }

        const TYPE_CLASSES = { hdd: 'drive-hdd', ssd: 'drive-ssd', nvme: 'drive-nvme', sat: 'drive-hdd', usb: 'drive-usb' };

        tbody.innerHTML = disks.map(disk => {
            const healthClass = disk.health === 'PASSED' ? 'status-running' : 'status-stopped';
            const healthIcon = disk.health === 'PASSED' ? 'icon-check' : 'icon-cross';
            const temp = disk.temperature != null ? disk.temperature : null;
            const tempClass = temp === null ? '' : (temp > 55 ? 'text-red' : (temp > 45 ? 'text-yellow' : ''));
            const tempText = temp !== null ? `${temp}°C` : '—';
            const powerOn = disk.power_on_hours ? `${disk.power_on_hours} ч` : '—';
            const typeLower = (disk.type || '').toLowerCase();
            const typeClass = TYPE_CLASSES[typeLower] || '';
            const healthBorderClass = disk.health === 'PASSED' ? 'smart-health-ok' : 'smart-health-critical';

            return `
                <tr data-device="${escapeHtml(disk.device)}" class="${healthBorderClass}">
                    <td class="clickable-device">${escapeHtml(disk.device)}</td>
                    <td>${escapeHtml(disk.model || '—')}</td>
                    <td>${escapeHtml(disk.serial || '—')}</td>
                    <td class="clickable-usage">${formatSize(disk.size)}</td>
                    <td class="${typeClass}">${escapeHtml(disk.type || '—')}</td>
                    <td class="clickable-health"><span class="status-badge ${healthClass}"><svg class="icon" width="12" height="12"><use href="/entware-manager/icons.svg?v=2#${healthIcon}"/></svg> ${escapeHtml(disk.health)}</span></td>
                    <td class="clickable-temp ${tempClass}">${tempText}</td>
                    <td>${powerOn}</td>
                </tr>
            `;
        }).join('');

        this.bindClickZones();
    },

    bindClickZones() {
        const tbody = document.getElementById('smartTableBody');
        if (!tbody) return;
        tbody.addEventListener('click', (e) => {
            const row = e.target.closest('tr');
            if (!row || !row.dataset.device) return;
            const device = row.dataset.device;
            if (e.target.closest('.clickable-health')) {
                this.showHealth(device);
            } else if (e.target.closest('.clickable-temp')) {
                this.showTestDialog(device);
            } else if (e.target.closest('.clickable-device')) {
                this.showInfo(device);
            } else if (e.target.closest('.clickable-usage')) {
                this.showUsage(device);
            } else {
                this.showAttributes(device);
            }
        });
    },

    async showInfo(device) {
        Modal.loading('Загрузка информации...');
        try {
            const data = await apiGet(`/smart.cgi?action=info&device=${encodeURIComponent(device)}`);
            if (data.error) { Modal.error(data.error); return; }
            let html = `
                <h3 style="margin-bottom: 12px;">Информация: ${escapeHtml(device)}</h3>
                <div style="font-family: monospace; white-space: pre-wrap; background: var(--input-bg); padding: 12px; border-radius: 6px; font-size: 13px;">
${escapeHtml(data.info || 'Нет данных')}
                </div>
            `;
            Modal.show(html, false, `SMART информация — ${escapeHtml(device)}`);
        } catch (err) {
            Modal.error('Ошибка: ' + err.message);
        }
    },

    async showUsage(device) {
        Modal.loading('Загрузка разделов...');
        try {
            const data = await apiGet(`/smart.cgi?action=usage&device=${encodeURIComponent(device)}`);
            if (data.error) { Modal.error(data.error); return; }
            const parts = data.partitions || [];
            if (parts.length === 0) {
                Modal.info('Разделы не найдены или df недоступен.', 'Разделы — ' + device);
                return;
            }
            let html = `
                <h3 style="margin-bottom: 12px;">Разделы: ${escapeHtml(device)}</h3>
                <div style="overflow-x: auto;">
                    <table class="packages-table" style="width: 100%;">
                        <thead><tr><th>Раздел</th><th>Точка</th><th>Размер</th><th>Исп.</th><th>Своб.</th><th>Занято</th></tr></thead>
                        <tbody>
            `;
            parts.forEach(p => {
                const pct = parseInt(p.pct) || 0;
                const color = pct >= 90 ? '#e53e3e' : (pct >= 80 ? '#d69e2e' : '#38a169');
                html += `<tr>
                    <td>${escapeHtml(p.part)}</td>
                    <td>${escapeHtml(p.mnt)}</td>
                    <td>${escapeHtml(p.size)}</td>
                    <td>${escapeHtml(p.used)}</td>
                    <td>${escapeHtml(p.avail)}</td>
                    <td><div style="display:flex;align-items:center;gap:8px;"><div style="flex:1;height:8px;background:var(--input-bg);border-radius:4px;overflow:hidden;"><div style="width:${pct}%;height:100%;background:${color};border-radius:4px;"></div></div><span style="font-weight:600;color:${color};">${pct}%</span></div></td>
                </tr>`;
            });
            html += `</tbody></table></div>`;
            Modal.show(html, false, `Разделы — ${escapeHtml(device)}`);
        } catch (err) {
            Modal.error('Ошибка: ' + err.message);
        }
    },

    async showAttributes(device) {
        Modal.loading('Загрузка атрибутов...');
        try {
            const data = await apiGet(`/smart.cgi?action=attributes&device=${encodeURIComponent(device)}`);
            if (data.error) { Modal.error(data.error); return; }

            let html = `
                <h3 style="margin-bottom: 12px;">Атрибуты SMART: ${escapeHtml(device)}</h3>
                <p style="font-size: 12px; color: var(--text-muted); margin-bottom: 8px;">Кликните на имя атрибута для справки</p>
                <div style="overflow-x: auto;">
                    <table class="packages-table" style="width: 100%;">
                        <thead>
                            <tr><th>ID</th><th>Имя атрибута</th><th>Value</th><th>Worst</th><th>Thresh</th><th>Raw</th><th>Статус</th></tr>
                        </thead>
                        <tbody>
            `;

            (data.attributes || []).forEach(attr => {
                const value = parseInt(attr.value) || 0;
                const threshold = parseInt(attr.threshold) || 0;
                const id = parseInt(attr.id) || 0;
                let impClass = '';
                let statusClass = 'status-running';
                let statusIcon = 'icon-check';
                let statusText = 'OK';

                // Важность атрибута (цвет текста)
                const CRITICAL_ATTRS = [5, 10, 187, 196, 197, 198];
                const IMPORTANT_ATTRS = [1, 3, 4, 7, 9, 12, 184, 188, 189, 190, 193, 194, 199];

                if (CRITICAL_ATTRS.includes(id)) {
                    impClass = 'smart-imp-critical';
                } else if (IMPORTANT_ATTRS.includes(id)) {
                    impClass = 'smart-imp-important';
                }

                if (threshold > 0) {
                    if (value <= threshold) {
                        statusClass = 'status-stopped';
                        statusIcon = 'icon-cross';
                        statusText = 'КРИТИЧНО';
                    } else if (value - threshold < 10) {
                        statusClass = 'status-warning';
                        statusIcon = 'icon-cross';
                        statusText = 'Предупреждение';
                    }
                }

                const desc = this.ATTR_DESC[id] || null;

                html += `
                    <tr class="smart-row-ok">
                        <td>${id}</td>
                        <td><span class="${impClass}${desc ? ' attr-name-clickable' : ''}"${desc ? ` data-desc="${escapeHtml(desc)}" title="${escapeHtml(desc)}"` : ''}>${escapeHtml(attr.name)}</span></td>
                        <td>${value}</td>
                        <td>${parseInt(attr.worst) || 0}</td>
                        <td>${threshold}</td>
                        <td>${escapeHtml(attr.raw)}</td>
                        <td><span class="status-badge ${statusClass}"><svg class="icon" width="12" height="12"><use href="/entware-manager/icons.svg?v=2#${statusIcon}"/></svg> ${statusText}</span></td>
                    </tr>
                `;
            });

            html += `
                        </tbody>
                    </table>
                </div>
                <div class="smart-legend">
                    <span class="smart-legend-item">
                        <span class="smart-legend-dot ok"></span>
                        OK
                    </span>
                    <span class="smart-legend-item">
                        <span class="smart-legend-dot warning"></span>
                        Предупреждение
                    </span>
                    <span class="smart-legend-item">
                        <span class="smart-legend-dot critical"></span>
                        Критично
                    </span>
                    <span class="smart-legend-item">
                        <span style="color: #e53e3e; font-weight: 700; font-size: 12px;">Критичный</span>
                        <span style="font-size: 12px;">атрибут</span>
                    </span>
                    <span class="smart-legend-item">
                        <span style="color: #d69e2e; font-weight: 600; font-size: 12px;">Важный</span>
                        <span style="font-size: 12px;">атрибут</span>
                    </span>
                </div>
            `;
            Modal.show(html, false, `SMART атрибуты — ${escapeHtml(device)}`);
            document.querySelectorAll('#modalBody .attr-name-clickable').forEach(el => {
                el.addEventListener('click', () => {
                    const d = el.dataset.desc;
                    if (d) Toast.show(d, false, 4000);
                });
            });
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