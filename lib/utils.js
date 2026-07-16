// Entware Manager - общие утилиты для JavaScript
// Версия: 1.1 (добавлены API_BASE, initTableSearch)
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

function initTableSearch(inputId, tableId, cellIndex) {
    var input = document.getElementById(inputId);
    var table = document.getElementById(tableId);
    if (!input || !table) return;
    var rows = table.getElementsByTagName('tr');
    input.addEventListener('keyup', function() {
        var filter = input.value.toLowerCase();
        for (var i = 1; i < rows.length; i++) {
            var cell = rows[i].getElementsByTagName('td')[cellIndex || 0];
            if (cell) {
                rows[i].style.display = cell.textContent.toLowerCase().indexOf(filter) > -1 ? '' : 'none';
            }
        }
    });
}
