// Entware Manager - общие утилиты для JavaScript
// Версия: 1.2 (добавлены apiFetch, apiGet, apiPost, apiPostJSON)
// Дата: 2026-07-16

const API_BASE = '/entware-cgi';
const UI_BASE = '/entware-manager';
const ICONS = UI_BASE + '/icons.svg?v=2';

function escapeHtml(text) {
    return String(text).replace(/[&<>"']/g, function(m) {
        if (m === '&') return '&amp;';
        if (m === '<') return '&lt;';
        if (m === '>') return '&gt;';
        if (m === '"') return '&quot;';
        if (m === "'") return '&#39;';
        return m;
    });
}

function apiFetch(path, options) {
    var url = API_BASE + path;
    if (!options || !options.method || options.method === 'GET') {
        var sep = path.indexOf('?') > -1 ? '&' : '?';
        url += sep + '_=' + Date.now();
    }
    return fetch(url, options).then(function(r) {
        if (r.status === 401 && path.indexOf('/login.cgi') === -1 && path.indexOf('/session.cgi') === -1) {
            // сессия истекла — вернуть к экрану входа
            if (typeof showLogin === 'function') showLogin();
            throw new Error('unauthorized');
        }
        if (!r.ok) throw new Error(r.statusText);
        return r;
    });
}

function apiGet(path) {
    return apiFetch(path).then(function(r) { return r.json(); });
}

function apiPost(path, body) {
    return apiFetch(path, {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: body
    }).then(function(r) { return r.json(); });
}

function apiPostJSON(path, data) {
    return apiFetch(path, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(data)
    }).then(function(r) { return r.json(); });
}

function initTableSearch(inputId, tableId, cellIndex) {
    var input = document.getElementById(inputId);
    var table = document.getElementById(tableId);
    if (!input || !table) return;
    var rows = table.getElementsByTagName('tr');
    input.addEventListener('keyup', function() {
        var filter = input.value.toLowerCase();
        var i;
        // Первый проход: обычные строки. Вложенные (.smart-usage-row) не участвуют в поиске.
        for (i = 1; i < rows.length; i++) {
            if (rows[i].className.indexOf('smart-usage-row') > -1) continue;
            var show = false;
            var cells = rows[i].getElementsByTagName('td');
            if (cellIndex < 0) {
                // Поиск по всем колонкам
                for (var j = 0; j < cells.length; j++) {
                    if (cells[j] && cells[j].textContent.toLowerCase().indexOf(filter) > -1) {
                        show = true;
                        break;
                    }
                }
            } else {
                // Поиск по конкретной колонке
                var cell = cells[cellIndex || 0];
                if (cell) show = cell.textContent.toLowerCase().indexOf(filter) > -1;
            }
            if (filter === '') {
                rows[i].style.removeProperty('display');
            } else {
                rows[i].style.display = show ? '' : 'none';
            }
        }
        // Второй проход: вложенные строки видимы только если родитель видим И открыт (.usage-open).
        for (i = 1; i < rows.length; i++) {
            if (rows[i].className.indexOf('smart-usage-row') === -1) continue;
            var parent = rows[i].previousElementSibling;
            if (parent && parent.style.display !== 'none' && parent.className.indexOf('usage-open') > -1) {
                rows[i].style.removeProperty('display');
            } else {
                rows[i].style.display = 'none';
            }
        }
    });
}

function loadScript(src) {
    return new Promise(function(resolve, reject) {
        if (document.querySelector('script[src="' + src + '"]')) {
            resolve();
            return;
        }
        var script = document.createElement('script');
        script.src = src;
        script.onload = resolve;
        script.onerror = reject;
        document.head.appendChild(script);
    });
}
