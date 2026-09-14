#!/bin/sh
set -eu

ROOT="/opt/web_entware"
INDEX="$ROOT/index.html"
BACKUP="$ROOT/index.html.before-local-i18n-v4"
TMP_BLOCK="/tmp/ewm-i18n-v4-block.$$.html"
TMP_INDEX="/tmp/ewm-i18n-v4-index.$$.html"

cleanup() {
    rm -f "$TMP_BLOCK" "$TMP_INDEX"
}
trap cleanup EXIT INT TERM

[ -f "$INDEX" ] || { echo "ERROR: $INDEX bulunamadi"; exit 1; }
grep -q 'EWM_LOCAL_I18N_BEGIN' "$INDEX" 2>/dev/null || {
    echo "ERROR: temel i18n patch bulunamadi."
    exit 1
}
grep -q 'EWM_LOCAL_I18N_V3_BEGIN' "$INDEX" 2>/dev/null || {
    echo "UYARI: V3 bulunamadi; V4 yine de uygulanacak."
}

if [ ! -f "$BACKUP" ]; then
    cp -p "$INDEX" "$BACKUP"
    echo "Backup: $BACKUP"
fi

# V4 idempotent: eski V4 blogunu sil, V3 ve temel patch'i koru.
sed '/<!-- EWM_LOCAL_I18N_V4_BEGIN -->/,/<!-- EWM_LOCAL_I18N_V4_END -->/d' "$INDEX" > "$TMP_INDEX"
mv "$TMP_INDEX" "$INDEX"

