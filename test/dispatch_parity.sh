#!/bin/sh
# shellcheck disable=SC3043,SC2086
# ==============================================
# Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
# Тест паритета диспетчеризации между режимами (миграция lighttpd -> go).
# Гарантирует: эндпоинт, работающий в go-режиме (cgi.go), обслуживается и в
# lighttpd-режиме (go.cgi + symlink из build-deploy.sh). Регрессия этого
# контракта = 404 при переходе lighttpd->go (см. TEST_PLAN §11).
# Запуск: sh test/dispatch_parity.sh   (подхватывается в make test)
# ==============================================
set -u

cd "$(dirname "$0")/.." || exit 1

PASS=0
FAIL=0
ok()  { PASS=$((PASS + 1)); echo "ok: $1"; }
bad() { FAIL=$((FAIL + 1)); echo "FAIL: $1"; }

# ---------- извлечение источников ----------
# flatDispatch (go-режим, cgi.go)
flat_disp=$(awk '/var flatDispatch = map\[string\]string\{/,/^\}/' go/internal/server/cgi.go \
  | grep -oE '"[a-z_]+":' | sed -E 's/[ ":]//g' | grep -v '^$' | sort -u)

# flat-маршруты go.cgi (lighttpd-режим): piped-списки + одиночные case
gocgi_piped=$(sed -n '100,124p' cgi-bin/go.cgi | grep -oE '[a-z_]+(\|[a-z_]+)+' | tr '|' '\n' | sort -u)
gocgi_single=$(sed -n '100,124p' cgi-bin/go.cgi | grep -oE '^[[:space:]]*[a-z_]+\)' | sed -E 's/[[:space:]]*//; s/\)//')
gocgi_flat=$(printf '%s\n%s\n' "$gocgi_piped" "$gocgi_single" | sort -u | grep -v '^$')

# подкаталоги: cgi.go subdirDispatch + go.cgi case-метки (4 известных, уникальны в коде)
sub_names=$(grep -oE '"(network|logger|monitor|service_watchdog)":' go/internal/server/cgi.go \
  | sed -E 's/[":]//g' | sort -u)
gocgi_sub=$(grep -oE '^[[:space:]]*(network|logger|monitor|service_watchdog)\)' cgi-bin/go.cgi \
  | sed -E 's/[[:space:]]*//; s/\)//' | sort -u)

# flat-список из build-deploy.sh (генерирует symlink-и)
bep=$(grep -E '^for ep in ' build-deploy.sh | head -1 | sed -E 's/for ep in //; s/; do//' \
  | tr ' ' '\n' | sort -u | grep -v '^$')
# подкаталожные списки build-deploy.sh (cd <subdir>; for ep in ...)
inner_build() {
  awk -v D="$1" '$0 ~ "cd[[:space:]].*" D "$" {c=1; next} \
    c && /^cd / {c=0} \
    c && /^for ep in / {sub(/for ep in /,""); sub(/; do/,""); gsub(/[ \t]+/,"\n"); print; next}' \
    build-deploy.sh | grep -v '^$' | sort -u
}

# ---------- A) паритет flat: cgi.go (go) == go.cgi (lighttpd) ----------
miss_disp=$(for x in $flat_disp; do echo "$gocgi_flat" | grep -qx "$x" || echo "$x"; done)
miss_gocgi=$(for x in $gocgi_flat; do echo "$flat_disp" | grep -qx "$x" || echo "$x"; done)
if [ -z "$miss_disp" ] && [ -z "$miss_gocgi" ]; then
  ok "A) flat: cgi.go(go) == go.cgi(lighttpd) — $(echo "$flat_disp" | wc -l) эндпоинтов"
else
  bad "A) flat-паритет нарушен: в go нет в lighttpd=[$miss_disp]; в lighttpd нет в go=[$miss_gocgi]"
fi

# ---------- B) build-deploy.sh генерирует symlink для каждого flatDispatch ----------
miss_bep=$(for x in $flat_disp; do echo "$bep" | grep -qx "$x" || echo "$x"; done)
if [ -z "$miss_bep" ]; then
  ok "B) build-deploy.sh покроет symlink-ами все $(echo "$flat_disp" | wc -l) flat-эндпоинтов"
else
  bad "B) build-deploy.sh НЕ сгенерирует symlink для: [$miss_bep] (будет 404 в lighttpd)"
fi

# ---------- C) паритет subdir: имена подкаталогов ----------
miss_sub=$(for x in $sub_names; do echo "$gocgi_sub" | grep -qx "$x" || echo "$x"; done)
if [ -z "$miss_sub" ]; then
  ok "C) subdir: cgi.go == go.cgi — подкаталоги [$(echo "$sub_names" | tr '\n' ' ')]"
else
  bad "C) subdir-паритет нарушен: [$miss_sub]"
fi

# ---------- D) каждый subdir покрыт go.cgi + build-deploy.sh (symlink-и) ----------
for d in $sub_names; do
  bi=$(inner_build "$d")
  if [ -n "$bi" ]; then
    ok "D) subdir '$d': go.cgi роутит + build-deploy.sh генерирует $(echo "$bi" | wc -l) symlink-ов"
  else
    bad "D) subdir '$d': build-deploy.sh НЕ генерирует symlink-и (будет 404 в lighttpd)"
  fi
done

# ---------- итог ----------
echo "========================================"
if [ "$FAIL" -eq 0 ]; then
  echo "PASS: $PASS  FAIL: $FAIL — паритет режимов lighttpd<->go цел"
else
  echo "PASS: $PASS  FAIL: $FAIL — ЕСТЬ РЕГРЕССИЯ МИГРАЦИИ"
fi
exit $FAIL
