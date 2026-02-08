.PHONY: help install dev build clean docker-build docker-up docker-down init-data

help:
	@echo "起重机生产线程序管理系统 - 命令列表"
	@echo ""
	@echo "  make install       - 安装所有依赖"
	@echo "  make init-data     - 初始化系统数据"
	@echo "  make dev           - 启动开发环境"
	@echo "  make build         - 构建生产版本"
	@echo "  make clean         - 清理构建文件"
	@echo "  make docker-build  - 构建Docker镜像"
	@echo "  make docker-up     - 启动Docker容器"
	@echo "  make docker-down   - 停止Docker容器"

install:
	@echo "安装后端依赖..."
	cd backend && go mod download
	@echo "安装前端依赖..."
	cd frontend && npm install
	@echo "依赖安装完成!"

init-data:
	@echo "初始化系统数据..."
	cd backend && go run init_all.go
	@echo "数据�完成!"

dev:
	@echo "启动开发环境..."
	@echo "请在不同终端窗口运行:"
	@echo "  终端1: cd backend && go run main.go"
	@echo "  终端2: cd frontend && npm run dev"

build:
	@echo "构建后端..."
	cd backend && go build -o crane-system
	@echo "构建前端..."
	cd frontend && npm run build
	@echo "构建完成!"

clean:
	@echo "清理构建文件..."
	rm -f backend/crane-system
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	@echo "清理完成!"

docker-build:
	@echo "构建Docker镜像..."
	docker-compose build
	@echo "镜像构建完成!"

docker-up:
	@echo "启动Docker容器..."
	docker-compose up -d
	@echo "容器启动完成!"
	@echo "访问地址: http://localhost"

docker-down:
	@echo "停止Docker容器..."
	docker-compose down
	@echo "容器已停止!"

docker-logs:
	docker-compose logs -f
