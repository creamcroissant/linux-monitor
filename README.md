# Linux系统监控平台

<!--
这是一个全面的Linux系统监控平台，用于实时监控和管理多台Linux服务器的系统性能指标。
项目由三个主要部分组成：
1. 客户端代理：部署在被监控的Linux服务器上，负责采集系统指标
2. 服务端：接收和处理代理上报的数据，提供API和数据存储
3. 前端：提供用户友好的Web界面，展示监控数据和图表

该项目适用于系统管理员和开发团队，帮助他们实时了解服务器状态和性能瓶颈。
-->

一个基于Go和Vue.js的分布式Linux系统监控平台，支持多服务器实时监控、指标收集和可视化展示。

## 项目概述

Linux系统监控平台是一个轻量级、高性能的分布式监控解决方案，旨在帮助系统管理员和开发者实时监控多台Linux服务器的系统状态。该平台由服务端、客户端代理和Web前端组成，提供了丰富的功能和友好的用户界面。

### 主要功能

- **多服务器监控**：集中管理和监控多台Linux服务器
- **实时数据展示**：通过WebSocket实时获取服务器性能指标
- **历史数据查询**：查看服务器历史性能数据和趋势图表
- **指标可视化**：直观展示CPU、内存、磁盘、网络等系统资源使用情况
- **服务器状态管理**：监控服务器在线状态，管理服务器信息
- **用户认证与授权**：基于JWT的用户认证系统，支持多角色权限管理

## 系统架构

### 总体架构

系统采用经典的客户端/服务器架构，主要包含三个组件：

1. **服务端(Server)**：核心组件，负责数据收集、存储和API提供
2. **客户端代理(Agent)**：部署在被监控服务器上，收集系统指标并上报
3. **Web前端(Frontend)**：提供用户交互界面，展示监控数据

### 技术栈

- **后端**：
  - Go语言开发
  - Gin Web框架
  - SQLite数据库
  - WebSocket实时通信
  - JWT认证

- **前端**：
  - Vue 3框架
  - Element Plus UI组件库
  - ECharts图表库
  - Vuex状态管理
  - Vue Router路由管理

- **客户端代理**：
  - Go语言开发
  - gopsutil库采集系统指标
  - WebSocket安全通信

## 项目结构

本项目遵循模块化设计，主要由以下部分组成：

```
linux-monitor/
├── server/             # 服务端代码
│   ├── main.go         # 服务端入口文件(包含所有后端功能实现)
│   ├── go.mod          # Go模块依赖
│   ├── go.sum          # Go依赖校验文件
│   ├── config.json     # 服务端配置文件
│   ├── server          # 编译后的服务端可执行文件
│   ├── linux-monitor   # 编译后的服务端可执行文件(别名)
│   └── server_log.txt  # 服务端日志文件
│
├── agent/              # 客户端代理
│   ├── main.go         # 代理入口文件
│   └── go.mod          # Go模块依赖
│
├── frontend/           # Web前端
│   ├── src/            # 源代码
│   │   ├── api/        # API接口
│   │   ├── assets/     # 静态资源
│   │   ├── components/ # 公共组件
│   │   ├── router/     # 路由配置
│   │   ├── store/      # 状态管理
│   │   ├── utils/      # 工具函数
│   │   ├── views/      # 页面组件
│   │   ├── App.vue     # 应用入口组件
│   │   └── main.js     # 应用入口文件
│   │
│   ├── public/         # 公共静态资源
│   ├── index.html      # HTML模板
│   ├── vite.config.js  # Vite配置
│   └── package.json    # 依赖配置
│
├── linux-monitor.db    # SQLite数据库文件(主数据库)
├── config.json         # 全局配置文件
├── Makefile            # 项目构建和部署脚本
└── README.md           # 项目文档
```

## 组件详解

### 服务端(Server)

服务端是整个系统的核心，负责接收客户端代理上报的数据，提供API接口给前端调用，管理用户认证等功能。服务端采用单文件设计，所有功能都集成在`main.go`文件中，便于部署和维护。

**主要功能**：
- 接收和处理客户端代理上报的系统指标
- 提供RESTful API接口供前端调用
- 管理用户认证和授权
- 存储历史监控数据
- WebSocket实时通信

**关键模块**：
- HTTP服务器：基于Gin框架，提供REST API
- WebSocket服务：处理与代理的实时通信
- 数据库模块：使用SQLite存储数据，自动创建表结构
- 认证模块：JWT令牌生成和验证，用户权限管理
- 加密模块：AES加密确保数据传输安全

### 客户端代理(Agent)

客户端代理部署在每台被监控的Linux服务器上，负责收集系统指标并通过WebSocket上报给服务端。

**主要功能**：
- 收集系统性能指标(CPU、内存、磁盘、网络等)
- 定期上报数据到服务端
- 支持加密通信
- 支持断线重连

