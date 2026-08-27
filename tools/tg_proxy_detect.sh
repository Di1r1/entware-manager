#!/bin/sh
# tg_proxy_detect.sh — найти рабочий локальный прокси для Telegram Bot API.
# BusyBox-compatible (Keenetic/Entware). Запуск: sh tg_proxy_detect.sh [BOT_TOKEN]
# Выводит готовую строку для поля «Прокси» (Настройки -> Telegram) и chat_id.
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
set -u

CONFIG="/opt/web_entware/telegram_config.json"
CURL=/opt/bin/curl
[ -x "$CURL" ] || CURL=curl

TOKEN="${1:-}"
if [ -z "$TOKEN" ] && command -v jq >/dev/null 2>&1 && [ -f "$CONFIG" ]; then
  TOKEN=$(jq -r '.bot_token // ""' "$CONFIG" 2>/dev/null)
fi
if [ -z "$TOKEN" ]; then
  echo "Токен не найден. Укажите: $0 <BOT_TOKEN>"
  echo "или заполните bot_token в $CONFIG, затем запустите без аргументов."
  exit 1
fi

# Уже настроенный в панели прокси — пробуем его первым (приоритет).
CFG_PROXY=""
if [ -f "$CONFIG" ] && command -v jq >/dev/null 2>&1; then
  CFG_PROXY=$(jq -r '.proxy_url // ""' "$CONFIG" 2>/dev/null)
fi

# --- Собрать кандидатов: mixed-inbound sing-box (HTTP+SOCKS одновременно) ---
# has_word — есть ли слово $2 в списке $1 (пробел/перевод строки разделители).
has_word() {
  for w in $1; do
    [ "$w" = "$2" ] && return 0
  done
  return 1
}

cands=""
if [ -d /opt/etc/awg-manager/singbox/config.d ]; then
  cands=$(grep -rE '"type":[[:space:]]*"mixed"' -A2 \
            /opt/etc/awg-manager/singbox/config.d/*.json 2>/dev/null \
            | grep -oE '"listen_port":[[:space:]]*[0-9]+' | grep -oE '[0-9]+' | sort -u)
fi
# Порт из уже настроенного прокси — пробуем первым.
if [ -n "$CFG_PROXY" ]; then
  CFG_PORT=$(echo "$CFG_PROXY" | sed -n 's#.*:\([0-9][0-9]*\)/*$#\1#p')
  if [ -n "$CFG_PORT" ]; then
    has_word "$cands" "$CFG_PORT" || cands="$CFG_PORT $cands"
  fi
fi
# Fallback на типичные порты прокси.
for p in 1080 1081 1082 1087 10871 3128 8080 2080 8888 10808; do
  has_word "$cands" "$p" || cands="$cands $p"
done

# URLs для проб: http-вариант каждого порта + уже настроенный прокси (если http/socks5).
urls=""
for p in $cands; do
  urls="$urls http://127.0.0.1:$p"
done
case "$CFG_PROXY" in
  http://*|socks5://*) has_word "$urls" "$CFG_PROXY" || urls="$urls $CFG_PROXY" ;;
esac

echo "=== ПРЯМОЕ СОЕДИНЕНИЕ (без прокси) ==="
d=$(curl -m 8 -s -o /dev/null -w '%{http_code}' "https://api.telegram.org/bot$TOKEN/getMe" 2>/dev/null)
[ -n "$d" ] || d=timeout
echo "DIRECT getMe: $d"

echo "=== ПРОБЫ ЛОКАЛЬНЫХ HTTP-ПРОКСИ ==="
# WORK — URL вернул 200 (токен валиден). REACH — достиг Telegram (любой HTTP-код
# кроме 000/timeout): прокси рабочий, но токен неверный/пустой. 000 = обрыв.
WORK=""
REACH=""
for u in $urls; do
  c=$(curl -m 8 -s -x "$u" -o /dev/null -w '%{http_code}' \
        "https://api.telegram.org/bot$TOKEN/getMe" 2>/dev/null)
  echo "$u -> ${c:-fail}"
  [ "$c" = "200" ] && WORK="$WORK $u"
  [ -n "$c" ] && [ "$c" != "000" ] && [ "$c" != "200" ] && REACH="$REACH $u=$c"
done

if [ -n "$WORK" ]; then
  u=$(echo "$WORK" | awk '{print $1}')
  echo
  echo ">>> ВПИШИТЕ В Прокси: $u"
  case "$u" in
    http://127.0.0.1:*) echo ">>> (mixed-inbound sing-box также принимает socks5://127.0.0.1:$(echo "$u" | sed 's#.*:##'))" ;;
  esac
  echo "=== ПОЛУЧЕНИЕ chat_id ==="
  up=$(curl -m 8 -s -x "$u" \
         "https://api.telegram.org/bot$TOKEN/getUpdates" 2>/dev/null)
  cid=$(echo "$up" | sed -n 's/.*"chat":{"id":\(-\{0,1\}[0-9][0-9]*\).*/\1/p' | head -1)
  if [ -n "$cid" ]; then
    echo ">>> chat_id = $cid"
  else
    echo ">>> chat_id не найден: напишите боту любое сообщение и повторите скрипт."
  fi
elif [ -n "$REACH" ]; then
  r=$(echo "$REACH" | awk '{print $1}')
  u="${r%=*}"
  code="${r#*=}"
  echo
  echo ">>> ПРОКСИ ДОСТУПЕН: $u (Telegram ответил кодом $code)."
  echo ">>> Значит обход провайдера работает! Впишите в Прокси: $u"
  case "$u" in
    http://127.0.0.1:*) echo ">>> (и socks5://127.0.0.1:$(echo "$u" | sed 's#.*:##') для mixed-inbound sing-box)." ;;
  esac
  echo ">>> Код $code = неверный/пустой токен. Укажите валидный bot_token и chat_id,"
  echo ">>> затем Сохранить -> Отправить тест (тест вернёт 200 при верном токене)."
  echo ">>> Другие доступные:"
  for r in $REACH; do
    echo ">>>   ${r%=*} (${r#*=})"
  done
else
  echo
  echo ">>> Рабочий HTTP-прокси не найден (все пробы дали 000/timeout)."
  echo ">>> 1) Оставьте поле Прокси ПУСТЫМ и пустите api.telegram.org через AWG-маршрут:"
  echo ">>>    Keenetic -> Политика маршрутизации -> 149.154.160.0/22, 91.108.4.0/22 -> awg-sys-Wireguard0"
  echo ">>> 2) Проверьте что sing-box запущен: tail /opt/etc/awg-manager/singbox/sing-box.log"
  echo ">>> НЕ используйте z2k tg-mtproxy (:1443/:1444) и rt-proxy (:1445) -- это MTProto, не HTTP/SOCKS."
fi