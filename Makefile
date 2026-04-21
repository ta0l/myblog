# 声明这些目标不是真实的文件名，防止与同名文件冲突
.PHONY: help install dev api web

# 默认执行的目标（直接输入 make 时触发）
help:
	@echo "======================================"
	@echo "  我的全栈博客开发工具箱"
	@echo "======================================"
	@echo "可用命令:"
	@echo "  make install  - 安装前后端所有依赖 (首次拉取代码后使用)"
	@echo "  make dev      - 🚀 同时启动前端和后端开发服务器"
	@echo "  make api      - 🟢 仅启动 Go 后端"
	@echo "  make web      - 🟠 仅启动 Astro 前端"

install:
	@echo "📦 正在安装依赖..."
	go mod tidy
	cd frontend && npm install

dev:
	@echo "🚀 正在启动全栈开发环境..."
	# 使用 -j 2 参数让 make 开启两个线程并发执行 api 和 web 任务
	@$(MAKE) -j 2 api web

api:
	@echo "🟢 启动 Go 后端 (运行于 http://localhost:8888)..."
	go run main.go

web:
	@echo "🟠 启动 Astro 前端 (运行于 http://localhost:4321)..."
	cd frontend && npm run dev