**关键模块**：
- 指标收集模块：使用gopsutil库采集系统数据
- 通信模块：WebSocket通信，支持加密和重连
- 配置管理：代理ID和服务器连接配置

### Web前端(Frontend)

Web前端提供用户友好的界面，展示服务器状态和性能指标，支持服务器管理、历史数据查询等功能。

**主要功能**：
- 服务器列表展示和管理
- 实时监控面板
- 历史数据查询和图表展示
- 用户登录和权限管理

**关键组件**：
- `Dashboard.vue`：显示所有服务器概览
- `AgentDetail.vue`：展示单个服务器的详细指标和图表
  - 包含多种资源类型的图表显示(CPU、内存、磁盘、负载等)
  - 支持实时数据更新和历史数据查询
  - 实现优化的图表渲染和清理机制
- `AgentManage.vue`：服务器管理界面
- `UserManage.vue`：用户管理界面
- `Login.vue`：用户登录界面

## 数据流

1. 客户端代理采集系统指标数据
2. 通过WebSocket上报给服务端(支持AES加密)
3. 服务端解密数据，存储到SQLite数据库并提供API接口
4. Web前端通过API获取数据并展示
5. 用户在前端进行操作，通过API与服务端交互
6. 前端使用ECharts库将数据可视化为多种图表

## 安装与部署

### 前置条件

- Go 1.20+
- Node.js 23.11.0+
- NPM 10.9.2+
- SQLite 3
- Linux/Unix环境(服务端和代理)

### 服务端部署

1. 克隆仓库
```bash
git clone https://github.com/creamcroissant/linux-monitor.git
cd linux-monitor
```

2. 编译服务端
```bash
cd server
go build -o linux-monitor-server
```

或使用项目根目录的Makefile：
```bash
make build-server
```

3. 配置服务端
编辑`config.json`文件：
```json
{
  "port": 8080,
  "db_path": "./linux-monitor.db",
  "encryption_key": "your-secret-key",
  "api_key": "your-api-key",
  "jwt_secret": "your-jwt-secret"
}
```

4. 运行服务端
```bash
./linux-monitor-server
```

或者使用命令行参数：
```bash
./linux-monitor-server -port 8080 -db ./linux-monitor.db -key your-encryption-key -apikey your-api-key
```

参数说明：
- `-port`：服务端监听端口，默认8080
- `-db`：SQLite数据库文件路径，默认为`./linux-monitor.db`
- `-key`：加密密钥，用于WebSocket通信加密和JWT生成
- `-apikey`：API密钥，用于服务端API认证
- `-config`：配置文件路径，默认为`./config.json`

### 客户端代理部署

1. 编译客户端代理
```bash
cd agent
go build -o linux-monitor-agent
```

或使用项目根目录的Makefile：
```bash
make build-agent
```

2. 运行客户端代理
```bash
./linux-monitor-agent -server ws://your-server-ip:8080/ws -interval 5 -key your-encryption-key
```

参数说明：
- `-server`：服务端WebSocket地址
- `-interval`：数据采集间隔(秒)
- `-key`：加密密钥，需与服务端一致

### 前端部署

1. 安装依赖
```bash
cd frontend
npm install
```

2. 开发模式运行
```bash
npm run dev
```

3. 生产环境构建
```bash
npm run build
```

或使用项目根目录的Makefile：
```bash
make build-frontend
```

