// ==============================================
// Entware Manager - модуль уведомлений (Modal и Toast)
// Версия: 1.0
// Дата: 2026-03-28
// ==============================================

const Modal = {
    element: null,
    bodyElement: null,
    titleElement: null,

    init() {
        this.element = document.getElementById('infoModal');
        this.bodyElement = document.getElementById('modalBody');
        this.titleElement = document.getElementById('modalTitle');
        const closeBtn = this.element?.querySelector('.close');
        if (closeBtn) {
            closeBtn.onclick = () => this.hide();
        }
        window.onclick = (event) => {
            if (event.target === this.element) this.hide();
        };
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') this.hide();
        });
    },

    show(content, isError = false, title = '') {
        if (!this.element || !this.bodyElement) return;
        if (this.titleElement && title) {
            this.titleElement.textContent = title;
        }
        if (isError) {
            this.element.classList.add('error-modal');
        } else {
            this.element.classList.remove('error-modal');
        }
        if (typeof content === 'string' && content.trim().startsWith('<')) {
            this.bodyElement.innerHTML = content;
        } else {
            this.bodyElement.textContent = content;
        }
        this.element.style.display = 'block';
    },

    hide() {
        if (this.element) this.element.style.display = 'none';
    },

    info(content, title = 'Информация') {
        this.show(content, false, title);
    },

    error(content, title = 'Ошибка') {
        this.show(content, true, title);
    },

    loading(title = 'Загрузка...') {
        this.show('<div class="loading-spinner"></div>', false, title);
    },

    // Запрос пароля в скрытом поле (type=password) через модалку.
    // onConfirm(value) вызывается со введённым паролем; '' — отмена.
    promptPassword(title, onConfirm) {
        if (!this.element || !this.bodyElement) { onConfirm(''); return; }
        if (this.titleElement) this.titleElement.textContent = title || 'Введите пароль';
        this.element.classList.remove('error-modal');
        this.bodyElement.innerHTML = `
            <div style="padding:6px 0 4px;">
                <input type="password" id="pwInput" placeholder="Пароль"
                    style="width:100%;padding:10px 12px;border-radius:8px;border:1px solid var(--input-border);background:var(--input-bg);color:var(--text-primary);font-size:14px;box-sizing:border-box;">
            </div>
            <div style="display:flex;gap:8px;justify-content:flex-end;margin-top:14px;">
                <button id="pwCancel" class="packages-delete-btn" style="background:#4a5568;">Отмена</button>
                <button id="pwOk" class="packages-delete-btn"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=5#icon-check"/></svg> OK</button>
            </div>`;
        this.element.style.display = 'block';

        var input = document.getElementById('pwInput');
        var ok = document.getElementById('pwOk');
        var cancel = document.getElementById('pwCancel');
        var closeBtn = this.element.querySelector('.close');
        var settled = false;

        // Временный обработчик клика по фону модалки (outsideClick) — снимается
        // в done(), чтобы не ломать крестик у последующих модалок (напр. просмотра
        // файла через Modal.info, где closeBtn.onclick из init() должен работать).
        var outsideClick = function(event) {
            if (event.target === Modal.element) done('');
        };

        var done = function(val) {
            if (settled) return;
            settled = true;
            // Восстанавливаем штатное поведение крестика (Modal.hide из init)
            // и убираем временный слушатель фона — иначе крестик у следующих
            // модалок остаётся привязанным к этому (уже завершённому) done.
            if (closeBtn) closeBtn.onclick = function() { Modal.hide(); };
            Modal.element.removeEventListener('click', outsideClick);
            Modal.hide();
            onConfirm(val);
        };
        input.focus();
        ok.onclick = function() { done(input.value); };
        cancel.onclick = function() { done(''); };
        input.addEventListener('keydown', function(e) {
            if (e.key === 'Enter') { e.preventDefault(); done(input.value); }
            if (e.key === 'Escape') { e.preventDefault(); done(''); }
        });
        // Закрытие через X или клик вне — тоже отмена (onConfirm('')).
        if (closeBtn) closeBtn.onclick = function() { done(''); };
        this.element.addEventListener('click', outsideClick);
    }
};

const Toast = {
    element: null,
    init() {
        this.element = document.createElement('div');
        this.element.id = 'toast';
        document.body.appendChild(this.element);
    },
    show(message, isError = false, duration = 3000) {
        if (!this.element) this.init();
        this.element.textContent = message;
        this.element.style.backgroundColor = isError ? '#e53e3e' : '#2ecc71';
        this.element.style.opacity = '1';
        setTimeout(() => {
            this.element.style.opacity = '0';
        }, duration);
    }
};

document.addEventListener('DOMContentLoaded', () => Modal.init());
