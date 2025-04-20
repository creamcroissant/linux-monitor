# Linux系统监控平台 构建脚本
# 使用方法:
#   make all      # 构建客户端代理和服务端
#   make agent    # 仅构建代理
#   make server   # 构建服务端(含前端)
#   make frontend # 仅构建前端
#   make clean    # 清理构建产物
#   make run-server # 运行服务端
#   make run-agent  # 运行客户端代理

.PHONY: all clean agent server frontend

# 默认目标：构建所有组件
all: agent server

# 构建目录初始化
init:
	@echo "创建必要的目录结构..."
	mkdir -p bin

# 构建代理程序
agent: init
	@echo "构建客户端代理..."
	cd agent && go mod tidy && go build -o ../bin/linux-monitor-agent
	@echo "客户端代理构建完成: bin/linux-monitor-agent"

# 构建前端
frontend:
	@echo "安装前端依赖并构建..."
	cd frontend && npm install
	cd frontend && npm run build
	@echo "前端构建完成: frontend/dist/"

# 构建服务端 (包含前端)
server: init frontend
	@echo "复制前端文件到服务端目录..."
	mkdir -p server/dist
	rm -rf server/dist/*
	cp -r frontend/dist/* server/dist/
	@echo "前端文件已复制到 server/dist/"
	
	# 同时复制一份到根目录，方便直接运行服务器
	mkdir -p dist
	rm -rf dist/*
	cp -r frontend/dist/* dist/
	@echo "前端文件也已复制到根目录的 dist/ 目录"
	
	@echo "构建服务端程序..."
	cd server && go mod tidy && go build -o ../bin/linux-monitor-server
	@echo "服务端构建完成: bin/linux-monitor-server"

# 清理构建产物
clean:
	@echo "清理所有构建产物..."
	rm -rf bin
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	rm -rf server/dist
	rm -rf dist
	@echo "清理完成"

# 运行服务端
run-server:
	@echo "启动服务端程序..."
	./bin/linux-monitor-server -port 8080 -apikey "change-me-in-production"

# 运行客户端代理
run-agent:
	@echo "启动客户端代理程序..."
	./bin/linux-monitor-agent -server "ws://localhost:8080/ws" -key "default-encryption-key-change-me" 

# 开发模式：运行前端开发服务器
dev-frontend:
	@echo "启动前端开发服务器..."
	cd frontend && npm run dev

# 帮助信息
help:
	@echo "Linux系统监控平台构建脚本"
	@echo "可用命令:"
	@echo "  make all          - 构建所有组件(代理和服务端)"
	@echo "  make agent        - 仅构建代理程序"
	@echo "  make server       - 构建服务端(包含前端)"
	@echo "  make frontend     - 仅构建前端"
	@echo "  make clean        - 清理所有构建产物"
	@echo "  make run-server   - 运行服务端"
	@echo "  make run-agent    - 运行客户端代理"
	@echo "  make dev-frontend - 启动前端开发服务器"
	@echo "  make help         - 显示帮助信息" 