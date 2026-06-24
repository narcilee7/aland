# Aland Makefile —— 一份够用的日常开发流
#
# Usage:
#   make              # 看帮助
#   make dev          # 启动 wails dev（前端 HMR + Go 热编译）
#   make build        # 生产构建 .app
#   make run          # 构建并启动
#   make check        # 类型检查
#
# 调试三件套：
#   make dev          # 热重载，前端用 wails 内置 devtools（Cmd+Opt+I）
#   make dev-debug    # 显式开 devtools 窗口
#   make stop         # 杀进程

# —— 变量 ——
APP_NAME      := aland
GO            := go
NPM           := npm
WAILS         := $(shell command -v wails 2>/dev/null || echo "$$HOME/go/bin/wails")
BUILD_DIR     := build/bin
APP_BUNDLE    := $(BUILD_DIR)/$(APP_NAME).app
APP_BIN       := $(APP_BUNDLE)/Contents/MacOS/$(APP_NAME)
PLATFORM      ?= darwin/universal

# —— 默认目标：help ——
.DEFAULT_GOAL := help

# —— 帮助 ——
.PHONY: help
help: ## 显示所有可用目标
	@awk 'BEGIN {FS = ":.*##"; printf "\n\033[1mAland Makefile\033[0m\n\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""

# —— 安装 / 依赖 ——
.PHONY: install
install: ## 安装所有依赖（wails CLI、Go modules、npm packages）
	@if ! command -v wails >/dev/null 2>&1 && [ ! -x $(WAILS) ]; then \
		echo "→ installing wails CLI..."; \
		$(GO) install github.com/wailsapp/wails/v2/cmd/wails@latest; \
	fi
	@echo "→ go mod tidy"
	$(GO) mod tidy
	@echo "→ npm install"
	cd frontend && $(NPM) install
	@echo "✓ dependencies ready"

.PHONY: deps
deps: install ## 同 install

# —— 开发 ——
.PHONY: dev
dev: ## 启动 wails dev（前端 HMR + Go 自动重编译）
	@if [ ! -x $(WAILS) ]; then \
		echo "❌ wails CLI not found. Run: make install"; \
		exit 1; \
	fi
	$(WAILS) dev

.PHONY: dev-trace
dev-trace: ## 启动 wails dev + Go 日志级别设为 Trace（最详细）
	$(WAILS) dev -loglevel Trace

.PHONY: dev-debug
dev-debug: ## 启动 wails dev + 详细日志
	$(WAILS) dev -loglevel Debug

# 在 dev 启动的 Aland 窗口里：右键 → Inspect Element 打开 webview devtools
# 不需要单独 flag

.PHONY: dev-debug
dev-debug: ## wails dev + 显式开 devtools 窗口
	$(WAILS) dev -devtools

# —— 构建 ——
.PHONY: build
build: ## 生产构建当前平台
	$(WAILS) build

.PHONY: build-debug
build-debug: ## 构建 debug 版本（含 devtools）
	$(WAILS) build -devtools

.PHONY: build-platform
build-platform: ## 跨平台构建（PLATFORM=windows/amd64 等）
	$(WAILS) build -platform $(PLATFORM)

.PHONY: build-skip-frontend
build-skip-frontend: ## 只构建 Go（前端不动）
	$(WAILS) build -s -skipfrontend

# —— 子模块 ——
.PHONY: backend
backend: ## 只编译 Go 后端
	$(GO) build ./...

.PHONY: backend-test
backend-test: ## 跑 Go 测试
	$(GO) test ./...

.PHONY: frontend
frontend: ## 只构建前端（tsc + vite build）
	cd frontend && $(NPM) run build

.PHONY: frontend-dev
frontend-dev: ## 单独跑 Vite dev（不开 wails 窗口，浏览器调试用）
	cd frontend && $(NPM) run dev

# —— 运行 ——
.PHONY: run
run: build ## 构建并启动 .app
	open $(APP_BUNDLE)

.PHONY: run-existing
run-existing: ## 直接启动已构建的 .app（不重新编译）
	@if [ ! -d $(APP_BUNDLE) ]; then \
		echo "❌ $(APP_BUNDLE) not found. Run: make build"; \
		exit 1; \
	fi
	open $(APP_BUNDLE)

.PHONY: stop
stop: ## 杀掉所有 aland 进程
	-pkill -f "$(APP_NAME).app/Contents/MacOS/$(APP_NAME)" 2>/dev/null || true
	@echo "✓ stopped"

# —— 检查 / 修复 ——
.PHONY: check
check: ## 类型检查（go vet + tsc）
	$(GO) vet ./...
	cd frontend && $(NPM) run build

.PHONY: vet
vet: ## Go 静态检查
	$(GO) vet ./...

.PHONY: fmt
fmt: ## 格式化代码（gofmt + goimports + prettier）
	$(GO) fmt ./...
	@command -v goimports >/dev/null 2>&1 && goimports -w . || echo "(goimports not installed, skipping)"
	@cd frontend && (command -v prettier >/dev/null 2>&1 && prettier --write "src/**/*.{ts,tsx}" || echo "(prettier not installed, skipping)")

.PHONY: tidy
tidy: ## 同步依赖（go mod tidy + npm install）
	$(GO) mod tidy
	cd frontend && $(NPM) install

# —— 清理 ——
.PHONY: clean
clean: ## 删除构建产物
	rm -rf $(BUILD_DIR)
	rm -rf frontend/dist
	@echo "✓ cleaned"

.PHONY: clean-all
clean-all: clean ## 完全清理（含 node_modules、wails 生成文件）
	rm -rf frontend/node_modules
	rm -rf frontend/wailsjs
	@echo "✓ fully cleaned"

.PHONY: reset
reset: clean-all install ## 完全重置：清理 + 重装依赖

# —— 诊断 ——
.PHONY: doctor
doctor: ## 跑 wails doctor 查环境
	$(WAILS) doctor

.PHONY: inspect
inspect: ## 查看构建产物结构
	@echo "App bundle:  $(APP_BUNDLE)"
	@echo "Binary:      $(APP_BIN)"
	@ls -la $(APP_BUNDLE)/Contents/MacOS/ 2>/dev/null || echo "(not built yet)"
	@echo ""
	@echo "Bundle size:"
	@du -sh $(APP_BUNDLE) 2>/dev/null || echo "(not built yet)"

.PHONY: bindings
bindings: ## 重新生成 wails JS 绑定（frontend/wailsjs/）
	$(WAILS) generate module

.PHONY: ports
ports: ## 查 wails dev 端口占用
	@lsof -nP -iTCP:34115 -sTCP:LISTEN 2>/dev/null || echo "(port 34115 free)"

.PHONY: tree
tree: ## 打印项目结构（排除 node_modules / build / git）
	@find . -type f \
		-not -path "*/node_modules/*" \
		-not -path "*/.git/*" \
		-not -path "*/build/bin/*" \
		-not -path "*/dist/*" \
		-not -path "*/.claude/*" \
		| sort
