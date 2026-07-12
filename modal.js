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
