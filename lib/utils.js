// Entware Manager - общие утилиты для JavaScript
// Версия: 1.0
// Дата: 2026-03-31

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
