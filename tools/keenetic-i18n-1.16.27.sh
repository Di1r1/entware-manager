#!/bin/sh
set -eu

ROOT="/opt/web_entware"
INDEX="$ROOT/index.html"
BACKUP="$ROOT/index.html.before-local-i18n"
TMP_BLOCK="/tmp/ewm-i18n-block.$$.html"
TMP_INDEX="/tmp/ewm-index.$$.html"

cleanup() {
    rm -f "$TMP_BLOCK" "$TMP_INDEX"
}
trap cleanup EXIT INT TERM

[ -f "$INDEX" ] || { echo "ERROR: $INDEX bulunamadi"; exit 1; }

VERSION="unknown"
if [ -f "$ROOT/version.json" ]; then
    VERSION=$(grep -m1 '"version"' "$ROOT/version.json" | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' || true)
fi

echo "Entware Manager version: $VERSION"
if [ "$VERSION" != "1.16.27" ]; then
    echo "UYARI: Bu patch 1.16.27 icin hazirlandi. Mevcut surum: $VERSION"
fi

if [ ! -f "$BACKUP" ]; then
    cp -p "$INDEX" "$BACKUP"
    echo "Backup: $BACKUP"
else
    echo "Backup zaten var: $BACKUP"
fi

# Idempotent: previous local patch varsa once temiz kopyaya don.
if grep -q 'EWM_LOCAL_I18N_BEGIN' "$INDEX" 2>/dev/null; then
    cp -p "$BACKUP" "$INDEX"
    echo "Onceki local i18n patch temizlendi."
fi

cat > "$TMP_BLOCK" <<'EOF'
<!-- EWM_LOCAL_I18N_BEGIN -->
<script id="ewm-local-i18n">
(function () {
  'use strict';

  var STORAGE_KEY = 'entware_language';
  var SUPPORTED = { ru: 'Русский', en: 'English', tr: 'Türkçe' };
  var DICTS = {
    tr: {
      'Управление пакетами и системой':'Paket ve sistem yönetimi',
      'Проверка системы':'Sistem kontrolü',
      'Выйти':'Çıkış yap',
      'Показать заголовок':'Başlığı göster',
      'Показать футер':'Alt bilgiyi göster',
      'Информация':'Bilgi',
      'Загрузка...':'Yükleniyor...',
      'ошибка':'hata',
      'Статистика':'İstatistikler',
      'Пакеты':'Paketler',
      'Процессы':'İşlemler',
      'Терминал':'Terminal',
      'Службы и Cron':'Servisler ve Cron',
      'Сеть':'Ağ',
      'Модули':'Modüller',
      'Настройки':'Ayarlar',
      'Защита':'Koruma',
      'Справка':'Yardım',
      'Логи':'Günlükler',
      'Введите пароль панели (задаётся в Настройки → Защита)':'Panel parolasını girin (Ayarlar → Koruma bölümünden ayarlanır)',
      'Пароль':'Parola',
      'Войти':'Giriş yap',
      'Пароль панели:':'Panel parolası:',
      'Неверный пароль':'Hatalı parola',
      'Загрузка графиков...':'Grafikler yükleniyor...',
      'Мин':'Min',
      'Средняя':'Ortalama',
      'Макс':'Maks',
      'Сейчас':'Şimdi',
      'Нет данных. Графики появятся через несколько минут.':'Henüz veri yok. Grafikler birkaç dakika içinde görünecek.',
      'Температура':'Sıcaklık',
      'Ошибка загрузки данных':'Veri yükleme hatası',
      'Обновить':'Yenile',
      'Установить':'Kur',
      'Удалить':'Kaldır',
      'Запустить':'Başlat',
      'Остановить':'Durdur',
      'Перезапустить':'Yeniden başlat',
      'Сохранить':'Kaydet',
      'Отмена':'İptal',
      'Закрыть':'Kapat',
      'Добавить':'Ekle',
      'Изменить':'Düzenle',
      'Проверить':'Kontrol et',
      'Очистить':'Temizle',
      'Скачать':'İndir',
      'Восстановить':'Geri yükle',
      'Создать':'Oluştur',
      'Применить':'Uygula',
      'Сбросить':'Sıfırla',
      'Поиск':'Ara',
      'Включено':'Etkin',
      'Выключено':'Devre dışı',
      'Работает':'Çalışıyor',
      'Остановлен':'Durduruldu',
      'Да':'Evet',
      'Нет':'Hayır',
      'Имя':'Ad',
      'Статус':'Durum',
      'Действие':'İşlem',
      'Описание':'Açıklama',
      'Версия':'Sürüm',
      'Размер':'Boyut',
      'Свободно':'Boş',
      'Использовано':'Kullanılan',
      'Всего':'Toplam',
      'Интерфейс':'Arayüz',
      'Маршруты':'Rotalar',
      'Сетевые интерфейсы':'Ağ arayüzleri',
      'Подключено':'Bağlı',
      'Отключено':'Bağlı değil',
      'Скорость':'Hız',
      'Сигнал':'Sinyal',
      'Канал':'Kanal',
      'Службы':'Servisler',
      'Автозапуск':'Otomatik başlatma',
      'Запущен':'Başlatıldı',
      'Не запущен':'Çalışmıyor',
      'Обновления':'Güncellemeler',
      'Доступные':'Mevcut',
      'Установленные':'Kurulu',
      'Все':'Tümü',
      'Назад':'Geri',
      'Далее':'İleri',
      'Подтвердить':'Onayla',
      'Внимание':'Dikkat',
      'Ошибка':'Hata',
      'Успешно':'Başarılı',
      'Язык':'Dil',
      'Язык интерфейса':'Arayüz dili'
    },
    en: {
      'Управление пакетами и системой':'Package and system management',
      'Проверка системы':'System check',
      'Выйти':'Log out',
      'Показать заголовок':'Show header',
      'Показать футер':'Show footer',
      'Информация':'Information',
      'Загрузка...':'Loading...',
      'ошибка':'error',
      'Статистика':'Statistics',
      'Пакеты':'Packages',
      'Процессы':'Processes',
      'Терминал':'Terminal',
      'Службы и Cron':'Services & Cron',
      'Сеть':'Network',
      'Модули':'Modules',
      'Настройки':'Settings',
      'Защита':'Protection',
      'Справка':'Help',
      'Логи':'Logs',
      'Введите пароль панели (задаётся в Настройки → Защита)':'Enter the panel password (set in Settings → Protection)',
      'Пароль':'Password',
      'Войти':'Sign in',
      'Пароль панели:':'Panel password:',
      'Неверный пароль':'Incorrect password',
      'Загрузка графиков...':'Loading charts...',
      'Мин':'Min',
      'Средняя':'Average',
      'Макс':'Max',
      'Сейчас':'Now',
      'Нет данных. Графики появятся через несколько минут.':'No data yet. Charts will appear in a few minutes.',
      'Температура':'Temperature',
      'Ошибка загрузки данных':'Failed to load data',
      'Обновить':'Refresh',
      'Установить':'Install',
      'Удалить':'Remove',
      'Запустить':'Start',
      'Остановить':'Stop',
      'Перезапустить':'Restart',
      'Сохранить':'Save',
      'Отмена':'Cancel',
      'Закрыть':'Close',
      'Добавить':'Add',
      'Изменить':'Edit',
      'Проверить':'Check',
      'Очистить':'Clear',
      'Скачать':'Download',
      'Восстановить':'Restore',
      'Создать':'Create',
      'Применить':'Apply',
      'Сбросить':'Reset',
      'Поиск':'Search',
      'Включено':'Enabled',
      'Выключено':'Disabled',
      'Работает':'Running',
      'Остановлен':'Stopped',
      'Да':'Yes',
      'Нет':'No',
      'Имя':'Name',
      'Статус':'Status',
      'Действие':'Action',
      'Описание':'Description',
      'Версия':'Version',
      'Размер':'Size',
      'Свободно':'Free',
      'Использовано':'Used',
      'Всего':'Total',
      'Интерфейс':'Interface',
      'Маршруты':'Routes',
      'Сетевые интерфейсы':'Network interfaces',
      'Подключено':'Connected',
      'Отключено':'Disconnected',
      'Скорость':'Speed',
      'Сигнал':'Signal',
      'Канал':'Channel',
      'Службы':'Services',
      'Автозапуск':'Autostart',
      'Запущен':'Started',
      'Не запущен':'Not running',
      'Обновления':'Updates',
      'Доступные':'Available',
      'Установленные':'Installed',
      'Все':'All',
      'Назад':'Back',
      'Далее':'Next',
      'Подтвердить':'Confirm',
      'Внимание':'Warning',
      'Ошибка':'Error',
      'Успешно':'Success',
      'Язык':'Language',
      'Язык интерфейса':'Interface language'
    }
  };

  var language = readLanguage();
  var translating = false;
  var observer = null;

  function readLanguage() {
    try {
      var saved = localStorage.getItem(STORAGE_KEY);
      if (SUPPORTED[saved]) return saved;
    } catch (e) {}
    var browser = String((navigator && navigator.language) || '').toLowerCase().split('-')[0];
    return SUPPORTED[browser] ? browser : 'ru';
  }

  function dict() { return DICTS[language] || {}; }

  function translated(value) {
    if (!value || typeof value !== 'string' || language === 'ru') return value;
    var key = value.trim();
    if (!key) return value;
    var replacement = dict()[key];
    if (typeof replacement !== 'string') return value;
    var pos = value.indexOf(key);
    return pos < 0 ? value : value.slice(0, pos) + replacement + value.slice(pos + key.length);
  }

  function ignored(node) {
    var p = node && (node.parentElement || node);
    return !!(p && p.closest && p.closest('script,style,code,pre,textarea'));
  }

  function translateText(node) {
    if (!node || node.nodeType !== 3 || ignored(node)) return;
    var next = translated(node.nodeValue);
    if (next !== node.nodeValue) node.nodeValue = next;
  }

  function translateAttrs(el) {
    if (!el || el.nodeType !== 1 || ignored(el)) return;
    ['title','placeholder','aria-label'].forEach(function (name) {
      if (!el.hasAttribute(name)) return;
      var cur = el.getAttribute(name);
      var next = translated(cur);
      if (next !== cur) el.setAttribute(name, next);
    });
  }

  function translateTree(root) {
    if (!root || language === 'ru') return;
    translating = true;
    try {
      if (root.nodeType === 3) { translateText(root); return; }
      if (root.nodeType !== 1 && root.nodeType !== 9) return;
      if (root.nodeType === 1) translateAttrs(root);
      var walker = document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT);
      var node;
      while ((node = walker.nextNode())) {
        if (node.nodeType === 3) translateText(node); else translateAttrs(node);
      }
    } finally { translating = false; }
  }

  function setLanguage(lang) {
    if (!SUPPORTED[lang]) return;
    try { localStorage.setItem(STORAGE_KEY, lang); } catch (e) {}
    location.reload();
  }

  function makeSelect() {
    var select = document.createElement('select');
    select.className = 'ewm-language-select';
    select.title = language === 'tr' ? 'Arayüz dili' : language === 'en' ? 'Interface language' : 'Язык интерфейса';
    select.setAttribute('aria-label', select.title);
    select.style.cssText = 'width:100%;max-width:150px;margin:4px auto 6px;padding:4px 6px;border-radius:6px;border:1px solid var(--border-color,#64748b);background:var(--card-bg,#1f2937);color:var(--text-primary,#fff);font-size:11px;';
    Object.keys(SUPPORTED).forEach(function (code) {
      var o = document.createElement('option');
      o.value = code;
      o.textContent = SUPPORTED[code];
      select.appendChild(o);
    });
    select.value = language;
    select.onchange = function () { setLanguage(select.value); };
    return select;
  }

  function addSelector() {
    var footer = document.querySelector('.sidebar-footer');
    if (footer && !footer.querySelector('.ewm-language-select')) {
      var wrap = document.createElement('div');
      wrap.style.cssText = 'display:flex;justify-content:center;width:100%;';
      wrap.appendChild(makeSelect());
      footer.insertBefore(wrap, footer.firstChild);
    }
    var card = document.querySelector('.login-card');
    if (card && !card.querySelector('.ewm-language-select')) {
      var loginSelect = makeSelect();
      loginSelect.style.maxWidth = '100%';
      loginSelect.style.marginTop = '10px';
      card.appendChild(loginSelect);
    }
  }

  function start() {
    document.documentElement.lang = language;
    translateTree(document.body);
    addSelector();
    if (!observer) {
      observer = new MutationObserver(function (mutations) {
        if (translating || language === 'ru') return;
        mutations.forEach(function (m) {
          if (m.type === 'characterData') translateText(m.target);
          if (m.type === 'attributes') translateAttrs(m.target);
          if (m.addedNodes) Array.prototype.forEach.call(m.addedNodes, translateTree);
        });
        addSelector();
      });
      observer.observe(document.body, {subtree:true,childList:true,characterData:true,attributes:true,attributeFilter:['title','placeholder','aria-label']});
    }
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start, {once:true});
  else start();

  window.EwmLocalI18n = { language:function(){return language;}, setLanguage:setLanguage, supported:SUPPORTED };
})();
</script>
<!-- EWM_LOCAL_I18N_END -->
EOF

awk '
FNR==NR { block = block $0 ORS; next }
/<\/body>/ && !done { printf "%s", block; done=1 }
{ print }
END { if (!done) exit 42 }
' "$TMP_BLOCK" "$INDEX" > "$TMP_INDEX" || {
    echo "ERROR: index.html icinde </body> bulunamadi; dosya degistirilmedi."
    exit 1
}

mv "$TMP_INDEX" "$INDEX"
chmod 0644 "$INDEX"

echo "Patch uygulandi."
echo "Diller: Русский / English / Türkçe"
echo "Tarayicida Ctrl+F5 yap ve sidebar altindaki dil seciciyi kullan."