cat > "$TMP_BLOCK" <<'EOB'
<!-- EWM_LOCAL_I18N_V4_BEGIN -->
<style id="ewm-local-i18n-v4-style">
.ewm-language-select {
  background:#fff !important;
  color:#111827 !important;
  border-color:#cbd5e1 !important;
  color-scheme:light !important;
}
.ewm-language-select option { background:#fff !important; color:#111827 !important; }
html.night .ewm-language-select,
body.night .ewm-language-select,
[data-theme="dark"] .ewm-language-select {
  background:#1f2937 !important;
  color:#f8fafc !important;
  border-color:#475569 !important;
  color-scheme:dark !important;
}
html.night .ewm-language-select option,
body.night .ewm-language-select option,
[data-theme="dark"] .ewm-language-select option {
  background:#1f2937 !important;
  color:#f8fafc !important;
}
</style>
<script id="ewm-local-i18n-v4">
(function () {
  'use strict';

  var EXACT = {"УСТАНОВЛЕН":"KURULUM","установлен":"kurulu","Удалить":"Kaldır","Установить":"Kur","Обновить пакет":"Paketi güncelle","есть обновление":"güncelleme var","доступен":"mevcut","Пакет":"Paket","Версия":"Sürüm","Статус":"Durum","Действие":"İşlem","htop не запущен. Откройте Настройки → Терминал, задайте пароль и нажмите Запустить.":"htop çalışmıyor. Ayarlar → Terminal bölümünü açın, parola belirleyin ve Başlat düğmesine basın.","Терминал не запущен. Откройте Настройки → Терминал, задайте пароль и нажмите Запустить.":"Terminal çalışmıyor. Ayarlar → Terminal bölümünü açın, parola belirleyin ve Başlat düğmesine basın.","СОСТОЯНИЕ":"DURUM","ТИП":"TÜR","Автозапуск при загрузке":"Açılışta otomatik başlat","загрузка...":"yükleniyor...","Загрузка...":"Yükleniyor...","УСТРОЙСТВО":"CİHAZ","МОДЕЛЬ":"MODEL","СЕРИЙНЫЙ №":"SERİ NO","HEALTH":"SAĞLIK","TEMP":"SICAKLIK","POWER-ON":"ÇALIŞMA SÜRESİ","РАЗДЕЛ":"BÖLÜM","ТОЧКА":"BAĞLAMA NOKTASI","ИСП.":"KULL.","СВОБ.":"BOŞ","ЗАНЯТО":"DOLULUK","SMART информация":"SMART bilgisi","Информация":"Bilgi","SMART атрибуты":"SMART öznitelikleri","Атрибуты SMART":"SMART öznitelikleri","Кликните на имя атрибута для справки":"Açıklama için öznitelik adına tıklayın","ИМЯ АТРИБУТА":"ÖZNİTELİK ADI","Предупреждение":"Uyarı","Критично":"Kritik","Критичный атрибут":"Kritik öznitelik","Важный атрибут":"Önemli öznitelik","SMART Health":"SMART Sağlığı","Health":"Sağlık","Прокси не запущен":"Proxy çalışmıyor","Запустить прокси":"Proxy'yi başlat","Остановить прокси":"Proxy'yi durdur","Этот порт уже используется":"Bu port zaten kullanımda","Порт":"Port","Роутер":"Yönlendirici","Щит":"Kalkan","Загрузка":"İndirme","График":"Grafik","Процесс":"İşlem","Терминал (ttyd)":"Terminal (ttyd)","Сохранить все на сервер":"Tümünü sunucuya kaydet","Сбросить по умолчанию":"Varsayılana sıfırla","Системный crontab (crontab -l)":"Sistem crontab'ı (crontab -l)","Сохранить системный crontab":"Sistem crontab'ını kaydet","Entware crontab (/opt/etc/crontab)":"Entware crontab'ı (/opt/etc/crontab)","Сохранить Entware crontab":"Entware crontab'ını kaydet","Включить":"Etkinleştir","Сохранить настройки":"Ayarları kaydet","Лог-файл не найден":"Günlük dosyası bulunamadı","Очистить лог":"Günlüğü temizle","Действия менеджера":"Yönetici işlemleri","Системные логи":"Sistem günlükleri","Очистить логи старше 30 дней":"30 günden eski günlükleri temizle","Ротация сейчас":"Şimdi döndür","Настройки логирования":"Günlükleme ayarları","Системные события":"Sistem olayları","Ошибка":"Hata","Успешно":"Başarılı","Внимание":"Dikkat","Сохранить":"Kaydet","Очистить":"Temizle","Закрыть":"Kapat","Отмена":"İptal","Да":"Evet","Нет":"Hayır"};
  var PARTS = {"Этот порт уже используется":"Bu port zaten kullanımda","Версия интерфейса:":"Arayüz sürümü:","версия загружается из":"sürüm şu dosyadan yüklenir:","интерфейс на базе CGI и вкладок":"CGI ve sekme tabanlı arayüz","Разработчик:":"Geliştirici:","Прокси не запущен":"Proxy çalışmıyor","Запустите прокси, чтобы подключиться к ПК по RDP.":"RDP ile bilgisayara bağlanmak için proxy'yi başlatın.","Сохранить системный crontab":"Sistem crontab'ını kaydet","Системный crontab":"Sistem crontab'ı","Сохранить Entware crontab":"Entware crontab'ını kaydet","Лог-файл не найден":"Günlük dosyası bulunamadı","Сохранить настройки":"Ayarları kaydet","Автозапуск при загрузке":"Açılışta otomatik başlat","загрузка...":"yükleniyor...","не запущен":"çalışmıyor","не запущена":"çalışmıyor","не установлен":"kurulu değil","процесс не найден":"işlem bulunamadı","сервис не отвечает":"servis yanıt vermiyor","Мбит/с":"Mbit/sn","Кбит/с":"Kbit/sn","Гбит/с":"Gbit/sn"};
  var HELP_HTML = "<h3>İstatistikler</h3>\n<p>Yönlendiricinin model, ana bilgisayar adı, mimari, çekirdek sürümü, çalışma süresi, RAM/CPU kullanımı, Entware paketleri ve <code>/opt</code> disk kullanımını gösterir. Tablo başlıklarına tıklayarak sıralama yapılabilir.</p>\n\n<h3>Paketler</h3>\n<p>Kurulu ve mevcut Entware paketlerini tek tabloda gösterir. Paket listelerini yenileyebilir, tüm paketleri güncelleyebilir, ada göre arayabilir ve paketleri kurup kaldırabilirsiniz.</p>\n<div class=\"command-block\">opkg list-installed<br>opkg list<br>opkg update<br>opkg list-upgradable<br>opkg install &lt;paket&gt;<br>opkg remove &lt;paket&gt;<br>opkg upgrade &lt;paket&gt;</div>\n\n<h3>İşlemler (htop)</h3>\n<p><strong>htop</strong>, ttyd üzerinden çalışan etkileşimli işlem görüntüleyicisidir. htop çalışmıyorsa Ayarlar → Terminal bölümünden parola belirleyip servisi başlatın. F3 arama, F5 ağaç görünümü, F6 sıralama, F9 işlem sinyali, q çıkış.</p>\n\n<h3>Terminal</h3>\n<p>Web terminali <code>/terminal/</code>, htop ise <code>/htop/</code> yolundan açılır. İki servis de yalnızca loopback portlarında dinler ve başlatılırken parola gerektirir. Terminal modu olarak Entware konsolu veya yönlendirici telnet konsolu seçilebilir.</p>\n\n<h3>Syncthing</h3>\n<p>Telefon, bilgisayar ve yönlendirici arasında bulutsuz dosya eşitlemesi sağlar. Kurulumdan sonra servisi bir kez başlatın ve Syncthing web arayüzünde kullanıcı/parola belirleyin.</p>\n<div class=\"command-block\">opkg update<br>opkg install syncthing<br>/opt/etc/init.d/S92syncthing start</div>\n\n<h3>Modüller</h3>\n<p>Yerel servisler <code>/opt/web_entware/bridge/</code> altındaki manifestolarla panele eklenir. Bir modül servis URL'siyle veya çalışan işlem adıyla izlenebilir. Yönetim düğmeleri için uygun <code>init.d</code> betiği ve yönetim izni gerekir.</p>\n<p>Basit bir süreç modülü örneği:</p>\n<div class=\"command-block\">{ \"id\": \"mydaemon\", \"name\": \"My Daemon\", \"process\": [\"mydaemon\"], \"init\": \"mydaemon\" }</div>\n\n<h3>Ağ</h3>\n<p>Arayüzleri, rotaları, ARP tablosunu, Wi‑Fi durumunu ve ağ izleme olaylarını gösterir. Ağ izleme daemon'u arayüz durum değişikliklerini ve internet erişimini takip eder.</p>\n<div class=\"command-block\">ip addr<br>ip route<br>ip neigh</div>\n\n<h3>Disk SMART</h3>\n<p>HDD, SSD, NVMe ve USB disklerini S.M.A.R.T. üzerinden izler. Model, seri numarası, boyut, sıcaklık, sağlık ve çalışma süresi bilgilerini gösterir. Disk satırına ve ilgili alanlara tıklayarak ayrıntılı bilgi, bölüm kullanımı, SMART öznitelikleri ve öz test ekranları açılır.</p>\n<div class=\"command-block\">smartctl -a /dev/sda | head -5</div>\n\n<h3>RDP</h3>\n<p>Yerel ağdaki bilgisayarlara tarayıcıdan RDP bağlantısı için <code>grdp-proxy</code> kullanılır. Proxy yalnızca <code>127.0.0.1</code> üzerinde dinler ve panel üzerinden erişilir. Windows kullanıcı adı/parolası yönlendiricide saklanmaz.</p>\n\n<h3>Servisler ve Cron</h3>\n<p><code>/opt/etc/init.d/</code> altındaki Entware servislerini başlatabilir, durdurabilir ve yeniden başlatabilirsiniz. Otomatik başlatma durumu S/K önekleriyle yönetilir. Service watchdog servislerin düşmesini izleyebilir ve istenirse otomatik yeniden başlatabilir.</p>\n<p>Sistem crontab'ı ve Entware crontab'ı ayrı editörlerden düzenlenir.</p>\n<div class=\"command-block\">crontab -l<br>crontab -e<br>cat /opt/etc/crontab</div>\n\n<h3>Ayarlar</h3>\n<p>ttyd/htop yönetimi, ana sayfa bağlantıları, bildirimler, panel güvenliği, bakım, yedekleme ve güncelleme seçenekleri burada bulunur. Panel parolası etkinleştirildiğinde oturum çereziyle korunan giriş ekranı kullanılır.</p>\n\n<h3>Koruma</h3>\n<p>Uzun süre yüksek CPU kullanan işlemleri izleyen koruma daemon'unu yönetir. Tarama aralığı, CPU eşiği, yüksek yük süresi, yok sayılacak işlemler ve taranacak azami işlem sayısı yapılandırılabilir.</p>\n\n<h3>Günlükler</h3>\n<p>Yönetici işlemleri ve sistem günlükleri görüntülenebilir. Eski günlükler temizlenebilir, günlük rotasyonu elle çalıştırılabilir ve günlükleme açılıp kapatılabilir.</p>\n\n<h3>Telegram bildirimleri</h3>\n<p>Bot token'ı ve <code>chat_id</code> ile sistem, ağ, servis, paket, sıcaklık ve kaynak uyarıları gönderilebilir. Bot komutları yalnızca yapılandırılmış kullanıcıya yanıt verir.</p>\n\n<h3>Sistem kontrolü</h3>\n<p>opkg, lighttpd, BusyBox araçları, cron, jq, ip, curl, bash ve bridge-utils gibi bağımlılıkları; ayrıca CGI/shell betiklerinin sözdizimini kontrol eder ve eksik paketler için öneriler gösterir.</p>\n\n<h3>Çevrimdışı kurulum ve yedekleme</h3>\n<p>İnternet erişimi olmayan yönlendiriciler için çevrimdışı kurulum paketi hazırlanabilir. Yedekleme; panel dosyalarını, yapılandırmaları ve paket listesini saklar.</p>\n\n<h3>Temalar</h3>\n<p>Tema düğmesiyle açık/koyu mod ve renk ön ayarları değiştirilebilir. Tema seçimi tarayıcının <code>localStorage</code> alanında saklanır.</p>\n\n<h3>Güvenlik notu</h3>\n<p>Terminal, htop ve RDP yardımcı servislerini yalnızca gerektiğinde çalıştırın. Paneli yerel ağ dışında kullanırken HTTPS/KeenDNS gibi güvenli erişim yöntemlerini tercih edin.</p>";

  function lang() {
    try {
      return localStorage.getItem('entware_language') || 'ru';
    } catch (e) {
      return 'ru';
    }
  }

  function preserveWhitespace(original, replacement) {
    var lead = (original.match(/^\s*/) || [''])[0];
    var trail = (original.match(/\s*$/) || [''])[0];
    return lead + replacement + trail;
  }

  function replaceParts(s) {
    var out = s;
    Object.keys(PARTS).sort(function(a,b){ return b.length-a.length; }).forEach(function(k) {
      if (out.indexOf(k) !== -1) out = out.split(k).join(PARTS[k]);
    });

    out = out.replace(/^SMART информация\s*—\s*/,'SMART bilgisi — ');
    out = out.replace(/^Информация:\s*/,'Bilgi: ');
    out = out.replace(/^SMART атрибуты\s*—\s*/,'SMART öznitelikleri — ');
    out = out.replace(/^Атрибуты SMART:\s*/,'SMART öznitelikleri: ');
    out = out.replace(/^SMART Health\s*—\s*/,'SMART Sağlığı — ');
    out = out.replace(/^Health:\s*/,'Sağlık: ');
    out = out.replace(/^Порт:\s*/,'Port: ');
    out = out.replace(/^Процессы:\s*/,'İşlemler: ');
    out = out.replace(/^Версия интерфейса:\s*/,'Arayüz sürümü: ');
    out = out.replace(/\(дата:\s*/g,'(tarih: ');
    out = out.replace(/\bМБ\b/g,'MB');
    out = out.replace(/\bГБ\b/g,'GB');
    out = out.replace(/\bКБ\b/g,'KB');
    return out;
  }

  function translateValue(value) {
    if (lang() !== 'tr' || typeof value !== 'string' || !value) return value;
    var core = value.trim();
    if (!core) return value;
    var translated = EXACT[core];
    if (typeof translated !== 'string') translated = replaceParts(core);
    if (translated === core) return value;
    return preserveWhitespace(value, translated);
  }

  function ignored(node) {
    var el = node && (node.parentElement || node);
    return !!(el && el.closest && el.closest('script,style,textarea,pre,code'));
  }

  function translateElement(el) {
    if (!el || el.nodeType !== 1 || ignored(el) || lang() !== 'tr') return;
    ['title','placeholder','aria-label','data-tooltip'].forEach(function(name) {
      if (!el.hasAttribute(name)) return;
      var cur = el.getAttribute(name);
      var next = translateValue(cur);
      if (next !== cur) el.setAttribute(name, next);
    });
    if (el.tagName === 'INPUT' && /^(button|submit|reset)$/i.test(el.type || '')) {
      var curv = el.value;
      var nextv = translateValue(curv);
      if (nextv !== curv) el.value = nextv;
    }
  }

  function translateNode(node) {
    if (!node || lang() !== 'tr') return;
    if (node.nodeType === 3) {
      if (ignored(node)) return;
      var next = translateValue(node.nodeValue);
      if (next !== node.nodeValue) node.nodeValue = next;
      return;
    }
    if (node.nodeType !== 1 && node.nodeType !== 9) return;
    if (node.nodeType === 1) translateElement(node);
    var walker = document.createTreeWalker(node, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT);
    var n;
    while ((n = walker.nextNode())) {
      if (n.nodeType === 3) {
        if (!ignored(n)) {
          var nt = translateValue(n.nodeValue);
          if (nt !== n.nodeValue) n.nodeValue = nt;
        }
      } else {
        translateElement(n);
      }
    }
  }

  function installTurkishHelp() {
    if (lang() !== 'tr') return;
    var hc = document.getElementById('helpContent');
    if (!hc || hc.getAttribute('data-ewm-tr-help') === '1') return;

    hc.innerHTML = HELP_HTML;
    hc.setAttribute('data-ewm-tr-help','1');

    var title = document.querySelector('.container h2');
    if (title && /Справка|Yardım/.test(title.textContent || '')) {
      var icon = title.querySelector('.stat-icon');
      title.innerHTML = '';
      if (icon) title.appendChild(icon);
      title.appendChild(document.createTextNode(' Entware Manager Yardımı'));
    }

    var note = document.querySelector('.container .note');
    if (note) {
      note.innerHTML = '<strong>🧬 Mimari:</strong> Entware Manager arayüzü Go tabanlı CGI uç noktaları ve panel bileşenlerinden oluşur. Çalıştırılabilir dosyalar <code>/opt/web_entware/cgi-bin/go/</code> altında bulunur; <code>go.cgi</code> uygun bileşeni ve uç noktayı çağırır.';
    }

    var search = document.getElementById('helpSearch');
    if (search) search.placeholder = 'Yardımda ara...';

    var intro = document.querySelectorAll('.container > p');
    for (var i=0;i<intro.length;i++) {
      if ((intro[i].textContent || '').indexOf('Ниже описаны') !== -1) {
        intro[i].textContent = 'Aşağıda panelin temel bölümleri, komut örnekleri ve yapılandırma notları açıklanmıştır.';
      }
    }
  }

  function remaining() {
    var found = [];
    var walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    var n;
    while ((n = walker.nextNode())) {
      if (ignored(n)) continue;
      var s = (n.nodeValue || '').trim();
      if (s && /[А-Яа-яЁё]/.test(s) && found.indexOf(s) === -1) found.push(s);
    }
    return found;
  }

  function run(root) {
    if (lang() !== 'tr') return;
    installTurkishHelp();
    translateNode(root || document.body);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function() { run(document.body); }, {once:true});
  } else {
    run(document.body);
  }

  var observer = new MutationObserver(function(ms) {
    if (lang() !== 'tr') return;
    installTurkishHelp();
    ms.forEach(function(m) {
      if (m.type === 'characterData') translateNode(m.target);
      if (m.type === 'attributes') translateElement(m.target);
      if (m.addedNodes) Array.prototype.forEach.call(m.addedNodes, translateNode);
    });
  });

  function startObserver() {
    if (!document.body) return;
    observer.observe(document.body, {
      subtree:true,
      childList:true,
      characterData:true,
      attributes:true,
      attributeFilter:['title','placeholder','aria-label','data-tooltip']
    });
  }
  if (document.body) startObserver();
  else document.addEventListener('DOMContentLoaded', startObserver, {once:true});

  window.EwmLocalI18nV4 = {
    rerun:function(){ run(document.body); },
    remaining:remaining
  };
})();
</script>
<!-- EWM_LOCAL_I18N_V4_END -->
EOB

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

echo "V4 patch uygulandi."
echo "Turkce arayuzde kalan Rusca metinler icin genisletilmis ceviri ve Turkce Yardim sayfasi eklendi."
echo "Tarayicida Ctrl+F5 yap."
