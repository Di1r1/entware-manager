#!/bin/sh
# ==============================================
# Поиск файлов в /tmp по имени (содержит заданную строку), исключая /tmp/mnt
# Версия: 1.1 (использование common.sh)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

echo "Content-type: application/json; charset=utf-8"
echo ""

QUERY_STRING="${QUERY_STRING:-}"
search=$(echo "$QUERY_STRING" | sed -n 's/.*q=\([^&]*\).*/\1/p' | sed 's/+/ /g; s/%/\\x/g')
search=$(printf '%b' "$search" 2>/dev/null)

if [ -z "$search" ]; then
    echo '[]'
    exit 0
fi

find /tmp -path "/tmp/mnt" -prune -o -type f -iname "*$search*" -print 2>/dev/null | while read file; do
    basename=$(basename "$file")
    name_esc=$(echo "$basename" | sed 's/\\/\\\\/g; s/"/\\"/g')
    path_esc=$(echo "$file" | sed 's/\\/\\\\/g; s/"/\\"/g')
    printf '{"name":"%s","path":"%s"}\n' "$name_esc" "$path_esc"
done | awk 'BEGIN { first=1; print "[" } { if (first) first=0; else print ","; print } END { print "]" }'
