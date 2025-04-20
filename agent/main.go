/*
linux-monitor 客户端代理

这是一个基于Go语言的Linux系统监控代理程序，部署在被监控的Linux服务器上，
负责采集系统性能指标并通过WebSocket连接上报给监控服务端。

主要功能：
- 采集系统CPU、内存、磁盘、网络等性能指标
- 收集系统基本信息（主机名、平台、内核版本等）
- 通过WebSocket上报数据到服务端
- 支持数据加密传输
- 自动重连机制

作者：Linux Monitor Team
版本：1.0.0
*/

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// Config 配置结构体，保存代理的配置信息
type Config struct {
	ServerURL     string // WebSocket服务器URL
	Interval      int    // 数据采集间隔（秒）
	EncryptionKey string // AES加密密钥
	AgentID       string // 代理唯一标识
}

// SystemMetrics 系统指标结构体，存储采集的系统性能数据
type SystemMetrics struct {
	AgentID        string                 `json:"agent_id"`        // 代理ID
	Timestamp      int64                  `json:"timestamp"`       // 时间戳
	CPUUsage       float64                `json:"cpu_usage"`       // CPU使用率
	MemoryInfo     map[string]interface{} `json:"memory_info"`     // 内存信息
	DiskInfo       map[string]interface{} `json:"disk_info"`       // 磁盘信息
	NetworkInfo    map[string]interface{} `json:"network_info"`    // 网络信息
	LoadAverage    map[string]interface{} `json:"load_average"`    // 负载平均值
	ProcessCount   int                    `json:"process_count"`   // 进程数量
	SystemInfo     map[string]interface{} `json:"system_info"`     // 系统信息
	UptimeSeconds  uint64                 `json:"uptime_seconds"`  // 系统运行时间(秒)
}

// 全局配置对象
var config Config

// 全局WebSocket连接和互斥锁
var wsConnection *websocket.Conn
var wsConnectionMutex = &sync.Mutex{}

// main 主函数，代理程序入口
func main() {
	// 解析命令行参数
	serverURL := flag.String("server", "ws://localhost:8080/ws", "WebSocket服务器URL")
	interval := flag.Int("interval", 5, "数据采集间隔（秒）")
	encryptionKey := flag.String("key", "default-encryption-key-change-me", "AES加密密钥")
	flag.Parse()

	// 设置全局配置
	config.ServerURL = *serverURL
	config.Interval = *interval
	config.EncryptionKey = *encryptionKey

	// 获取或生成代理ID
	agentID, err := getOrCreateAgentID()
	if err != nil {
		log.Fatalf("获取或创建代理ID失败: %v", err)
	}
	config.AgentID = agentID

	log.Printf("代理已启动，ID: %s", agentID)
	log.Printf("连接到服务器: %s", config.ServerURL)
	log.Printf("采集间隔: %d秒", config.Interval)

	// 启动主采集循环
	for {
		// 采集系统指标
		metrics, err := collectMetrics()
		if err != nil {
			log.Printf("采集指标出错: %v", err)
			time.Sleep(time.Duration(config.Interval) * time.Second)
			continue
		}

		// 发送指标到服务器
		err = sendMetrics(metrics)
		if err != nil {
			log.Printf("发送指标出错: %v", err)
		}

		// 等待下一个采集周期
		time.Sleep(time.Duration(config.Interval) * time.Second)
	}
}

