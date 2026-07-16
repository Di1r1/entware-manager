#!/bin/sh
# ==============================================
# Entware Manager - просмотр текстового файла
# GET: ?path=... → JSON {status,name,content}
# ==============================================

. /opt/web_entware/lib/common.sh

path=$(get_param "path" "")

case "$path" in
    /tmp/*|/dev/shm/*) ;;
    *) json_out '{"status":"error","message":"Доступ запрещен"}' ;;
esac

[ ! -f "$path" ] && json_out '{"status":"error","message":"Файл не найден"}'

size=$(wc -c < "$path" 2>/dev/null || echo 0)
if [ "$size" -gt 1048576 ]; then
    json_out '{"status":"error","message":"Файл слишком большой (макс. 1 MB)"}'
fi

# Проверка на бинарный файл (null-байты в первых 4KB)
if head -c 4096 "$path" 2>/dev/null | od -An -tx1 2>/dev/null | tr -d ' \n' | grep -q '00'; then
    json_out '{"status":"error","message":"Невозможно отобразить бинарный файл"}'
fi

name=$(basename "$path")
name_esc=$(echo "$name" | sed 's/\\/\\\\/g; s/"/\\"/g')
content=$(tail -n 1000 "$path" 2>/dev/null | /opt/bin/jq -Rs . 2>/dev/null)

if [ -z "$content" ]; then
    content=$(tail -n 1000 "$path" 2>/dev/null | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g' | awk '{printf "%s\\n", $0}')
    content="\"$content\""
fi

echo "Content-type: application/json; charset=utf-8"
echo ""
printf '{"status":"ok","name":"%s","content":%s}\n' "$name_esc" "$content"
