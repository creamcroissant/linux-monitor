# Linux监控平台 - 客户端代理

这是Linux系统监控平台的客户端代理程序，部署在被监控的Linux服务器上，负责采集系统性能指标并上报给服务端。

## 功能特性

- **系统指标采集**：收集CPU、内存、磁盘、网络等性能指标
- **自动注册**：首次运行时自动向服务端注册
- **WebSocket通信**：通过WebSocket实时上报数据
- **数据加密**：支持AES加密传输保证安全性
- **断线重连**：网络异常时自动重连
- **轻量高效**：资源占用低，对被监控系统影响小

## 系统需求

- Go 1.16+
- Linux操作系统(支持大多数主流发行版)
- 网络连接(能够访问监控服务端)

## 部署说明

### 编译

```bash
# 安装依赖并编译
go mod tidy
go build -o linux-monitor-agent
```

或者使用项目根目录的Makefile:

```bash
cd ..
make agent
```

### 运行

```bash
./linux-monitor-agent -server "ws://your-server-ip:8080/ws" -interval 5 -key "your-encryption-key"
```

参数说明:
- `-server`: 服务端WebSocket地址
- `-interval`: 数据采集间隔(秒)，默认为5秒
- `-key`: 加密密钥，需与服务端保持一致

### 设置为系统服务

创建systemd服务文件 `/etc/systemd/system/linux-monitor-agent.service`:

```ini
[Unit]
Description=Linux Monitor Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=/path/to/linux-monitor-agent -server "ws://your-server-ip:8080/ws" -interval 5 -key "your-encryption-key"
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启用并运行服务:

```bash
systemctl daemon-reload
systemctl enable linux-monitor-agent
systemctl start linux-monitor-agent
```

检查服务状态:

```bash
systemctl status linux-monitor-agent
```

## 代码结构

代理程序是单文件结构，所有功能都在`main.go`中实现:

- **主程序入口**: 初始化配置和启动主循环
- **指标采集**: 使用gopsutil库收集系统性能数据
- **WebSocket通信**: 负责与服务端的通信
- **加密模块**: 实现数据加密传输

## 自定义开发

### 添加新的指标采集

修改`collectMetrics`函数，添加新的指标:

```go
func collectMetrics() (SystemMetrics, error) {
    // ... existing code ...
    
    // 添加新的指标采集
    metrics.NewCustomMetric = collectCustomMetric()
    
    return metrics, nil
}

func collectCustomMetric() float64 {
    // 实现新指标的采集逻辑
    return value
}
```

然后需要在`SystemMetrics`结构体中添加对应的字段:

```go
type SystemMetrics struct {
    // ... existing fields ...
    NewCustomMetric float64 `json:"new_custom_metric"`
}
```

### 修改WebSocket通信

如需修改通信逻辑，可以调整`sendMetrics`和`getWebSocketConnection`函数:

```go
func sendMetrics(metrics SystemMetrics) error {
    // 自定义通信逻辑
}
```

## 故障排除

### 代理无法连接到服务端

1. 检查服务端地址是否正确
2. 确认服务端是否已启动
3. 检查网络连接和防火墙配置
4. 验证加密密钥是否与服务端一致

### 资源使用率异常

如果代理程序占用过多资源:

1. 增加数据采集间隔(-interval参数)
2. 检查网络连接质量
3. 减少采集的指标类型(修改代码)

## 日志说明

代理程序的日志输出到标准输出(stdout)，可以通过以下方式重定向:

```bash
./linux-monitor-agent ... > agent.log 2>&1
```

或者在systemd服务中配置日志:

```ini
[Service]
...
StandardOutput=append:/var/log/linux-monitor-agent.log
StandardError=append:/var/log/linux-monitor-agent.log
``` 