// getOrCreateAgentID 从文件获取代理ID或创建新的ID
func getOrCreateAgentID() (string, error) {
	// 获取用户配置目录
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	// 创建代理配置目录
	agentDir := filepath.Join(configDir, "linux-monitor")
	err = os.MkdirAll(agentDir, 0755)
	if err != nil {
		return "", err
	}

	// ID文件路径
	idFilePath := filepath.Join(agentDir, "agent-id")
	
	// 检查文件是否存在
	if _, err := os.Stat(idFilePath); err == nil {
		// 从文件读取ID
		data, err := os.ReadFile(idFilePath)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	// 生成新的UUID
	newID := uuid.New().String()
	
	// 写入文件
	err = os.WriteFile(idFilePath, []byte(newID), 0644)
	if err != nil {
		return "", err
	}
	
	return newID, nil
}

// collectMetrics 采集系统性能指标
func collectMetrics() (SystemMetrics, error) {
	// 初始化指标结构体
	metrics := SystemMetrics{
		AgentID:     config.AgentID,
		Timestamp:   time.Now().Unix(),
		MemoryInfo:  make(map[string]interface{}),
		DiskInfo:    make(map[string]interface{}),
		NetworkInfo: make(map[string]interface{}),
		LoadAverage: make(map[string]interface{}),
		SystemInfo:  make(map[string]interface{}),
	}

	// 采集CPU使用率
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		metrics.CPUUsage = cpuPercent[0]
	}

	// 采集内存信息
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		metrics.MemoryInfo["total"] = memInfo.Total      // 总内存
		metrics.MemoryInfo["used"] = memInfo.Used        // 已用内存
		metrics.MemoryInfo["percent"] = memInfo.UsedPercent // 内存使用率
	}

	// 采集磁盘信息
	diskInfo, err := disk.Usage("/")
	if err == nil {
		metrics.DiskInfo["total"] = diskInfo.Total       // 总磁盘空间
		metrics.DiskInfo["used"] = diskInfo.Used         // 已用空间
		metrics.DiskInfo["percent"] = diskInfo.UsedPercent // 磁盘使用率
	}

	// 采集网络信息
	netIO, err := net.IOCounters(false)
	if err == nil && len(netIO) > 0 {
		metrics.NetworkInfo["bytes_sent"] = netIO[0].BytesSent // 发送字节数
		metrics.NetworkInfo["bytes_recv"] = netIO[0].BytesRecv // 接收字节数
	}

	// 采集TCP/UDP连接数
	tcpConns, _ := net.Connections("tcp")
	udpConns, _ := net.Connections("udp")
	metrics.NetworkInfo["tcp_connections"] = len(tcpConns) // TCP连接数
	metrics.NetworkInfo["udp_connections"] = len(udpConns) // UDP连接数

	// 采集负载平均值
	loadInfo, err := load.Avg()
	if err == nil {
		metrics.LoadAverage["load1"] = loadInfo.Load1   // 1分钟负载
		metrics.LoadAverage["load5"] = loadInfo.Load5   // 5分钟负载
		metrics.LoadAverage["load15"] = loadInfo.Load15 // 15分钟负载
	}

	// 采集进程数
	processes, _ := process.Processes()
	metrics.ProcessCount = len(processes)

	// 采集系统信息
	hostInfo, err := host.Info()
	if err == nil {
		metrics.SystemInfo["hostname"] = hostInfo.Hostname       // 主机名
		metrics.SystemInfo["os"] = hostInfo.OS                   // 操作系统
		metrics.SystemInfo["platform"] = hostInfo.Platform       // 系统平台
		metrics.SystemInfo["kernel_version"] = hostInfo.KernelVersion // 内核版本
		metrics.UptimeSeconds = hostInfo.Uptime                  // 系统运行时间
	}

	return metrics, nil
}

// encrypt 使用AES加密数据
func encrypt(data []byte, key string) ([]byte, error) {
	log.Printf("加密数据，长度: %d字节", len(data))
	log.Printf("使用加密密钥（前6个字符）: %s...", key[:min(6, len(key))])
	
	// 将密钥转换为32字节（AES-256）
	keyBytes := []byte(key)
	if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	} else if len(keyBytes) < 32 {
		// 如果密钥太短，进行填充
		newKey := make([]byte, 32)
		copy(newKey, keyBytes)
		keyBytes = newKey
	}

	// 创建加密器
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("创建加密器失败: %v", err)
	}

	// 创建随机IV
	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("生成IV失败: %v", err)
	}

	// 加密数据
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], data)
	
	log.Printf("加密后数据长度: %d字节", len(ciphertext))

	return ciphertext, nil
}

// min 返回a和b中较小的值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sendMetrics sends the collected metrics to the server
func sendMetrics(metrics SystemMetrics) error {
	// Convert metrics to JSON
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %v", err)
	}

	// Encrypt data
	encryptedData, err := encrypt(data, config.EncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt metrics: %v", err)
	}

	// Use a persistent WebSocket connection
	static_conn := getWebSocketConnection()
	if static_conn == nil {
		return fmt.Errorf("could not get WebSocket connection")
	}

	// Send data
	err = static_conn.WriteMessage(websocket.BinaryMessage, encryptedData)
	if err != nil {
		// Connection might be broken, reset it
		resetWebSocketConnection()
		return fmt.Errorf("failed to send metrics: %v", err)
	}

	return nil
}

// getWebSocketConnection returns an existing connection or creates a new one
func getWebSocketConnection() *websocket.Conn {
	wsConnectionMutex.Lock()
	defer wsConnectionMutex.Unlock()
	
	// If we already have a connection, check if it's still valid
	if wsConnection != nil {
		// Send a ping to check connection
		err := wsConnection.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(2*time.Second))
		if err == nil {
			return wsConnection
		}
		// Connection is broken, close it
		wsConnection.Close()
		wsConnection = nil
		log.Println("WebSocket connection lost, reconnecting...")
	}
	
	// Create a new connection
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second
	conn, _, err := dialer.Dial(config.ServerURL, nil)
	if err != nil {
		log.Printf("Failed to connect to server: %v", err)
		return nil
	}
	
	// Setup ping handler to keep connection alive
	conn.SetPingHandler(func(data string) error {
		err := conn.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(5*time.Second))
		if err != nil {
			log.Printf("Error sending pong: %v", err)
		}
		return nil
	})
	
	wsConnection = conn
	log.Println("Connected to server via WebSocket")
	return wsConnection
}

// resetWebSocketConnection closes and resets the WebSocket connection
func resetWebSocketConnection() {
	wsConnectionMutex.Lock()
	defer wsConnectionMutex.Unlock()
	
	if wsConnection != nil {
		wsConnection.Close()
		wsConnection = nil
	}
} 