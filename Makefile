SHELL := /bin/bash
.PHONY: all deploy ipk release clean version check test lint ci archives help install-router

ARCHS := arm64 mips mipsel
MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

VERSION := $(shell git describe --tags --match 'v*' --abbrev=0 2>/dev/null | sed 's/^v//')
ifeq ($(VERSION),)
  VERSION := $(shell jq -r '.version' $(MAKEFILE_DIR)/version.json 2>/dev/null || python3 -c "import json; print(json.load(open('$(MAKEFILE_DIR)/version.json'))['version'])" 2>/dev/null || grep -o '"version"[^,]*' $(MAKEFILE_DIR)/version.json | cut -d'"' -f4 || echo "unknown")
endif

.DEFAULT_GOAL := help

all: deploy ipk
	@echo "=== Готово: deploy + ipk для всех архитектур ==="

deploy:
	@echo "=== Сборка deploy v$(VERSION) ==="
	@$(MAKEFILE_DIR)/build-deploy.sh --tar

deploy-%:
	@echo "=== Сборка deploy для $* ==="
	@$(MAKEFILE_DIR)/build-deploy.sh --arch=$*

ipk: deploy
	@echo "=== Сборка ipk v$(VERSION) ==="
	@$(MAKEFILE_DIR)/build-ipk.sh

ipk-%: deploy-%
	@echo "=== Сборка ipk для $* ==="
	@$(MAKEFILE_DIR)/build-ipk.sh --arch=$*

release: clean all archives
	@echo "============================================"
	@echo " Релиз v$(VERSION) собран"
	@echo " Файлы:"
	@ls -lh $(MAKEFILE_DIR)/dist/ 2>/dev/null || echo "  (нет ipk/tar.gz)"
	@echo "============================================"

clean:
	@echo "=== Очистка ==="
	rm -rf "$(MAKEFILE_DIR)/deploy" "$(MAKEFILE_DIR)/dist"
	@echo "✓ Очищено"

version:
	@echo "$(VERSION)"

archives: deploy
	@echo "=== Сборка per-arch tar.gz ==="
	@for arch in $(ARCHS); do \
		rm -rf "/tmp/deploy" && \
		cp -a "$(MAKEFILE_DIR)/deploy" "/tmp/deploy" && \
		find "/tmp/deploy/cgi-bin/go" -mindepth 1 -maxdepth 1 -type d ! -name "$$arch" -exec rm -rf {} + && \
		mkdir -p "$(MAKEFILE_DIR)/dist" && tar -czf "$(MAKEFILE_DIR)/dist/entware-manager-$$arch.tar.gz" -C /tmp "deploy" && \
		echo "  ✓ dist/entware-manager-$$arch.tar.gz"; \
	done

check:
	@echo "=== Проверка зависимостей ==="
	@ok=true; \
	for cmd in go tar find chmod; do \
		if command -v $$cmd &>/dev/null; then \
			echo "  [✓] $$cmd"; \
		else \
			echo "  [✗] $$cmd — требуется, но не найден"; \
			ok=false; \
		fi; \
	done; \
	if command -v jq &>/dev/null; then \
		echo "  [✓] jq"; \
	elif command -v python3 &>/dev/null; then \
		echo "  [✓] python3 (замена jq)"; \
	else \
		echo "  [✗] jq или python3 — требуется для чтения version.json"; \
		ok=false; \
	fi; \
	if command -v upx &>/dev/null || [ -x /tmp/upx-4.2.4-amd64_linux/upx ]; then \
		echo "  [✓] upx (опционально)"; \
	else \
		echo "  [ ] upx не найден (бинарники без сжатия)"; \
	fi; \
	$$ok

test:
	@echo "=== Go test ==="
	@cd "$(MAKEFILE_DIR)/go" && go test ./... 2>&1
	@echo "=== Shell syntax (sh -n) ==="
	@for f in $(MAKEFILE_DIR)/*.sh $(MAKEFILE_DIR)/Install/*.sh $(MAKEFILE_DIR)/lib/*.sh $(MAKEFILE_DIR)/logger/scripts/*.sh; do \
		if head -1 "$$f" 2>/dev/null | grep -q '#!/bin/bash\|#!/usr/bin/env bash'; then \
			echo "  [–] $$(basename "$$f") (bash, пропущен)"; continue; fi; \
		sh -n "$$f" 2>&1 && echo "  [✓] $$(basename "$$f")" || echo "  [✗] $$f"; \
	done
	@echo "=== Shell unit tests ==="
	@sh "$(MAKEFILE_DIR)/test/migrate_tests.sh" 2>&1

lint:
	@echo "=== Go vet ==="
	@cd "$(MAKEFILE_DIR)/go" && go vet ./... 2>&1
	@echo "  [✓] go vet пройден"
	@if command -v shellcheck &>/dev/null; then \
		echo "=== ShellCheck ==="; \
		shellcheck --severity=warning $(MAKEFILE_DIR)/*.sh $(MAKEFILE_DIR)/Install/*.sh $(MAKEFILE_DIR)/lib/*.sh $(MAKEFILE_DIR)/logger/scripts/*.sh 2>&1 && \
		echo "  [✓] ShellCheck пройден"; \
	else \
		echo "  [ ] shellcheck не найден (пропущено)"; \
	fi
	@if command -v checkbashisms &>/dev/null; then \
		echo "=== checkbashisms ==="; \
		checkbashisms $(MAKEFILE_DIR)/Install/*.sh $(MAKEFILE_DIR)/lib/*.sh $(MAKEFILE_DIR)/logger/scripts/*.sh 2>&1 && \
		echo "  [✓] checkbashisms пройден"; \
	else \
		echo "  [ ] checkbashisms не найден (пропущено)"; \
	fi

ci: check lint test
	@echo "=== CI пройден ==="

ROOT_HOST ?= 192.168.3.1
ROOT_PORT ?= 222
ROOT_DIR  ?= /opt/tmp

install-router:
	@if [ ! -d "$(MAKEFILE_DIR)/deploy" ]; then \
		echo "Ошибка: deploy/ не найден. Сначала сделай make deploy"; \
		exit 1; \
	fi
	@echo "=== Копирование deploy/ тар+ssh на root@$(ROOT_HOST):$(ROOT_PORT) ==="
	@tar czf - -C "$(MAKEFILE_DIR)" deploy | ssh -p $(ROOT_PORT) -o StrictHostKeyChecking=no root@$(ROOT_HOST) "tar xzf - -C $(ROOT_DIR)"
	@echo "=== Установка на роутере ==="
	@ssh -p $(ROOT_PORT) -o StrictHostKeyChecking=no root@$(ROOT_HOST) "cd $(ROOT_DIR)/deploy && sh Install/install.sh"

help:
	@echo "Entware Manager — сборка"
	@echo ""
	@echo "Цели:"
	@echo "  all            deploy + ipk для всех архитектур"
	@echo "  deploy         сборка deploy/ (Go + файлы)"
	@echo "  ipk            сборка ipk (зависит от deploy)"
	@echo "  archives       per-arch tar.gz из deploy/"
	@echo "  release        clean → deploy → ipk → archives"
	@echo "  clean          удалить deploy/ и dist/"
	@echo "  version        показать версию"
	@echo "  check          проверка инструментов"
	@echo "  test           go test ./..."
	@echo "  lint           go vet + shellcheck"
	@echo "  ci             check → lint → test"
	@echo "  install-router собрать и установить на роутер"
	@echo ""
	@echo "Для одной arch:  make deploy-arm64 ipk-arm64"
	@echo ""
	@echo "Версия: $(VERSION) | Архитектуры: $(ARCHS)"
