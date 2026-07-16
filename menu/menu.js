/**
 * Entware Manager - модуль динамического меню
 * Версия: 1.0
 * Дата: 2026-03-26
 * Описание: загружает конфигурацию меню из menu.json и динамически строит меню
 */

const Menu = (() => {
    let menuContainer = null;
    let menuItems = [];

    // Загрузка данных меню из JSON
    async function loadMenuData() {
        try {
            const response = await fetch(UI_BASE + '/menu/menu.json?_=' + Date.now());
            if (!response.ok) throw new Error('Network error');
            const data = await response.json();
            // ожидаем структуру { version, comment, items }
            return data.items || data; // fallback на старый формат, если нет items
        } catch (err) {
            console.error('Ошибка загрузки меню, используем дефолтное', err);
            // Дефолтное меню (запасной вариант)
            return [
                { tab: "stats", icon: "stats", text: "Статистика" },
                { tab: "packages", icon: "package", text: "Установленные" },
                { tab: "available", icon: "package", text: "Доступные" },
                { tab: "updates", icon: "update", text: "Обновления" },
                { tab: "processes", icon: "process", text: "Процессы" },
                { tab: "terminal", icon: "terminal", text: "Терминал" },
                { tab: "system-services", icon: "services", text: "Службы и Cron" },
                { tab: "settings", icon: "settings", text: "Настройки" },
                { tab: "monitor", icon: "shield", text: "Защита" },
                { tab: "help", icon: "help", text: "Справка" }
            ];
        }
    }

    // Отрисовка меню
    function renderMenu(items) {
        if (!menuContainer) return;
        menuContainer.innerHTML = '';
        items.forEach(item => {
            const li = document.createElement('li');
            li.dataset.tab = item.tab;
            li.innerHTML = `
                <span class="menu-icon">
                    <svg class="icon" width="16" height="16">
                        <use href="/entware-manager/icons.svg?v=2#icon-${item.icon}"/>
                    </svg>
                </span>
                <span class="menu-text">${escapeHtml(item.text)}</span>
            `;
            li.addEventListener('click', (e) => {
                if (window.innerWidth <= 800) {
                    const sidebar = document.getElementById('sidebar');
                    if (sidebar) sidebar.classList.remove('menu-open');
                }
                e.preventDefault();
                // Активируем пункт
                document.querySelectorAll('.menu li').forEach(i => i.classList.remove('active'));
                li.classList.add('active');
                const tabName = li.dataset.tab;
                if (typeof loadTab === 'function') {
                    loadTab(tabName);
                } else {
                    console.error('loadTab function not found');
                }
            });
            menuContainer.appendChild(li);
        });
        menuItems = items;
    }

    // Инициализация модуля
    async function init(containerSelector) {
        const container = document.querySelector(containerSelector);
        if (!container) {
            console.error('Menu container not found');
            return;
        }
        menuContainer = container;
        const items = await loadMenuData();
        renderMenu(items);
    }

    // Получить текущий список пунктов меню
    function getMenuItems() {
        return menuItems;
    }

    // Установить активный пункт по имени вкладки
    function setActiveTab(tabName) {
        if (!menuContainer) return;
        const items = menuContainer.querySelectorAll('li');
        items.forEach(li => {
            if (li.dataset.tab === tabName) {
                li.classList.add('active');
            } else {
                li.classList.remove('active');
            }
        });
    }

    return { init, getMenuItems, setActiveTab };
})();
