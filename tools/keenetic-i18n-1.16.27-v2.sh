#!/bin/sh
set -eu

ROOT="/opt/web_entware"
INDEX="$ROOT/index.html"
BACKUP="$ROOT/index.html.before-local-i18n-v2"
TMP_BLOCK="/tmp/ewm-i18n-v2-block.$$.html"
TMP_INDEX="/tmp/ewm-i18n-v2-index.$$.html"

cleanup() {
    rm -f "$TMP_BLOCK" "$TMP_INDEX"
}
trap cleanup EXIT INT TERM

[ -f "$INDEX" ] || { echo "ERROR: $INDEX bulunamadi"; exit 1; }

grep -q 'EWM_LOCAL_I18N_BEGIN' "$INDEX" 2>/dev/null || {
    echo "ERROR: once temel i18n patch uygulanmali."
    exit 1
}

if grep -q 'EWM_LOCAL_I18N_V2_BEGIN' "$INDEX" 2>/dev/null; then
    echo "V2 patch zaten uygulanmis."
    exit 0
fi

if [ ! -f "$BACKUP" ]; then
    cp -p "$INDEX" "$BACKUP"
    echo "Backup: $BACKUP"
fi

cat > "$TMP_BLOCK" <<'EOF'
<!-- EWM_LOCAL_I18N_V2_BEGIN -->
<style id="ewm-local-i18n-v2-style">
.ewm-language-select {
  background: #ffffff !important;
  color: #111827 !important;
  border-color: #cbd5e1 !important;
  color-scheme: light !important;
}
.ewm-language-select option {
  background: #ffffff !important;
  color: #111827 !important;
}
html.night .ewm-language-select {
  background: #1f2937 !important;
  color: #f8fafc !important;
  border-color: #475569 !important;
  color-scheme: dark !important;
}
html.night .ewm-language-select option {
  background: #1f2937 !important;
  color: #f8fafc !important;
}
</style>
<script id="ewm-local-i18n-v2">
(function () {
  'use strict';

  var EXTRA = {
    tr: {
      'Терминал и ссылки':'Terminal ve bağlantılar',
      'Уведомления':'Bildirimler',
      'Безопасность':'Güvenlik',
      'Обслуживание':'Bakım',
      'Управление ttyd':'ttyd yönetimi',
      'Текущее состояние ttyd':'Mevcut ttyd durumu',
      'htop (8089, доступ /htop/):':'htop (8089, erişim /htop/):',
      'Терминал (9089, доступ /terminal/):':'Terminal (9089, erişim /terminal/):',
      'Консоль Entware':'Entware konsolu',
      'Управление веб-терминалами ttyd.':'ttyd web terminallerini yönetir.',
      'Пароль обязателен':'Parola zorunludur',
      'для обоих сервисов — терминал доступен извне через панель, без пароля запуск запрещён.':'her iki servis için de; terminal panele üzerinden erişilebilir ve parola olmadan başlatılamaz.',
      'Доступ: панель →':'Erişim: panel →',
      'и':'ve',
      '(тот же origin, порты 9089/8089 слушают только loopback).':'(aynı origin; 9089/8089 portları yalnızca loopback üzerinde dinler).',
      'Управление ссылками на главной (общие для всех устройств)':'Ana sayfa bağlantılarını yönet (tüm cihazlar için ortak)',
      'Здесь можно добавлять, редактировать и удалять ссылки. Изменения сразу видны на всех устройствах.':'Buradan bağlantı ekleyebilir, düzenleyebilir ve silebilirsiniz. Değişiklikler tüm cihazlarda hemen görünür.',
      'Добавить ссылку':'Bağlantı ekle',
      'ИКОНКА':'SİMGE',
      'НАЗВАНИЕ':'AD',
      'ДЕЙСТВИЯ':'İŞLEMLER',
      'Роутер':'Yönlendirici',
      'Пакет':'Paket',
      'Щит':'Kalkan',
      'Папка':'Klasör',
      'Ссылка':'Bağlantı',
      'Текущий статус':'Mevcut durum',
      'Текущее состояние':'Mevcut durum',
      'Настройки терминала':'Terminal ayarları',
      'Настройки безопасности':'Güvenlik ayarları',
      'Настройки уведомлений':'Bildirim ayarları',
      'Системное обслуживание':'Sistem bakımı',
      'Состояние':'Durum',
      'Доступ':'Erişim',
      'порт':'port',
      'доступ':'erişim',
      'Запущено':'Çalışıyor',
      'Остановлено':'Durduruldu',
      'running':'çalışıyor',
      'stopped':'durduruldu',
      'Сохранено':'Kaydedildi',
      'Сохранить настройки':'Ayarları kaydet',
      'Настройки сохранены':'Ayarlar kaydedildi',
      'Включить':'Etkinleştir',
      'Отключить':'Devre dışı bırak',
      'Проверка':'Kontrol',
      'Тест':'Test',
      'Отправить тест':'Test gönder',
      'Токен':'Token',
      'Чат ID':'Sohbet ID',
      'Пароль панели':'Panel parolası',
      'Защита панели':'Panel koruması',
      'Включить защиту':'Korumayı etkinleştir',
      'Текущий пароль':'Mevcut parola',
      'Новый пароль':'Yeni parola',
      'Повторите пароль':'Parolayı tekrar girin',
      'Резервная копия':'Yedek',
      'Создать резервную копию':'Yedek oluştur',
      'Восстановление':'Geri yükleme',
      'Выберите файл':'Dosya seçin',
      'Обновление':'Güncelleme',
      'Проверить обновления':'Güncellemeleri kontrol et',
      'Последняя версия':'Son sürüm',
      'Текущая версия':'Mevcut sürüm',
      'Доступно обновление':'Güncelleme mevcut',
      'Обновлений нет':'Güncelleme yok'
    },
    en: {
      'Терминал и ссылки':'Terminal & links',
      'Уведомления':'Notifications',
      'Безопасность':'Security',
      'Обслуживание':'Maintenance',
      'Управление ttyd':'ttyd management',
      'Текущее состояние ttyd':'Current ttyd status',
      'htop (8089, доступ /htop/):':'htop (8089, access /htop/):',
      'Терминал (9089, доступ /terminal/):':'Terminal (9089, access /terminal/):',
      'Консоль Entware':'Entware console',
      'Управление веб-терминалами ttyd.':'Manage ttyd web terminals.',
      'Пароль обязателен':'A password is required',
      'для обоих сервисов — терминал доступен извне через панель, без пароля запуск запрещён.':'for both services; the terminal is reachable through the panel and cannot be started without a password.',
      'Доступ: панель →':'Access: panel →',
      '(тот же origin, порты 9089/8089 слушают только loopback).':'(same origin; ports 9089/8089 listen on loopback only).',
      'Управление ссылками на главной (общие для всех устройств)':'Manage home-page links (shared by all devices)',
      'Здесь можно добавлять, редактировать и удалять ссылки. Изменения сразу видны на всех устройствах.':'Add, edit and remove links here. Changes are immediately visible on all devices.',
      'Добавить ссылку':'Add link',
      'ИКОНКА':'ICON',
      'НАЗВАНИЕ':'NAME',
      'ДЕЙСТВИЯ':'ACTIONS',
      'Роутер':'Router',
      'Пакет':'Package',
      'Щит':'Shield',
      'Папка':'Folder',
      'Ссылка':'Link',
      'Текущий статус':'Current status',
      'Текущее состояние':'Current status',
      'Настройки терминала':'Terminal settings',
      'Настройки безопасности':'Security settings',
      'Настройки уведомлений':'Notification settings',
      'Системное обслуживание':'System maintenance',
      'Состояние':'Status',
      'Доступ':'Access',
      'Запущено':'Running',
      'Остановлено':'Stopped',
      'Сохранено':'Saved',
      'Сохранить настройки':'Save settings',
      'Настройки сохранены':'Settings saved',
      'Включить':'Enable',
      'Отключить':'Disable',
      'Проверка':'Check',
      'Тест':'Test',
      'Отправить тест':'Send test',
      'Токен':'Token',
      'Чат ID':'Chat ID',
      'Пароль панели':'Panel password',
      'Защита панели':'Panel protection',
      'Включить защиту':'Enable protection',
      'Текущий пароль':'Current password',
      'Новый пароль':'New password',
      'Повторите пароль':'Repeat password',
      'Резервная копия':'Backup',
      'Создать резервную копию':'Create backup',
      'Восстановление':'Restore',
      'Выберите файл':'Choose file',
      'Обновление':'Update',
      'Проверить обновления':'Check for updates',
      'Последняя версия':'Latest version',
      'Текущая версия':'Current version',
      'Доступно обновление':'Update available',
      'Обновлений нет':'No updates'
    }
  };

  function lang() {
    try {
      var v = localStorage.getItem('entware_language');
      return v === 'tr' || v === 'en' ? v : 'ru';
    } catch (e) { return 'ru'; }
  }

  function map() { return EXTRA[lang()] || {}; }

  function replaceValue(value) {
    if (!value || typeof value !== 'string' || lang() === 'ru') return value;
    var out = value;
    var keys = Object.keys(map()).sort(function(a,b){ return b.length - a.length; });
    for (var i = 0; i < keys.length; i++) {
      var k = keys[i];
      if (out.indexOf(k) !== -1) out = out.split(k).join(map()[k]);
    }
    return out;
  }

  function ignored(node) {
    var p = node && (node.parentElement || node);
    return !!(p && p.closest && p.closest('script,style,code,pre,textarea'));
  }

  function translateNode(node) {
    if (!node || lang() === 'ru') return;
    if (node.nodeType === 3) {
      if (ignored(node)) return;
      var next = replaceValue(node.nodeValue);
      if (next !== node.nodeValue) node.nodeValue = next;
      return;
    }
    if (node.nodeType !== 1 && node.nodeType !== 9) return;
    if (node.nodeType === 1 && !ignored(node)) {
      ['title','placeholder','aria-label'].forEach(function(name){
        if (!node.hasAttribute(name)) return;
        var cur = node.getAttribute(name);
        var next = replaceValue(cur);
        if (next !== cur) node.setAttribute(name, next);
      });
    }
    var walker = document.createTreeWalker(node, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT);
    var n;
    while ((n = walker.nextNode())) {
      if (n.nodeType === 3) {
        if (!ignored(n)) {
          var nt = replaceValue(n.nodeValue);
          if (nt !== n.nodeValue) n.nodeValue = nt;
        }
      } else if (!ignored(n)) {
        ['title','placeholder','aria-label'].forEach(function(name){
          if (!n.hasAttribute(name)) return;
          var cur = n.getAttribute(name);
          var next = replaceValue(cur);
          if (next !== cur) n.setAttribute(name, next);
        });
      }
    }
  }

  function run() {
    translateNode(document.body);
    var obs = new MutationObserver(function(ms){
      ms.forEach(function(m){
        if (m.type === 'characterData') translateNode(m.target);
        if (m.type === 'attributes') translateNode(m.target);
        if (m.addedNodes) Array.prototype.forEach.call(m.addedNodes, translateNode);
      });
    });
    obs.observe(document.body, {subtree:true,childList:true,characterData:true,attributes:true,attributeFilter:['title','placeholder','aria-label']});

    window.EwmLocalI18nV2 = {
      rerun: function(){ translateNode(document.body); },
      remaining: function(){
        var found = [];
        var walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
        var n;
        while ((n = walker.nextNode())) {
          if (!ignored(n) && /[А-Яа-яЁё]/.test(n.nodeValue || '')) {
            var s = (n.nodeValue || '').trim();
            if (s && found.indexOf(s) === -1) found.push(s);
          }
        }
        return found;
      }
    };
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', run, {once:true});
  else run();
})();
</script>
<!-- EWM_LOCAL_I18N_V2_END -->
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

echo "V2 patch uygulandi."
echo "Ayarlar sayfasi ve dinamik metinler icin ek ceviriler eklendi."
echo "Tarayicida Ctrl+F5 yap."