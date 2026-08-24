// Entware Manager - SMART мониторинг дисков
// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
// Версия: 1.1 (кликабельные зоны, цветные типы, подсветка здоровья)
// Дата: 2026-07-17

// formatSize — в lib/utils.js (единая реализация).


// Сортировка SMART-таблицы: типы колонок явные (Temp «45°C», Power-On «123 ч»).
// Открытые вложенные строки заполненности при сортировке схлопываются
// (пересоздаются лениво при следующем клике).
function enableSmartTableSorting() {
    const table = document.getElementById('smartTable');
    if (!table || table.dataset.sortable) return;
    initTableSorting(table, {
        dataTypes: ['string', 'string', 'string', 'size', 'string', 'string', 'number', 'number'],
        onSort: function(idx, dataType) {
            document.querySelectorAll('#smartTableBody .smart-usage-row').forEach(function(r) { r.remove(); });
            document.querySelectorAll('#smartTableBody tr.usage-open').forEach(function(r) { r.classList.remove('usage-open'); });
            sortTable(table, idx, dataType);
        }
    });
}

const SMART = {
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
        if (this.testPollInterval) {
            clearInterval(this.testPollInterval);
            this.testPollInterval = null;
        }
        if (this.reloadTimer) {
            clearTimeout(this.reloadTimer);
            this.reloadTimer = null;
        }
        this.reloadAttempts = 0;
    },

    renderHTML() {
        contentDiv.innerHTML = `
            <h2 style="display: flex; align-items: center; gap: 8px;">
                <span class="stat-icon" style="width: 28px; height: 28px;">
                    <svg class="icon" width="28" height="28">
                        <use href="/entware-manager/icons.svg?v=6#icon-hdd"/>
                    </svg>
                </span>
                SMART дисков
            </h2>
            <div style="display: flex; gap: 8px; align-items: center; margin-bottom: 20px;">
                <div class="search-container" style="display: flex; gap: 8px; align-items: center; flex: 1; background: var(--input-bg); border: 2px solid var(--input-border); border-radius: 40px; padding: 0 12px; transition: border-color 0.3s ease, box-shadow 0.3s ease;">
                    <svg class="icon" width="18" height="18" style="color: var(--text-muted);"><use href="/entware-manager/icons.svg?v=6#icon-search"/></svg>
                    <input type="text" id="searchSmart" placeholder="Поиск по модели/серийнику..." style="flex: 1; background: transparent; border: none; outline: none; padding: 14px 0; font-size: 16px; color: var(--text-primary);">
                </div>
                <button id="refreshSmart" class="packages-delete-btn btn-neutral"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Обновить</button>
            </div>
            <div id="smart-table-container" class="packages-table-wrapper">
                <div class="loading-spinner"></div>
            </div>
        `;

        document.getElementById('refreshSmart').addEventListener('click', () => this.loadDisks(true));
        initTableSearch('searchSmart', 'smartTable', -1);
        enableSmartTableSorting();
    },

    async loadDisks(forceRefresh, silent) {
        const container = document.getElementById('smart-table-container');
        if (!container) return;
        // Guard от наложения: пока идёт запрос (особенно долгий refresh до 65с
        // на спящем диске), не запускаем параллельный. По завершении сброс.
        if (this.reloading) return;
        this.reloading = true;
        this.setRefreshing(true);
        // При фоновой дозагрузке не прячем таблицу за спиннером — только
        // перерисовываем содержимое, когда пришли данные.
        if (!silent) container.innerHTML = '<div class="loading-spinner"></div>';

        try {
            const url = forceRefresh ? '/smart.cgi?action=list&refresh=1' : '/smart.cgi?action=list';
            const data = await apiGet(url);
            this.renderTable(data.disks || []);
            this.scheduleReload(data.disks || []);
        } catch (err) {
            if (!silent) container.innerHTML = `<p class="error" style="padding:1rem;">Ошибка: ${escapeHtml(err.message)}</p>`;
        } finally {
            this.reloading = false;
            this.setRefreshing(false);
        }
    },

    // Индикация состояния кнопки «Обновить»: дизейбл + подпись во время запроса,
    // чтобы клик во время in-flight не выглядел «пропавшим».
    setRefreshing(on) {
        const btn = document.getElementById('refreshSmart');
        if (!btn) return;
        btn.disabled = on;
        btn.innerHTML = on
            ? '<svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Обновление…'
            : '<svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Обновить';
    },

    // Асинхронная дозагрузка: если в ответе есть диски со статусом loading или
    // busy (ещё «просыпаются»/на диске висит незавершённый smartctl), повторяем
    // запрос с паузой, пока все не ответят полными данными.
    scheduleReload(disks) {
        const hasPending = (disks || []).some(d => d.attr_health === 'loading' || d.attr_health === 'busy');
        if (!hasPending) { this.reloadAttempts = 0; return; }
        if (this.reloadTimer) return;
        if (++this.reloadAttempts > 30) { this.reloadAttempts = 0; return; }
        this.reloadTimer = setTimeout(() => {
            this.reloadTimer = null;
            this.loadDisks(true, true);
        }, 5000);
    },

    renderTable(disks) {
        const container = document.getElementById('smart-table-container');
        if (!container) return;

        if (disks.length === 0) {
            container.innerHTML = '<p style="padding:1rem;">Диски не найдены или smartmontools не установлен.</p>';
            return;
        }

        const TYPE_CLASSES = { hdd: 'drive-hdd', ssd: 'drive-ssd', nvme: 'drive-nvme', sat: 'drive-hdd', usb: 'drive-usb' };

        let html = `<table class="packages-table" id="smartTable">
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
            <tbody id="smartTableBody">`;

        html += disks.map(disk => {
            const attrHealth = disk.attr_health || 'ok';
            let healthClass = 'status-running';
            let healthIcon = 'icon-check';
            let healthText = disk.health;
            if (attrHealth === 'inactive') {
                healthClass = '';
                healthIcon = '';
                healthText = '—';
            } else if (attrHealth === 'loading') {
                healthClass = 'status-warning';
                healthIcon = 'icon-refresh';
                healthText = 'Загрузка…';
            } else if (attrHealth === 'busy') {
                healthClass = 'status-warning';
                healthIcon = 'icon-alert';
                healthText = 'Не отвечает';
            } else if (attrHealth === 'warning') {
                healthClass = 'status-warning';
                healthIcon = 'icon-alert';
            } else if (attrHealth === 'critical' || disk.health !== 'PASSED') {
                healthClass = 'status-stopped';
                healthIcon = 'icon-cross';
            }
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
                    <td class="clickable-usage"><span class="usage-arrow"><svg class="icon" width="12" height="12"><use href="/entware-manager/icons.svg?v=6#icon-arrow-right"/></svg></span> ${formatSize(disk.size)}</td>
                    <td class="${typeClass}">${escapeHtml(disk.type || '—')}</td>
                    <td class="clickable-health">${healthClass ? `<span class="status-badge ${healthClass}"><svg class="icon" width="12" height="12"><use href="/entware-manager/icons.svg?v=6#${healthIcon}"/></svg> ${escapeHtml(healthText)}</span>` : `<span style="color: var(--text-muted);">${escapeHtml(healthText)}</span>`}</td>
                    <td class="clickable-temp ${tempClass}">${tempText}</td>
                    <td>${powerOn}</td>
                </tr>
            `;
        }).join('');

        html += '</tbody></table>';
        container.innerHTML = html;
        this.bindClickZones();
        initTableSearch('searchSmart', 'smartTable', -1);
        enableSmartTableSorting();
    },

    bindClickZones() {
        const tbody = document.getElementById('smartTableBody');
        if (!tbody) return;
        if (tbody.dataset.clickBound) return;
        tbody.dataset.clickBound = '1';
        tbody.addEventListener('click', (e) => {
            if (e.target.closest('.smart-usage-row')) return;
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
                this.toggleUsage(row, device);
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

    async toggleUsage(row, device) {
        const tbody = document.getElementById('smartTableBody');
        if (!tbody) return;
        const next = row.nextElementSibling;
        if (next && next.classList.contains('smart-usage-hidden')) {
            next.classList.remove('smart-usage-hidden');
            next.style.display = '';
            row.classList.add('usage-open');
            return;
        }
        if (next && next.classList.contains('smart-usage-loaded')) {
            next.classList.add('smart-usage-hidden');
            next.style.display = 'none';
            row.classList.remove('usage-open');
            return;
        }
        if (next && next.classList.contains('smart-usage-row')) {
            next.remove();
            row.classList.remove('usage-open');
            return;
        }
        row.classList.add('usage-open');
        const usageRow = document.createElement('tr');
        usageRow.className = 'smart-usage-row smart-usage-loading';
        usageRow.innerHTML = '<td colspan="8"><div class="smart-usage-cell"><span class="smart-usage-spinner"></span> Загрузка разделов...</div></td>';
        row.insertAdjacentElement('afterend', usageRow);
        try {
            const data = await apiGet(`/smart.cgi?action=usage&device=${encodeURIComponent(device)}`);
            if (this.rowRemoved(usageRow, row)) return;
            const parts = data.partitions || [];
            if (parts.length === 0) {
                usageRow.innerHTML = '<td colspan="8"><div class="smart-usage-cell">Разделы не найдены или df недоступен.</div></td>';
                return;
            }
            let html = '<td colspan="8"><div class="smart-usage-cell"><div style="overflow-x:auto;"><table class="packages-table" style="width:100%;">';
            html += '<thead><tr><th>Раздел</th><th>Точка</th><th>Размер</th><th>Исп.</th><th>Своб.</th><th>Занято</th></tr></thead><tbody>';
            parts.forEach(p => {
                const pct = parseInt(p.pct) || 0;
                const usageClass = pct >= 90 ? 'smart-usage-critical' : (pct >= 80 ? 'smart-usage-warning' : 'smart-usage-ok');
                html += `<tr>
                    <td>${escapeHtml(p.part)}</td>
                    <td>${escapeHtml(p.mnt)}</td>
                    <td>${escapeHtml(p.size)}</td>
                    <td>${escapeHtml(p.used)}</td>
                    <td>${escapeHtml(p.avail)}</td>
                    <td><div style="display:flex;align-items:center;gap:8px;"><div style="flex:1;height:8px;background:var(--input-bg);border-radius:4px;overflow:hidden;"><div style="width:${pct}%;height:100%;" class="smart-usage-fill ${usageClass}"></div></div><span class="${usageClass}">${pct}%</span></div></td>
                </tr>`;
            });
            html += '</tbody></table></div></div></td>';
            usageRow.innerHTML = html;
            usageRow.classList.remove('smart-usage-loading');
            usageRow.classList.add('smart-usage-loaded');
        } catch (err) {
            if (this.rowRemoved(usageRow, row)) return;
            usageRow.innerHTML = `<td colspan="8"><div class="smart-usage-cell">Ошибка: ${escapeHtml(err.message)}</div></td>`;
        }
    },

    rowRemoved(usageRow, parentRow) {
        return !usageRow.isConnected || parentRow.style.display === 'none';
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

                // Важность атрибута — единый источник: флаг importance из бэкенда
                // (smart.go attrImportance); fallback на локальные списки для старых ответов.
                const CRITICAL_ATTRS = [5, 10, 187, 196, 197, 198];
                const IMPORTANT_ATTRS = [1, 3, 4, 7, 9, 12, 184, 188, 189, 190, 193, 194, 199];
                const imp = attr.importance || (CRITICAL_ATTRS.includes(id) ? 'critical'
                    : IMPORTANT_ATTRS.includes(id) ? 'important' : '');

                if (imp === 'critical') {
                    impClass = 'smart-imp-critical';
                } else if (imp === 'important') {
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
                        <td><span class="status-badge ${statusClass}"><svg class="icon" width="12" height="12"><use href="/entware-manager/icons.svg?v=6#${statusIcon}"/></svg> ${statusText}</span></td>
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
                        <span class="smart-imp-critical" style="font-size: 12px;">Критичный</span>
                        <span style="font-size: 12px;">атрибут</span>
                    </span>
                    <span class="smart-legend-item">
                        <span class="smart-imp-important" style="font-size: 12px;">Важный</span>
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
                <button class="packages-delete-btn smart-btn-test-short" data-device="${escapeHtml(device)}" data-test-type="short">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-play"/></svg> Короткий тест (~2 мин)
                </button>
                <button class="packages-delete-btn smart-btn-test-long" data-device="${escapeHtml(device)}" data-test-type="long">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-play"/></svg> Полный тест (~60-120 мин)
                </button>
                <button class="packages-delete-btn smart-btn-test-conveyance" data-device="${escapeHtml(device)}" data-test-type="conveyance">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-play"/></svg> Conveyance тест (~5 мин)
                </button>
            </div>
            <div id="smartTestStatus" style="font-family: monospace; background: var(--input-bg); padding: 12px; border-radius: 6px; min-height: 60px;">
                Нажмите кнопку для запуска теста...
            </div>
        `;
        Modal.show(html, false, `SMART тест — ${escapeHtml(device)}`);
        document.querySelectorAll('#modalBody button[data-test-type]').forEach(btn => {
            btn.addEventListener('click', () => this.runTest(btn.dataset.device, btn.dataset.testType));
        });
    },

    async runTest(device, type) {
        const statusDiv = document.getElementById('smartTestStatus');
        if (!statusDiv) return;
        statusDiv.innerHTML = `Запуск ${escapeHtml(type)} теста...`;

        try {
            const data = await apiPost('/smart.cgi', `action=selftest&device=${encodeURIComponent(device)}&type=${type}`);
            if (data.error) { statusDiv.innerHTML = `Ошибка: ${escapeHtml(data.error)}`; return; }

            statusDiv.innerHTML = `Тест запущен: ${escapeHtml(data.message || 'OK')}`;

            // Polling статуса
            let attempts = 0;
            const maxAttempts = 240;
            const poll = async () => {
                if (statusDiv.offsetParent === null) {
                    clearInterval(this.testPollInterval);
                    this.testPollInterval = null;
                    return;
                }
                try {
                    const d = await apiGet(`/smart.cgi?action=selftest&device=${encodeURIComponent(device)}`);
                    if (d.status) {
                        statusDiv.innerHTML = `Статус: ${escapeHtml(d.status)}<br>Прогресс: ${escapeHtml(d.progress || '?')}%`;
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
            statusDiv.innerHTML = `Ошибка: ${escapeHtml(err.message)}`;
        }
    }
};

// Экспорт для совместимости
window.SMART = SMART;