#!/bin/sh
# ==============================================
# Entware Manager - просмотр текстового файла
# GET: ?path=...
#   XHR (fetch) → JSON
#   Браузер    → HTML
# ==============================================

. /opt/web_entware/lib/common.sh

path=$(get_param "path" "")
is_xhr=0; [ -n "$HTTP_X_REQUESTED_WITH" ] && is_xhr=1

error_out() {
    local msg="$1"
    if [ "$is_xhr" -eq 1 ]; then
        json_out "{\"status\":\"error\",\"message\":\"$msg\"}"
    else
        html_header
        echo "<!DOCTYPE html><html><head><meta charset='UTF-8'><link rel='stylesheet' href='/entware-manager/style.css'></head><body class='packages-body'><div class='packages-container'><p class='error'>$(html_escape "$msg")</p><p style='margin-top:1rem;'><a href='javascript:history.back()' class='packages-delete-btn' style='background:#4a5568;'><svg class=\"icon\" width=\"14\" height=\"14\"><use href=\"/entware-manager/icons.svg?v=2#icon-arrow-left\"/></svg> Назад</a></p></div></body></html>"
        exit 0
    fi
}

case "$path" in
    /tmp/*|/dev/shm/*) ;;
    *) error_out "Доступ запрещен" ;;
esac

[ ! -f "$path" ] && error_out "Файл не найден"

size=$(wc -c < "$path" 2>/dev/null || echo 0)
[ "$size" -gt 1048576 ] && error_out "Файл слишком большой (макс. 1 MB)"

# Проверка на бинарный файл (null-байты в первых 4KB)
if command -v od >/dev/null 2>&1; then
    head -c 4096 "$path" 2>/dev/null | od -An -tx1 2>/dev/null | tr -d ' \n' | grep -q '00' && error_out "Невозможно отобразить бинарный файл"
fi

name=$(basename "$path")

# --- XHR: JSON ---
if [ "$is_xhr" -eq 1 ]; then
    name_esc=$(echo "$name" | sed 's/\\/\\\\/g; s/"/\\"/g')
    content=$(tail -n 1000 "$path" 2>/dev/null | /opt/bin/jq -Rs . 2>/dev/null)
    if [ -z "$content" ]; then
        content=$(tail -n 1000 "$path" 2>/dev/null | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g' | awk '{printf "%s\\n", $0}')
        content="\"$content\""
    fi
    echo "Content-type: application/json; charset=utf-8"
    echo ""
    printf '{"status":"ok","name":"%s","content":%s}\n' "$name_esc" "$content"
    exit 0
fi

# --- Браузер: HTML ---
file_content=$(tail -n 1000 "$path" 2>/dev/null)
html_header
cat <<VIEWHTML
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>$(html_escape "$name") — просмотр файла</title>
    <link rel="stylesheet" href="/entware-manager/style.css">
</head>
<body class="packages-body">
<div class="packages-container">
    <h2 style="display: flex; align-items: center; gap: 8px;">
        <span class="stat-icon" style="width: 28px; height: 28px;">
            <svg class="icon" width="28" height="28">
                <use href="/entware-manager/icons.svg?v=2#icon-file"/>
            </svg>
        </span>
        $(html_escape "$name")
    </h2>
    <pre class="file-viewer-content">$(html_escape "$file_content")</pre>
    <p style="margin-top: 1rem;"><a href="javascript:history.back()" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=2#icon-arrow-left"/></svg> Назад</a></p>
</div>
</body>
</html>
VIEWHTML