4. 使用Nginx部署前端
将`frontend/dist`目录下的文件部署到Nginx服务器，参考配置：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    root /usr/share/nginx/html;  # 指向dist目录
    index index.html;

    # 静态文件缓存设置
    location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
        expires 30d;
        add_header Cache-Control "public, no-transform";
    }

    # 主应用路由
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API 代理
    location /api/ {
        proxy_pass http://localhost:8080/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 代理
    location /ws {
        proxy_pass http://localhost:8080/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }
}
```

## 使用指南

### 初始登录

默认管理员账户：
- 用户名：admin
- 密码：admin（首次登录后请立即修改）

### 添加被监控服务器

1. 在被监控服务器上部署并运行客户端代理
2. 代理会自动注册到服务端并开始上报数据
3. 在Web前端可以看到新添加的服务器

### 查看监控数据

1. 在仪表板页面可以看到所有服务器的状态概览
2. 点击单个服务器的详情按钮可查看详细监控指标
3. 在详情页可以查看历史数据图表，支持按时间范围筛选：
   - 过去1小时
   - 过去1天
   - 过去7天

### 图表功能

1. 系统提供多种资源类型的图表：
   - CPU使用率
   - 内存使用率
   - 磁盘使用率
   - 系统负载
   - 进程数量
   - 网络流量
   - 连接数

2. 图表特性：
   - 实时数据更新
   - 历史数据查询
   - 自适应时间轴
   - 数据点悬停提示
   - 响应式布局

### 管理服务器

在管理页面可以：
1. 编辑服务器名称和描述
2. 删除不再监控的服务器
3. 查看服务器详细信息

## 开发指南

### 添加新指标

#### 客户端代理修改

1. 在`agent/main.go`中的`collectMetrics`函数添加新指标采集逻辑
2. 在`Metrics`结构体中添加新字段

#### 服务端修改

1. 在`server/main.go`中更新数据库表结构
2. 更新WebSocket处理逻辑保存新指标

#### 前端修改

1. 在`frontend/src/views/AgentDetail.vue`中添加新的图表组件显示指标
2. 更新`createChartOption`函数以支持新的图表类型
3. 在标签页组件中添加新的标签页

### 自定义告警

本项目可扩展支持自定义告警功能：

1. 在服务端添加告警阈值设置和检测逻辑
2. 在前端添加告警配置界面
3. 实现邮件、短信或其他通知方式

## API参考

### 认证API

#### 登录

```
POST /api/login
```

**请求参数**：

```json
{
  "username": "admin",
  "password": "password"
}
```

**响应**：

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "username": "admin",
    "role": "admin"
  }
}
```

### 服务器API

#### 获取所有服务器

```
GET /api/agents
```

**响应**：

```json
{
  "data": [
    {
      "id": "server-id-1",
      "name": "Web服务器",
      "hostname": "web-server-1",
      "ip_address": "192.168.1.100",
      "platform": "linux",
      "is_online": true,
      "last_seen": "2023-05-10T15:20:30Z"
    },
    ...
  ]
}
```

#### 获取单个服务器详情

```
GET /api/agents/:id
```

**响应**：

```json
{
  "data": {
    "id": "server-id-1",
    "name": "Web服务器",
    "hostname": "web-server-1",
    "ip_address": "192.168.1.100",
    "platform": "linux",
    "is_online": true,
    "last_seen": "2023-05-10T15:20:30Z"
  }
}
```

#### 获取服务器指标数据

```
GET /api/agents/:id/metrics?from=1620000000&to=1620100000&limit=100
```

**响应**：

```json
{
  "data": [
    {
      "timestamp": 1620050000,
      "cpu_usage": 45.2,
      "memory_info": {
        "percent": 60.5,
        "total": 16777216,
        "used": 10160000
      },
      "disk_info": {
        "percent": 75.3,
        "total": 1073741824,
        "used": 805306368
      },
      "load_average": {
        "load1": 0.5,
        "load5": 0.7,
        "load15": 0.65
      },
      "network_info": {
        "bytes_sent": 1024000,
        "bytes_recv": 2048000,
        "tcp_connections": 120,
        "udp_connections": 30
      },
      "process_count": 210
    },
    ...
  ]
}
```

### 用户API

#### 获取所有用户 (仅管理员)

```
GET /api/admin/users
```

**响应**：

```json
{
  "data": [
    {
      "username": "admin",
      "role": "admin",
      "created_at": 1620050000
    },
    ...
  ]
}
```

#### 创建新用户 (仅管理员)

```
POST /api/admin/users
```

**请求参数**：

```json
{
  "username": "user1",
  "password": "password",
  "role": "user"
}
```

**响应**：

```json
{
  "message": "用户创建成功",
  "data": {
    "username": "user1",
    "role": "user"
  }
}
```

#### 删除用户 (仅管理员)

```
DELETE /api/admin/users/:username
```

**响应**：

```json
{
  "message": "用户删除成功"
}
```

## 系统优化

### 性能优化

1. **前端优化**：
   - 使用延迟加载(lazy loading)减少初始加载时间
   - 图表资源在切换标签页时才创建，减少资源占用
   - 图表实例在不使用时及时清理释放资源

2. **后端优化**：
   - 使用SQLite索引加速查询
   - 限制查询结果数量防止数据量过大
   - 定期清理过期历史数据

### 可靠性增强

1. **自动重连机制**：客户端代理断线自动重连
2. **数据库定期备份**：防止数据丢失
3. **错误处理和日志**：详细记录系统运行状态

## 常见问题解答

### 服务端无法启动

1. 检查端口是否被占用
2. 检查数据库文件权限
3. 确认配置文件格式正确

### 客户端代理连接失败

1. 确认服务端地址和端口正确
2. 检查加密密钥是否与服务端一致
3. 检查网络连接和防火墙设置

### 图表数据不显示

1. 确认客户端代理正常上报数据
2. 检查浏览器控制台是否有错误信息
3. 确认时间范围设置合理
4. 尝试清除浏览器缓存或使用隐私模式访问

## 贡献指南

欢迎为项目做出贡献！以下是参与本项目的方式：

1. Fork 本仓库
2. 创建您的功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交您的修改 (`git commit -m '添加一些很棒的功能'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开一个 Pull Request

## 版权和许可

本项目采用 MIT 许可证 - 详情请参阅 [LICENSE](LICENSE) 文件。

## 联系方式

如有问题或建议，请通过以下方式联系：

- GitHub Issues：https://github.com/creamcroissant/linux-monitor/issues

---

祝您使用愉快！ 