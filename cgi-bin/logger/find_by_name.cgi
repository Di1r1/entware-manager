#!/bin/sh
# ==============================================
# Поиск файлов в /tmp по имени (содержит заданную строку), исключая /tmp/mnt
# Версия: 1.1 (использование common.sh)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

QUERY_STRING="${QUERY_STRING:-}"
search=$(echo "$QUERY_STRING" | sed -n 's/.*q=\([^&]*\).*/\1/p')
search=$(url_decode "$search")

if [ -z "$search" ]; then
    json_out '[]'
fi

result=$(find /tmp -path "/tmp/mnt" -prune -o -type f -iname "*$search*" -print 2>/dev/null | while read file; do
    basename=$(basename "$file")
    name_esc=$(echo "$basename" | sed 's/\\/\\\\/g; s/"/\\"/g')
    path_esc=$(echo "$file" | sed 's/\\/\\\\/g; s/"/\\"/g')
    printf '{"name":"%s","path":"%s"}\n' "$name_esc" "$path_esc"
done | awk 'BEGIN { first=1; print "[" } { if (first) first=0; else print ","; print } END { print "]" }')
json_out "$result"
