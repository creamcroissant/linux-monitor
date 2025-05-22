/*
linux-monitor 服务端

这是一个基于Go语言的Linux系统监控平台服务端，使用Gin框架提供REST API和WebSocket服务，
用于接收客户端代理上报的系统指标，存储历史数据，并向前端提供数据服务。

主要功能：
- 接收和处理客户端代理通过WebSocket上报的系统指标
- 提供RESTful API接口给前端调用
- 用户认证和授权管理
- 数据存储和历史查询

作者：Linux Monitor Team
版本：1.0.0
*/

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// Config 配置结构体，用于保存服务端配置
type Config struct {
	Port          int    `json:"port"`           // HTTP服务器监听端口
	DBPath        string `json:"db_path"`        // SQLite数据库文件路径
	EncryptionKey string `json:"encryption_key"` // AES加密密钥
	APIKey        string `json:"api_key"`        // API认证密钥
	JWTSecret     string `json:"jwt_secret"`     // JWT密钥
}

// SystemMetrics 系统指标结构体，用于存储从客户端代理接收的监控数据
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

// Agent 代理信息结构体，用于存储代理服务器的基本信息
type Agent struct {
	ID        string    `json:"id"`         // 代理唯一标识
	Name      string    `json:"name"`       // 代理名称
	LastSeen  time.Time `json:"last_seen"`  // 最后一次活动时间
	IsOnline  bool      `json:"is_online"`  // 是否在线
	Hostname  string    `json:"hostname"`   // 主机名
	Platform  string    `json:"platform"`   // 操作系统平台
	IPAddress string    `json:"ip_address"` // IP地址
	CreatedAt time.Time `json:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}

// User 用户信息结构体，用于存储用户认证和权限信息
type User struct {
	Username    string `json:"username"`     // 用户名
	Password    string `json:"password_hash"` // 密码哈希，存储的是加密后的密码
	Role        string `json:"role"`         // 角色，admin或user
	CreatedAt   int64  `json:"created_at"`   // 创建时间
}

// 全局变量
var (
	config   Config                        // 全局配置对象
	db       *sql.DB                       // 数据库连接
	clients  = make(map[string]*websocket.Conn) // WebSocket客户端连接映射表，键为代理ID
	upgrader = websocket.Upgrader{        // WebSocket升级器
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许任何来源的连接请求
		},
	}
	// agent异常状态缓存
	offlineAlerted = make(map[string]bool) // 离线告警缓存
	highLoadStart = make(map[string]int64) // 高负载起始时间缓存
	highLoadAlerted = make(map[string]bool) // 高负载告警缓存
)

// Claims JWT令牌的声明结构体
type Claims struct {
	Username string `json:"username"` // 用户名
	Role     string `json:"role"`     // 角色
	jwt.RegisteredClaims              // JWT标准声明
}

// webhook结构体
type Webhook struct {
	Type    string `json:"type"` // serverchan/custom
	Name    string `json:"name"`
	SendKey string `json:"sendkey,omitempty"`
	URL     string `json:"url,omitempty"`
	Enabled bool   `json:"enabled"`
}

// main 主函数，服务端程序入口
func main() {
	// 命令行参数定义
	configFile := flag.String("config", "./config.json", "配置文件路径")
	port := flag.Int("port", 0, "HTTP服务器端口（覆盖配置文件）")
	dbPath := flag.String("db", "", "SQLite数据库路径（覆盖配置文件）")
	encryptionKey := flag.String("key", "", "AES加密密钥（覆盖配置文件）")
	apiKey := flag.String("apikey", "", "API认证密钥（覆盖配置文件）")
	flag.Parse()

	// 加载配置文件
	var err error
	config, err = loadConfig(*configFile)
	if err != nil {
		log.Printf("加载配置文件警告：%v，将使用默认值或命令行参数", err)
	}

	// 命令行参数优先级高于配置文件
	if *port != 0 {
		config.Port = *port
	}
	if *dbPath != "" {
		config.DBPath = *dbPath
	}
	if *encryptionKey != "" {
		config.EncryptionKey = *encryptionKey
	}
	if *apiKey != "" {
		config.APIKey = *apiKey
	}

	// 初始化数据库
	err = initDB()
	if err != nil {
		log.Fatalf("初始化数据库失败：%v", err)
	}
	defer db.Close()

	// 启动时自动生成hostname.json（如不存在）
	hostnameFile := "hostname.json"
	if _, err := os.Stat(hostnameFile); os.IsNotExist(err) {
		data, _ := json.MarshalIndent(map[string]string{}, "", "  ")
		_ = ioutil.WriteFile(hostnameFile, data, 0644)
	}

	// 启动时自动生成webhook.json（如不存在）
	webhookFile := "webhook.json"
	if _, err := os.Stat(webhookFile); os.IsNotExist(err) {
		_ = ioutil.WriteFile(webhookFile, []byte("[]"), 0644)
	}

	// 设置Gin路由
	r := gin.Default()

	// 认证路由（无中间件）
	r.POST("/api/login", login)          // 用户登录
	r.POST("/api/register", register)     // 用户注册

	// 公共API路由（只读）
	publicApi := r.Group("/api")
	{
		publicApi.GET("/agents", getAgents)               // 获取所有代理列表
		publicApi.GET("/agents/:id", getAgentByID)        // 获取指定代理详情
		publicApi.GET("/agents/:id/metrics", getAgentMetrics) // 获取指定代理的监控指标
	}

	// 受保护的API路由（写操作）
	protectedApi := r.Group("/api")
	protectedApi.Use(apiKeyMiddleware())
	{
		protectedApi.PUT("/agents/:id", updateAgent)      // 更新代理信息
		protectedApi.DELETE("/agents/:id", deleteAgent)   // 删除代理
	}

	// 安全API路由（JWT或ApiKey）
	secureApi := r.Group("/api")
	secureApi.Use(authMiddleware())
	{
		secureApi.GET("/users/me", getCurrentUser)        // 获取当前用户信息
		secureApi.PUT("/users/password", updatePassword)  // 更新密码
	}

	// 管理员路由
	adminApi := r.Group("/api/admin")
	adminApi.Use(adminMiddleware())
	{
		adminApi.GET("/users", getUsers)                  // 获取所有用户列表
		adminApi.POST("/users", createUser)               // 创建新用户
		adminApi.DELETE("/users/:username", deleteUser)   // 删除用户
	}

	// 新增webhook API路由
	r.GET("/api/webhook", getWebhook) // 获取webhook配置
	r.PUT("/api/webhook", adminMiddleware(), setWebhook) // 设置webhook配置
	r.POST("/api/webhook/test", adminMiddleware(), testWebhook) // 测试webhook

	// WebSocket处理器
	r.GET("/ws", handleWebSocket) // WebSocket连接处理
	
	// 添加基本首页
	r.GET("/", func(c *gin.Context) {
		// 重定向到前端应用
		c.File("./dist/index.html")
	})

	// 为前端Vue应用提供静态文件服务
	r.Static("/assets", "./dist/assets")
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.File("./dist/favicon.ico")
	})
	
	// 确保在所有路由之后，处理所有未匹配的路由
	r.NoRoute(func(c *gin.Context) {
		// 检查请求路径是否为静态资源
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/assets/") {
			// 先尝试发送相对于dist目录的资源文件
			assetPath := "./dist" + path
			if _, err := os.Stat(assetPath); err == nil {
				c.File(assetPath)
				return
			}
			log.Printf("找不到静态资源: %s", assetPath)
		}
		
		// 所有其他路由都返回index.html
		c.File("./dist/index.html")
	})

	// 启动HTTP服务器
	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("服务器启动，端口：%d", config.Port)
	log.Fatal(r.Run(addr))

	// 启动自动告警任务
	go alertTask()
}

// Initialize the SQLite database
func initDB() error {
	// Ensure the database directory exists
	dbDir := filepath.Dir(config.DBPath)
	if dbDir != "." && dbDir != "/" {
		err := os.MkdirAll(dbDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create database directory: %v", err)
		}
	}

	// Open database connection
	var err error
	db, err = sql.Open("sqlite3", config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	// 检查users表是否存在last_login列
	columns, err := getTableColumns("users")
	if err != nil {
		log.Printf("检查users表结构错误: %v", err)
	} else {
		hasLastLogin := false
		for _, column := range columns {
			if column == "last_login" {
				hasLastLogin = true
				break
			}
		}
		
		// 如果存在last_login列，进行迁移
		if hasLastLogin {
			log.Println("检测到last_login列，开始迁移users表...")
			
			// 使用事务进行迁移
			tx, err := db.Begin()
			if err != nil {
				log.Printf("开始迁移事务失败: %v", err)
			} else {
				// 1. 创建临时表，无last_login列
				_, err = tx.Exec(`
					CREATE TABLE users_temp (
						username TEXT PRIMARY KEY, 
						password TEXT NOT NULL,
						role TEXT DEFAULT 'user',
						created_at INTEGER
					)
				`)
				if err != nil {
					tx.Rollback()
					log.Printf("创建临时表失败: %v", err)
				} else {
					// 2. 复制数据到临时表
					_, err = tx.Exec(`INSERT INTO users_temp SELECT username, password, role, created_at FROM users`)
					if err != nil {
						tx.Rollback()
						log.Printf("复制用户数据失败: %v", err)
					} else {
						// 3. 删除原表
						_, err = tx.Exec(`DROP TABLE users`)
						if err != nil {
							tx.Rollback()
							log.Printf("删除原用户表失败: %v", err)
						} else {
							// 4. 重命名临时表
							_, err = tx.Exec(`ALTER TABLE users_temp RENAME TO users`)
							if err != nil {
								tx.Rollback()
								log.Printf("重命名临时表失败: %v", err)
							} else {
								// 提交事务
								err = tx.Commit()
								if err != nil {
									log.Printf("提交迁移事务失败: %v", err)
								} else {
									log.Println("成功迁移users表，移除last_login列")
								}
							}
						}
					}
				}
			}
		}
	}

	// Create tables if they don't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			name TEXT,
			last_seen INTEGER,
			hostname TEXT,
			platform TEXT,
			ip_address TEXT
		);

		CREATE TABLE IF NOT EXISTS metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT,
			timestamp INTEGER,
			cpu_usage REAL,
			memory_total INTEGER,
			memory_used INTEGER,
			memory_percent REAL,
			disk_total INTEGER,
			disk_used INTEGER,
			disk_percent REAL,
			network_sent INTEGER,
			network_recv INTEGER,
			load_avg_1 REAL,
			load_avg_5 REAL,
			load_avg_15 REAL,
			process_count INTEGER,
			FOREIGN KEY(agent_id) REFERENCES agents(id)
		);

		CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY, 
			password TEXT NOT NULL,
			role TEXT DEFAULT 'user',
			created_at INTEGER
		);

		CREATE INDEX IF NOT EXISTS idx_metrics_agent_timestamp ON metrics(agent_id, timestamp);
	`)
	if err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	// Check and update agents table structure if necessary
	log.Println("检查agents表结构...")
	columns, err = getTableColumns("agents")
	if err != nil {
		return fmt.Errorf("failed to check agents table columns: %v", err)
	}

	// 检查是否存在created_at和updated_at列
	hasCreatedAt := false
	hasUpdatedAt := false
	for _, column := range columns {
		if column == "created_at" {
			hasCreatedAt = true
		}
		if column == "updated_at" {
			hasUpdatedAt = true
		}
	}

	// 添加缺少的列
	if !hasCreatedAt {
		_, err = db.Exec("ALTER TABLE agents ADD COLUMN created_at INTEGER DEFAULT 0")
		if err != nil {
			log.Printf("Warning: Could not add created_at column: %v", err)
		} else {
			log.Println("已添加 created_at 列到 agents 表")
		}
	}

	if !hasUpdatedAt {
		_, err = db.Exec("ALTER TABLE agents ADD COLUMN updated_at INTEGER DEFAULT 0")
		if err != nil {
			log.Printf("Warning: Could not add updated_at column: %v", err)
		} else {
			log.Println("已添加 updated_at 列到 agents 表")
		}
	}

	// 更新创建时间为0的记录
	_, err = db.Exec("UPDATE agents SET created_at = ? WHERE created_at IS NULL OR created_at = 0", time.Now().Unix())
	if err != nil {
		log.Printf("Warning: Could not update agents created_at: %v", err)
	}

	// Create default admin user if no users exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check users: %v", err)
	}

	if count == 0 {
		// 生成密码哈希
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("无法为默认管理员用户生成密码哈希：%v", err)
		}
		
		// Create default admin user with hashed password
		_, err = db.Exec(
			"INSERT INTO users (username, password, role, created_at) VALUES (?, ?, ?, ?)",
			"admin", string(hashedPassword), "admin", time.Now().Unix(),
		)
		if err != nil {
			return fmt.Errorf("failed to create default admin user: %v", err)
		}
		log.Println("创建默认管理员用户（用户名：admin，密码：admin）")
	}

	// Create a background task to clean up old data
	go cleanupTask()

	return nil
}

// 获取表的列名
func getTableColumns(tableName string) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid, notnull, pk int
		var name, dataType string
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &dataType, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

// cleanupTask removes old metrics and updates agent statuses
func cleanupTask() {
	for {
		// Delete metrics older than 7 days
		_, err := db.Exec("DELETE FROM metrics WHERE timestamp < ?", time.Now().Unix()-7*24*60*60)
		if err != nil {
			log.Printf("Error cleaning up old metrics: %v", err)
		}

		// Just remove old metrics, don't change agent status
		// Agents will be considered offline if last_seen is older than 30 seconds
		// but we don't need to modify the last_seen value

		time.Sleep(1 * time.Hour)
	}
}

// decrypt decrypts data using AES
func decrypt(data []byte, key string) ([]byte, error) {
	if len(data) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short: received only %d bytes, need at least %d bytes", len(data), aes.BlockSize)
	}

	// 打印解密信息
	log.Printf("Trying to decrypt message of length: %d bytes", len(data))
	
	// Convert key to 32 bytes for AES-256
	keyBytes := []byte(key)
	if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	} else if len(keyBytes) < 32 {
		// Pad key if too short
		newKey := make([]byte, 32)
		copy(newKey, keyBytes)
		keyBytes = newKey
	}
	
	log.Printf("Using encryption key (first 6 chars): %s...", key[:min(6, len(key))])

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	// Get IV from first block
	iv := data[:aes.BlockSize]
	ciphertext := data[aes.BlockSize:]

	// Decrypt
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	
	// Log the first few bytes of decrypted data
	if len(ciphertext) > 20 {
		log.Printf("Decrypted data starts with: %s", string(ciphertext[:20]))
	}

	return ciphertext, nil
}

// min returns the smaller of a or b
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleWebSocket handles WebSocket connections from agents
func handleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()
	
	// 增加缓冲区大小
	conn.SetReadLimit(65536)
	
	// Set initial values
	var agentID string
	remoteAddr := c.Request.RemoteAddr
	
	// 设置处理Ping消息
	conn.SetPingHandler(func(message string) error {
		log.Printf("Received ping from agent, sending pong")
		err := conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(5*time.Second))
		if err != nil {
			log.Printf("Error sending pong: %v", err)
		}
		return nil
	})
	
	// 设置处理Pong消息
	conn.SetPongHandler(func(message string) error {
		log.Printf("Received pong from agent")
		return nil
	})
	
	// 最长允许连接断开的时间
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	
	// Start a goroutine to send periodic pings
	stopPinger := make(chan struct{})
	heartbeatTicker := time.NewTicker(10 * time.Second)
	defer heartbeatTicker.Stop()
	
	go func() {
		for {
			select {
			case <-heartbeatTicker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
					log.Printf("Error sending ping to agent %s: %v", agentID, err)
					select {
					case stopPinger <- struct{}{}:
					default:
					}
					return
				}
				log.Printf("Sent ping to agent %s", agentID)
			case <-stopPinger:
				return
			}
		}
	}()

	// Log connection details
	log.Printf("WebSocket connection established from %s", remoteAddr)

	// Handle messages
	for {
		// Reset read deadline with each message
		if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			log.Printf("Error setting read deadline: %v", err)
			break
		}

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		// Process the message based on its type
		switch messageType {
		case websocket.TextMessage:
			handleAgentMessage(conn, message, &agentID, remoteAddr)
		case websocket.BinaryMessage:
			log.Printf("Received binary message from %s", remoteAddr)
			handleAgentMessage(conn, message, &agentID, remoteAddr)
		default:
			log.Printf("Received message of type %d from %s", messageType, remoteAddr)
		}
	}

	// 连接关闭，记录断开状态
	log.Printf("WebSocket connection closed for agent %s (%s)", agentID, remoteAddr)
	
	// 如果有agent ID，从客户端映射中移除
	if agentID != "" {
		delete(clients, agentID)
		// 记录agent断开连接的时间
		log.Printf("Agent %s disconnected", agentID)
	}
}

// handleAgentMessage processes messages received from agents
func handleAgentMessage(conn *websocket.Conn, message []byte, agentID *string, remoteAddr string) {
	// 记录接收到的消息
	if *agentID != "" {
		log.Printf("Received data from agent %s, message length: %d bytes", *agentID, len(message))
	} else {
		log.Printf("Received data from %s, message length: %d bytes", remoteAddr, len(message))
	}
	
	// 如果agentID已经存在（说明这是来自已知agent的消息），直接更新last_seen
	if *agentID != "" {
		// 获取当前时间戳
		now := time.Now().Unix()
		// 只更新last_seen时间
		_, err := db.Exec("UPDATE agents SET last_seen = ? WHERE id = ?", now, *agentID)
		if err != nil {
			log.Printf("Failed to update agent last_seen: %v", err)
		} else {
			log.Printf("Updated last_seen for agent %s to %d", *agentID, now)
		}
	}
	
	// 如果是二进制消息，需要先解密
	if len(message) > 0 {
		var metrics SystemMetrics
		
		// 尝试解析JSON
		err := json.Unmarshal(message, &metrics)
		if err != nil {
			log.Printf("Raw data is not valid JSON: %v, attempting to decrypt", err)
			// 如果解析JSON失败，可能是加密数据，尝试解密
			decrypted, err := decrypt(message, config.EncryptionKey)
			if err != nil {
				log.Printf("Failed to decrypt message: %v", err)
				return
			}
			
			log.Printf("Successfully decrypted message, length: %d bytes", len(decrypted))
			
			// 再次尝试解析解密后的JSON
			err = json.Unmarshal(decrypted, &metrics)
			if err != nil {
				log.Printf("Failed to parse metrics JSON after decryption: %v", err)
				// 打印解密后的数据前20字节用于调试
				if len(decrypted) > 20 {
					log.Printf("First 20 bytes of decrypted data: %v", decrypted[:20])
				}
				return
			}
			log.Printf("Successfully parsed metrics JSON from decrypted data")
		} else {
			log.Printf("Successfully parsed metrics JSON directly from message")
		}
		
		// 打印收到的metrics数据摘要
		log.Printf("Received metrics - CPU: %.2f%%, Mem: %.2f%%, Disk: %.2f%%", 
			metrics.CPUUsage,
			getMemoryPercent(metrics.MemoryInfo),
			getDiskPercent(metrics.DiskInfo))
		
		// 设置或更新agentID
		if metrics.AgentID != "" {
			*agentID = metrics.AgentID
			
			// 保存连接到客户端映射
			clients[*agentID] = conn
			
			log.Printf("Agent identified: %s", *agentID)
			
			// 更新agent在数据库中的信息
			updateAgentInfo(*agentID, metrics, remoteAddr)
			
			// 存储指标到数据库
			err = storeMetrics(metrics)
			if err != nil {
				log.Printf("Failed to store metrics: %v", err)
			} else {
				log.Printf("Successfully stored metrics for agent %s", *agentID)
				// 检查数据库中是否实际存储了数据
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM metrics WHERE agent_id = ?", *agentID).Scan(&count)
				if err != nil {
					log.Printf("Failed to check metrics count: %v", err)
				} else {
					log.Printf("Total metrics count for agent %s in database: %d", *agentID, count)
				}
			}
		} else {
			log.Printf("Received metrics without agent ID from %s", remoteAddr)
		}
	} else {
		log.Printf("Received empty message from %s", remoteAddr)
	}
}

// 辅助函数，从内存信息map中获取percent值
func getMemoryPercent(memoryInfo map[string]interface{}) float64 {
	if memoryInfo == nil {
		return 0
	}
	
	if percent, ok := memoryInfo["percent"].(float64); ok {
		return percent
	}
	return 0
}

// 辅助函数，从磁盘信息map中获取percent值
func getDiskPercent(diskInfo map[string]interface{}) float64 {
	if diskInfo == nil {
		return 0
	}
	
	if percent, ok := diskInfo["percent"].(float64); ok {
		return percent
	}
	return 0
}

// updateAgentInfo updates agent information in the database
func updateAgentInfo(agentID string, metrics SystemMetrics, remoteAddr string) {
	// 获取当前时间戳
	now := time.Now().Unix()

	// 1. 读取hostname.json
	hostnameOverride := ""
	hostnameMap := map[string]string{}
	hostnameFile := "hostname.json"
	if data, err := ioutil.ReadFile(hostnameFile); err == nil {
		_ = json.Unmarshal(data, &hostnameMap)
		if v, ok := hostnameMap[agentID]; ok && v != "" {
			hostnameOverride = v
		}
	}

	// 从系统信息中提取主机名和平台信息
	var hostname, platform string
	if metrics.SystemInfo != nil {
		if h, ok := metrics.SystemInfo["hostname"].(string); ok {
			hostname = h
		}
		if p, ok := metrics.SystemInfo["platform"].(string); ok {
			platform = p
		}
	}
	// 优先使用hostname.json中的主机名
	if hostnameOverride != "" {
		hostname = hostnameOverride
	}
	
	// 如果主机名或平台为空，设置默认值
	if hostname == "" {
		hostname = "unknown-host"
	}
	if platform == "" {
		platform = "Unknown"
	}
	
	// 获取IP地址（去除端口部分）
	ipAddress := remoteAddr
	if colonIndex := strings.LastIndex(ipAddress, ":"); colonIndex != -1 {
		ipAddress = ipAddress[:colonIndex]
	}
	
	// 先检查表结构
	hasCreatedAt := false
	hasUpdatedAt := false
	
	columns, err := getTableColumns("agents")
	if err != nil {
		log.Printf("检查agents表结构错误: %v", err)
	} else {
		for _, column := range columns {
			if column == "created_at" {
				hasCreatedAt = true
			}
			if column == "updated_at" {
				hasUpdatedAt = true
			}
		}
	}
	
	// 检查agent是否已存在
	var exists bool
	err = db.QueryRow("SELECT 1 FROM agents WHERE id = ?", agentID).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error checking agent existence: %v", err)
		return
	}
	
	if err == sql.ErrNoRows {
		// Agent不存在，插入新记录
		var insertQuery string
		var insertArgs []interface{}
		
		if hasCreatedAt && hasUpdatedAt {
			insertQuery = "INSERT INTO agents (id, name, last_seen, hostname, platform, ip_address, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
			insertArgs = []interface{}{agentID, hostname, now, hostname, platform, ipAddress, now, now}
		} else if hasCreatedAt {
			insertQuery = "INSERT INTO agents (id, name, last_seen, hostname, platform, ip_address, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)"
			insertArgs = []interface{}{agentID, hostname, now, hostname, platform, ipAddress, now}
		} else if hasUpdatedAt {
			insertQuery = "INSERT INTO agents (id, name, last_seen, hostname, platform, ip_address, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)"
			insertArgs = []interface{}{agentID, hostname, now, hostname, platform, ipAddress, now}
		} else {
			insertQuery = "INSERT INTO agents (id, name, last_seen, hostname, platform, ip_address) VALUES (?, ?, ?, ?, ?, ?)"
			insertArgs = []interface{}{agentID, hostname, now, hostname, platform, ipAddress}
		}
		
		_, err = db.Exec(insertQuery, insertArgs...)
		if err != nil {
			log.Printf("Failed to insert new agent %s: %v", agentID, err)
		} else {
			log.Printf("New agent registered: %s (hostname: %s, platform: %s)", agentID, hostname, platform)
		}
	} else {
		// Agent存在，更新记录
		var updateQuery string
		var updateArgs []interface{}
		
		if hasUpdatedAt {
			updateQuery = "UPDATE agents SET last_seen = ?, name = COALESCE(name, ?), hostname = ?, platform = ?, ip_address = ?, updated_at = ? WHERE id = ?"
			updateArgs = []interface{}{now, hostname, hostname, platform, ipAddress, now, agentID}
		} else {
			updateQuery = "UPDATE agents SET last_seen = ?, name = COALESCE(name, ?), hostname = ?, platform = ?, ip_address = ? WHERE id = ?"
			updateArgs = []interface{}{now, hostname, hostname, platform, ipAddress, agentID}
		}
		
		_, err = db.Exec(updateQuery, updateArgs...)
		if err != nil {
			log.Printf("Failed to update agent %s: %v", agentID, err)
		} else {
			log.Printf("Updated agent info: %s", agentID)
		}
	}
}

// 存储代理上报的指标数据
func storeMetrics(metrics SystemMetrics) error {
	if metrics.AgentID == "" {
		log.Printf("收到的指标数据为空，无法存储")
		return nil
	}

	// 获取时间戳
	timestamp := metrics.Timestamp
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}
	
	log.Printf("存储代理 %s 的指标数据，时间戳: %d", metrics.AgentID, timestamp)
	
	// 提取CPU使用率
	cpuUsage := metrics.CPUUsage
	
	// 提取内存信息
	var memTotal, memUsed int64
	var memPercent float64
	if metrics.MemoryInfo != nil {
		if total, ok := metrics.MemoryInfo["total"].(float64); ok {
			memTotal = int64(total)
		}
		if used, ok := metrics.MemoryInfo["used"].(float64); ok {
			memUsed = int64(used)
		}
		if percent, ok := metrics.MemoryInfo["percent"].(float64); ok {
			memPercent = percent
		}
	}
	
	// 提取磁盘信息
	var diskTotal, diskUsed int64
	var diskPercent float64
	if metrics.DiskInfo != nil {
		if total, ok := metrics.DiskInfo["total"].(float64); ok {
			diskTotal = int64(total)
		}
		if used, ok := metrics.DiskInfo["used"].(float64); ok {
			diskUsed = int64(used)
		}
		if percent, ok := metrics.DiskInfo["percent"].(float64); ok {
			diskPercent = percent
		}
	}
	
	// 提取网络信息
	var netSent, netRecv int64
	if metrics.NetworkInfo != nil {
		if sent, ok := metrics.NetworkInfo["bytes_sent"].(float64); ok {
			netSent = int64(sent)
		}
		if recv, ok := metrics.NetworkInfo["bytes_recv"].(float64); ok {
			netRecv = int64(recv)
		}
	}
	
	// 提取负载信息
	var loadAvg1, loadAvg5, loadAvg15 float64
	if metrics.LoadAverage != nil {
		if l1, ok := metrics.LoadAverage["load1"].(float64); ok {
			loadAvg1 = l1
		}
		if l5, ok := metrics.LoadAverage["load5"].(float64); ok {
			loadAvg5 = l5
		}
		if l15, ok := metrics.LoadAverage["load15"].(float64); ok {
			loadAvg15 = l15
		}
	}
	
	// 提取进程数
	processCount := metrics.ProcessCount
	
	// 打印提取到的主要指标信息
	log.Printf("存储指标: agent=%s, CPU=%.2f%%, 内存=%.2f%%, 磁盘=%.2f%%, 进程数=%d",
		metrics.AgentID, cpuUsage, memPercent, diskPercent, processCount)
	
	// 创建metrics表（如果不存在）
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			cpu_usage REAL,
			memory_total INTEGER,
			memory_used INTEGER,
			memory_percent REAL,
			disk_total INTEGER,
			disk_used INTEGER,
			disk_percent REAL,
			network_sent INTEGER,
			network_recv INTEGER,
			load_avg_1 REAL,
			load_avg_5 REAL,
			load_avg_15 REAL,
			process_count INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	
	if err != nil {
		log.Printf("创建metrics表失败: %v", err)
		return err
	}
	
	// 创建索引（如果不存在）
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_metrics_agent_time ON metrics (agent_id, timestamp)
	`)
	
	if err != nil {
		log.Printf("创建索引失败: %v", err)
		// 继续执行，不返回
	}
	
	// 插入指标数据
	stmt, err := db.Prepare(`
		INSERT INTO metrics (
			agent_id, timestamp, 
			cpu_usage, 
			memory_total, memory_used, memory_percent,
			disk_total, disk_used, disk_percent,
			network_sent, network_recv,
			load_avg_1, load_avg_5, load_avg_15,
			process_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	
	if err != nil {
		log.Printf("准备插入指标语句失败: %v", err)
		return err
	}
	defer stmt.Close()
	
	_, err = stmt.Exec(
		metrics.AgentID, timestamp,
		cpuUsage,
		memTotal, memUsed, memPercent,
		diskTotal, diskUsed, diskPercent,
		netSent, netRecv,
		loadAvg1, loadAvg5, loadAvg15,
		processCount,
	)
	
	if err != nil {
		log.Printf("插入指标数据失败: %v", err)
		return err
	}
	
	log.Printf("成功存储代理 %s 的指标数据", metrics.AgentID)
	
	// 清理旧数据（保留30天内的数据）
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Unix()
	_, err = db.Exec("DELETE FROM metrics WHERE timestamp < ?", thirtyDaysAgo)
	if err != nil {
		log.Printf("清理旧指标数据失败: %v", err)
	} else {
		log.Printf("已清理30天前的旧指标数据")
	}

	return nil
}

// 创建JWT令牌
func createToken(username, role string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

// API中间件：验证API密钥
func apiKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 首先检查Authorization头部（Bearer令牌）
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(config.JWTSecret), nil
			})

			if err == nil && token.Valid {
				if claims, ok := token.Claims.(*Claims); ok {
					c.Set("username", claims.Username)
					c.Set("role", claims.Role)
					c.Next()
					return
				}
			}
		}

		// 然后检查X-API-Key头部
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == config.APIKey {
			// API密钥有效
			c.Set("role", "admin") // API密钥授予管理员权限
			c.Next()
			return
		}

		// 如果没有有效的令牌或API密钥，检查是否有例外路由
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		c.Abort()
	}
}

// 身份验证中间件：验证JWT令牌
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 首先尝试解析JWT令牌
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(config.JWTSecret), nil
			})

			if err == nil && token.Valid {
				if claims, ok := token.Claims.(*Claims); ok {
					c.Set("username", claims.Username)
					c.Set("role", claims.Role)
					c.Next()
					return
				}
			}
		}

		// 然后检查API密钥
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == config.APIKey {
			c.Set("role", "admin")
			c.Next()
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		c.Abort()
	}
}

// 管理员权限中间件
func adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 验证身份
		role, exists := c.Get("role")
		if !exists {
			// 先验证JWT
			authHeader := c.GetHeader("Authorization")
			log.Printf("管理员API请求，Authorization: %s", authHeader)
			
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
					return []byte(config.JWTSecret), nil
				})

				if err != nil {
					log.Printf("JWT解析错误: %v", err)
				} else if token.Valid {
					if claims, ok := token.Claims.(*Claims); ok {
						role = claims.Role
						c.Set("role", role)
						log.Printf("验证成功，用户: %s, 角色: %s", claims.Username, claims.Role)
						exists = true
					}
				}
			}

			// 如果JWT验证失败，检查API密钥
			if !exists {
				apiKey := c.GetHeader("X-API-Key")
				if apiKey == config.APIKey {
					role = "admin"
					c.Set("role", role)
					log.Printf("API密钥验证成功")
					exists = true
				}
			}
		}

		// 检查是否有管理员权限
		if exists && role == "admin" {
			log.Printf("管理员权限验证通过: %v", role)
			c.Next()
			return
		}

		log.Printf("管理员权限验证失败，角色: %v, 存在: %v", role, exists)
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		c.Abort()
	}
}

// 登录处理函数
func login(c *gin.Context) {
	var credentials struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}

	// 查询用户
	var user User
	err := db.QueryRow("SELECT username, password, role FROM users WHERE username = ?", credentials.Username).Scan(&user.Username, &user.Password, &user.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户名或密码"})
		} else {
			log.Printf("数据库查询错误: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		}
		return
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password))
	if err == nil { // 密码正确
		// 创建JWT令牌
		token, err := createToken(user.Username, user.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建令牌"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"username": user.Username,
				"role":     user.Role,
			},
		})
	} else {
		log.Printf("密码验证失败: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户名或密码"})
	}
}

// 注册处理函数
func register(c *gin.Context) {
	var credentials struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}

	// 检查用户名是否已存在
	var exists bool
	err := db.QueryRow("SELECT 1 FROM users WHERE username = ?", credentials.Username).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("数据库查询错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	if err == nil { // 用户已存在
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(credentials.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("密码哈希错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	// 插入新用户
	_, err = db.Exec(
		"INSERT INTO users (username, password, role, created_at) VALUES (?, ?, ?, ?)",
		credentials.Username, string(hashedPassword), "user", time.Now().Unix(),
	)
	if err != nil {
		log.Printf("创建用户错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建用户"})
		return
	}

	// 创建JWT令牌
	token, err := createToken(credentials.Username, "user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建令牌"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user": gin.H{
			"username": credentials.Username,
			"role":     "user",
		},
	})
}

// 获取当前用户信息
func getCurrentUser(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var user User
	err := db.QueryRow("SELECT username, role, created_at FROM users WHERE username = ?", username).Scan(&user.Username, &user.Role, &user.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取用户信息"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username":  user.Username,
		"role":      user.Role,
		"createdAt": user.CreatedAt,
	})
}

// 更新密码
func updatePassword(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var passwordData struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&passwordData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}

	// 验证当前密码
	var storedPassword string
	err := db.QueryRow("SELECT password FROM users WHERE username = ?", username).Scan(&storedPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(passwordData.CurrentPassword))
	if err != nil { // 密码不正确
		c.JSON(http.StatusUnauthorized, gin.H{"error": "当前密码不正确"})
		return
	}

	// 哈希新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordData.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("密码哈希错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	// 更新密码
	_, err = db.Exec("UPDATE users SET password = ? WHERE username = ?", string(hashedPassword), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新密码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码已更新"})
}

// 获取所有用户（仅管理员）
func getUsers(c *gin.Context) {
	log.Printf("请求获取用户列表")
	
	rows, err := db.Query("SELECT username, role, created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		log.Printf("获取用户列表查询错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取用户列表"})
		return
	}
	defer rows.Close()

	var users []gin.H
	for rows.Next() {
		var username, role string
		var createdAt int64
		
		if err := rows.Scan(&username, &role, &createdAt); err != nil {
			log.Printf("扫描用户数据错误: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "处理用户数据错误"})
			return
		}
		
		users = append(users, gin.H{
			"username":  username,
			"role":      role,
			"createdAt": createdAt,
		})
	}
	
	if err := rows.Err(); err != nil {
		log.Printf("遍历用户数据错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理用户数据错误"})
		return
	}

	log.Printf("成功获取用户列表，共 %d 个用户", len(users))
	c.JSON(http.StatusOK, users)
}

// 创建用户（仅管理员）
func createUser(c *gin.Context) {
	var userData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}

	// 检查角色有效性
	if userData.Role != "admin" && userData.Role != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色必须是 'admin' 或 'user'"})
		return
	}

	// 检查用户名是否已存在
	var exists bool
	err := db.QueryRow("SELECT 1 FROM users WHERE username = ?", userData.Username).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	if err == nil { // 用户已存在
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("密码哈希错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		log.Printf("创建事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	// 创建用户 - 移除last_login字段
	_, err = tx.Exec(
		"INSERT INTO users (username, password, role, created_at) VALUES (?, ?, ?, ?)",
		userData.Username, string(hashedPassword), userData.Role, time.Now().Unix(),
	)
	if err != nil {
		tx.Rollback()
		log.Printf("创建用户失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		log.Printf("提交事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	log.Printf("成功创建用户: %s, 角色: %s", userData.Username, userData.Role)
	c.JSON(http.StatusCreated, gin.H{
		"username":  userData.Username,
		"role":      userData.Role,
		"createdAt": time.Now().Unix(),
	})
}

// 删除用户（仅管理员）
func deleteUser(c *gin.Context) {
	username := c.Param("username")

	// 检查用户是否存在
	var exists bool
	err := db.QueryRow("SELECT 1 FROM users WHERE username = ?", username).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		} else {
			log.Printf("检查用户存在性错误: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		}
		return
	}

	// 防止删除管理员账户
	if username == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能删除系统管理员账户"})
		return
	}

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		log.Printf("创建事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	// 删除用户
	result, err := tx.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		tx.Rollback()
		log.Printf("删除用户错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败"})
		return
	}

	// 检查是否实际删除了用户
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		log.Printf("获取受影响行数错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	if rowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在或已被删除"})
		return
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		log.Printf("提交事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	log.Printf("成功删除用户: %s", username)
	c.JSON(http.StatusOK, gin.H{"message": "用户已删除"})
}

// 获取所有代理
func getAgents(c *gin.Context) {
	log.Printf("API call: %s %s", c.Request.Method, c.Request.URL.Path)
	
	// 执行查询获取所有代理
	query := "SELECT id, name, hostname, platform, ip_address, last_seen, COALESCE(created_at, 0) as created_at, COALESCE(updated_at, 0) as updated_at FROM agents ORDER BY created_at DESC"
	
	log.Printf("执行查询: %s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("数据库查询错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取代理列表失败", "detail": "数据库查询出现错误", "message": err.Error()})
		return
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var agent Agent
		var lastSeenUnix sql.NullInt64
		var createdAtUnix, updatedAtUnix sql.NullInt64
		
		err := rows.Scan(
			&agent.ID, 
			&agent.Name, 
			&agent.Hostname, 
			&agent.Platform, 
			&agent.IPAddress, 
			&lastSeenUnix, 
			&createdAtUnix, 
			&updatedAtUnix)
		
		if err != nil {
			log.Printf("数据行扫描错误: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解析代理数据失败", "detail": "处理数据库结果时出错", "message": err.Error()})
			return
		}
		
		// 处理可能为空的时间字段
		if lastSeenUnix.Valid {
			agent.LastSeen = time.Unix(lastSeenUnix.Int64, 0)
			// 检查是否在线 - 如果最后一次通信在10秒内则视为在线
			agent.IsOnline = time.Since(agent.LastSeen) < 10*time.Second
		} else {
			agent.LastSeen = time.Time{}
			agent.IsOnline = false
		}
		
		if createdAtUnix.Valid {
			agent.CreatedAt = time.Unix(createdAtUnix.Int64, 0)
		}
		
		if updatedAtUnix.Valid {
			agent.UpdatedAt = time.Unix(updatedAtUnix.Int64, 0)
		}
		
		// 设置默认值
		if agent.Name == "" {
			agent.Name = agent.Hostname
		}
		if agent.Platform == "" {
			agent.Platform = "Unknown"
		}
		
		agents = append(agents, agent)
	}
	
	if err = rows.Err(); err != nil {
		log.Printf("数据遍历错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取代理列表失败", "detail": "遍历结果集时发生错误", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agents)
}

// 根据ID获取代理
func getAgentByID(c *gin.Context) {
	agentID := c.Param("id")
	log.Printf("API call: %s %s (id: %s)", c.Request.Method, c.Request.URL.Path, agentID)

	// 查询代理详情
	query := "SELECT id, name, hostname, platform, ip_address, last_seen, COALESCE(created_at, 0) as created_at, COALESCE(updated_at, 0) as updated_at FROM agents WHERE id = ?"
	
	log.Printf("执行查询: %s", query)
	
	var agent Agent
	var lastSeenUnix sql.NullInt64
	var createdAtUnix, updatedAtUnix sql.NullInt64
	
	err := db.QueryRow(query, agentID).Scan(
		&agent.ID, 
		&agent.Name, 
		&agent.Hostname, 
		&agent.Platform, 
		&agent.IPAddress, 
		&lastSeenUnix, 
		&createdAtUnix, 
		&updatedAtUnix)
		
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("代理不存在: %s", agentID)
			c.JSON(http.StatusNotFound, gin.H{"error": "代理不存在", "detail": "找不到指定ID的代理"})
		} else {
			log.Printf("获取代理错误: %s, %v", agentID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "detail": err.Error()})
		}
		return
	}
	
	// 设置LastSeen和在线状态
	if lastSeenUnix.Valid {
		agent.LastSeen = time.Unix(lastSeenUnix.Int64, 0)
		// 将在线状态判断从5分钟改为10秒以减少误导
		agent.IsOnline = time.Since(agent.LastSeen) < 10*time.Second
	} else {
		agent.LastSeen = time.Time{}
		agent.IsOnline = false
	}
	
	if createdAtUnix.Valid {
		agent.CreatedAt = time.Unix(createdAtUnix.Int64, 0)
	}
	
	if updatedAtUnix.Valid {
		agent.UpdatedAt = time.Unix(updatedAtUnix.Int64, 0)
	}
	
	if agent.Name == "" {
		agent.Name = agent.Hostname
	}
	if agent.Platform == "" {
		agent.Platform = "Unknown"
	}

	c.JSON(http.StatusOK, agent)
}

// 更新代理
func updateAgent(c *gin.Context) {
	agentID := c.Param("id")
	log.Printf("API call: %s %s (id: %s)", c.Request.Method, c.Request.URL.Path, agentID)

	var updateData struct {
		Name     string `json:"name"`
		Hostname string `json:"hostname"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		log.Printf("无效的请求数据: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效", "detail": err.Error()})
		return
	}

	// 检查代理是否存在
	var exists bool
	err := db.QueryRow("SELECT 1 FROM agents WHERE id = ?", agentID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("要更新的代理不存在: %s", agentID)
			c.JSON(http.StatusNotFound, gin.H{"error": "代理不存在", "detail": "找不到指定ID的代理"})
		} else {
			log.Printf("检查代理存在性错误: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "detail": err.Error()})
		}
		return
	}

	// 更新代理名称和主机名
	now := time.Now().Unix()
	result, err := db.Exec("UPDATE agents SET name = ?, hostname = ?, updated_at = ? WHERE id = ?", updateData.Name, updateData.Hostname, now, agentID)
	if err != nil {
		log.Printf("更新代理失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新代理失败", "detail": err.Error()})
		return
	}

	// 同步写入hostname.json
	hostnameFile := "hostname.json"
	hostnameMap := map[string]string{}
	if data, err := ioutil.ReadFile(hostnameFile); err == nil {
		_ = json.Unmarshal(data, &hostnameMap)
	}
	if updateData.Hostname != "" {
		hostnameMap[agentID] = updateData.Hostname
		if data, err := json.MarshalIndent(hostnameMap, "", "  "); err == nil {
			_ = ioutil.WriteFile(hostnameFile, data, 0644)
		}
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("已更新代理 %s, 影响行数: %d", agentID, rowsAffected)

	c.JSON(http.StatusOK, gin.H{"message": "代理已更新", "agent_id": agentID})
}

// 删除代理
func deleteAgent(c *gin.Context) {
	agentID := c.Param("id")
	log.Printf("API call: %s %s (id: %s)", c.Request.Method, c.Request.URL.Path, agentID)

	// 检查代理是否存在
	var exists bool
	err := db.QueryRow("SELECT 1 FROM agents WHERE id = ?", agentID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("要删除的代理不存在: %s", agentID)
			c.JSON(http.StatusNotFound, gin.H{"error": "代理不存在", "detail": "找不到指定ID的代理"})
		} else {
			log.Printf("检查代理存在性错误: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "detail": err.Error()})
		}
		return
	}

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		log.Printf("开始事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "detail": err.Error()})
		return
	}

	// 删除相关的指标
	result, err := tx.Exec("DELETE FROM metrics WHERE agent_id = ?", agentID)
	if err != nil {
		tx.Rollback()
		log.Printf("删除代理指标失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除代理指标失败", "detail": err.Error()})
		return
	}
	
	metricsRowsDeleted, _ := result.RowsAffected()
	log.Printf("已删除代理 %s 的 %d 条指标记录", agentID, metricsRowsDeleted)

	// 删除代理
	result, err = tx.Exec("DELETE FROM agents WHERE id = ?", agentID)
	if err != nil {
		tx.Rollback()
		log.Printf("删除代理失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除代理失败", "detail": err.Error()})
		return
	}
	
	agentRowsDeleted, _ := result.RowsAffected()
	if agentRowsDeleted == 0 {
		tx.Rollback()
		log.Printf("代理 %s 可能已被删除", agentID)
		c.JSON(http.StatusNotFound, gin.H{"error": "代理不存在", "detail": "代理可能已被删除"})
		return
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		log.Printf("提交事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "detail": err.Error()})
		return
	}
	
	log.Printf("已成功删除代理 %s 及其所有指标数据", agentID)

	c.JSON(http.StatusOK, gin.H{
		"message": "代理已删除",
		"agent_id": agentID,
		"metrics_deleted": metricsRowsDeleted,
	})
}

// 获取代理指标
func getAgentMetrics(c *gin.Context) {
	agentID := c.Param("id")
	log.Printf("API call: %s %s (agent_id: %s)", c.Request.Method, c.Request.URL.Path, agentID)
	
	// 获取时间范围参数
	timeFromStr := c.DefaultQuery("from", "0")
	timeToStr := c.DefaultQuery("to", fmt.Sprintf("%d", time.Now().Unix()))
	
	timeFrom, err := strconv.ParseInt(timeFromStr, 10, 64)
	if err != nil {
		log.Printf("无效的from时间参数: %s, %v", timeFromStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的时间范围参数", "detail": fmt.Sprintf("from参数格式错误: %v", err)})
		return
	}
	
	timeTo, err := strconv.ParseInt(timeToStr, 10, 64)
	if err != nil {
		log.Printf("无效的to时间参数: %s, %v", timeToStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的时间范围参数", "detail": fmt.Sprintf("to参数格式错误: %v", err)})
		return
	}
	
	// 获取数据限制
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		log.Printf("无效的limit参数: %s, 使用默认值100", limitStr)
		limit = 100
	}

	// 首先检查代理是否存在
	var exists bool
	err = db.QueryRow("SELECT 1 FROM agents WHERE id = ?", agentID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("要查询的代理不存在: %s", agentID)
			c.JSON(http.StatusNotFound, gin.H{"error": "代理不存在", "detail": "找不到指定ID的代理"})
		} else {
			log.Printf("检查代理存在性错误: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "detail": fmt.Sprintf("检查代理存在性错误: %v", err)})
		}
		return
	}
	
	log.Printf("查询代理 %s 的指标, 从 %d 到 %d, 限制 %d 条", agentID, timeFrom, timeTo, limit)
	
	// 查询该agent的metrics总记录数
	var metricsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM metrics WHERE agent_id = ?", agentID).Scan(&metricsCount)
	if err != nil {
		log.Printf("查询metrics总数量错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "detail": fmt.Sprintf("查询指标数量时出错: %v", err)})
		return
	}
	
	log.Printf("找到 %d 条代理 %s 的指标记录", metricsCount, agentID)
	
	// 如果该agent没有任何metrics记录，直接返回空数组
	if metricsCount == 0 {
		log.Printf("代理 %s 没有任何指标数据，返回空数组", agentID)
		c.JSON(http.StatusOK, []map[string]interface{}{})
		return
	}
	
	// 查询指定时间范围内的记录数
	var rangeCount int
	err = db.QueryRow("SELECT COUNT(*) FROM metrics WHERE agent_id = ? AND timestamp >= ? AND timestamp <= ?", 
		agentID, timeFrom, timeTo).Scan(&rangeCount)
	if err != nil {
		log.Printf("查询时间范围内的metrics数量错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "detail": fmt.Sprintf("查询时间范围内的指标数量时出错: %v", err)})
		return
	}
	
	log.Printf("时间范围内找到 %d 条代理 %s 的指标记录", rangeCount, agentID)
	
	// 如果时间范围内没有记录，返回空数组
	if rangeCount == 0 {
		log.Printf("指定时间范围内没有数据，返回空数组")
		c.JSON(http.StatusOK, []map[string]interface{}{})
		return
	}
	
	// 查询指标
	query := `
		SELECT 
			timestamp, cpu_usage, 
			memory_total, memory_used, memory_percent,
			disk_total, disk_used, disk_percent,
			network_sent, network_recv,
			load_avg_1, load_avg_5, load_avg_15,
			process_count
		FROM metrics
		WHERE agent_id = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp DESC
		LIMIT ?
	`
	
	log.Printf("执行查询: %s", query)
	rows, err := db.Query(query, agentID, timeFrom, timeTo, limit)
	
	if err != nil {
		log.Printf("查询代理指标错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取指标", "detail": fmt.Sprintf("数据库查询失败: %v", err)})
		return
	}
	defer rows.Close()
	
	var metrics []map[string]interface{}
	for rows.Next() {
		var (
			timestamp                                                          int64
			cpuUsage, memPercent, diskPercent                                 float64
			loadAvg1, loadAvg5, loadAvg15                                     float64
			memTotal, memUsed, diskTotal, diskUsed, netSent, netRecv          int64
			processCount                                                       int
		)
		
		if err := rows.Scan(
			&timestamp, &cpuUsage,
			&memTotal, &memUsed, &memPercent,
			&diskTotal, &diskUsed, &diskPercent,
			&netSent, &netRecv,
			&loadAvg1, &loadAvg5, &loadAvg15,
			&processCount,
		); err != nil {
			log.Printf("扫描指标行数据错误: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "处理指标数据错误", "detail": fmt.Sprintf("解析数据行失败: %v", err)})
			return
		}
		
		log.Printf("行数据: timestamp=%d, cpu=%.2f%%, mem=%.2f%%, disk=%.2f%%", 
			timestamp, cpuUsage, memPercent, diskPercent)
		
		metric := map[string]interface{}{
			"timestamp":    timestamp,
			"cpu_usage":    cpuUsage,
			"memory_info": map[string]interface{}{
				"total":   memTotal,
				"used":    memUsed,
				"percent": memPercent,
			},
			"disk_info": map[string]interface{}{
				"total":   diskTotal,
				"used":    diskUsed,
				"percent": diskPercent,
			},
			"network_info": map[string]interface{}{
				"bytes_sent": netSent,
				"bytes_recv": netRecv,
			},
			"load_average": map[string]interface{}{
				"load1":  loadAvg1,
				"load5":  loadAvg5,
				"load15": loadAvg15,
			},
			"process_count": processCount,
		}
		
		metrics = append(metrics, metric)
	}
	
	if err = rows.Err(); err != nil {
		log.Printf("指标数据遍历错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据遍历错误", "detail": fmt.Sprintf("遍历数据集时发生错误: %v", err)})
		return
	}
	
	log.Printf("成功获取代理 %s 的 %d 条指标记录", agentID, len(metrics))
	
	// 如果查询结果为空（理论上不应该发生，因为之前已经检查了），也返回空数组
	if len(metrics) == 0 {
		log.Printf("查询结果为空，返回空数组")
		c.JSON(http.StatusOK, []map[string]interface{}{})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// 生成模拟指标数据
func generateFallbackMetrics(agentID string, timeFrom, timeTo int64, limit int) []map[string]interface{} {
	log.Printf("请求的时间范围内没有数据，而不是生成模拟数据，返回空数组")
	return []map[string]interface{}{}
}

// 加载配置文件
func loadConfig(configFile string) (Config, error) {
	var config Config
	
	// 检查配置文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// 配置文件不存在，创建默认配置
		config = Config{
			Port:          8080,
			DBPath:        "./linux-monitor.db",
			EncryptionKey: "default-encryption-key-change-me",
			APIKey:        "change-me-in-production",
			JWTSecret:     uuid.New().String(), // 生成新的UUID作为JWT密钥
		}
		
		// 保存默认配置到文件
		if err := saveConfig(configFile, config); err != nil {
			return config, fmt.Errorf("无法保存默认配置：%v", err)
		}
		
		log.Printf("创建了新的配置文件：%s", configFile)
	} else {
		// 读取配置文件
		data, err := ioutil.ReadFile(configFile)
		if err != nil {
			return config, fmt.Errorf("无法读取配置文件：%v", err)
		}
		
		// 解析JSON配置
		if err := json.Unmarshal(data, &config); err != nil {
			return config, fmt.Errorf("无法解析配置文件：%v", err)
		}
		
		// 如果JWT密钥为空，生成新的并保存
		if config.JWTSecret == "" {
			config.JWTSecret = uuid.New().String()
			if err := saveConfig(configFile, config); err != nil {
				log.Printf("警告：无法保存更新的JWT密钥：%v", err)
			}
		}
	}
	
	return config, nil
}

// 保存配置到文件
func saveConfig(configFile string, config Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("无法序列化配置：%v", err)
	}
	
	if err := ioutil.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("无法写入配置文件：%v", err)
	}
	
	return nil
}

// 获取webhook配置
func getWebhook(c *gin.Context) {
	data, err := ioutil.ReadFile("webhook.json")
	if err != nil {
		c.JSON(500, gin.H{"error": "无法读取webhook.json"})
		return
	}
	var arr []Webhook
	if err := json.Unmarshal(data, &arr); err != nil {
		c.JSON(500, gin.H{"error": "webhook.json格式错误"})
		return
	}
	c.JSON(200, arr)
}

// 设置webhook配置（仅管理员）
func setWebhook(c *gin.Context) {
	var arr []Webhook
	if err := c.ShouldBindJSON(&arr); err != nil {
		c.JSON(400, gin.H{"error": "参数无效"})
		return
	}
	data, _ := json.MarshalIndent(arr, "", "  ")
	_ = ioutil.WriteFile("webhook.json", data, 0644)
	c.JSON(200, gin.H{"message": "已保存"})
}

// Server酱推送
func sendServerChan(sendKey, title, desp string) ([]byte, error) {
	apiUrl := fmt.Sprintf("https://sctapi.ftqq.com/%s.send", sendKey)
	data := url.Values{}
	data.Set("title", title)
	data.Set("desp", desp)
	log.Printf("[ServerChan] POST %s, title=%s, desp=%s", apiUrl, title, desp)
	resp, err := http.PostForm(apiUrl, data)
	if err != nil {
		log.Printf("[ServerChan] 请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ServerChan] 读取响应失败: %v", err)
		return nil, err
	}
	log.Printf("[ServerChan] 响应: %s", string(body))
	return body, nil
}

func alertTask() {
	for {
		agents := []Agent{}
		rows, err := db.Query("SELECT id, name, hostname, last_seen FROM agents")
		if err == nil {
			for rows.Next() {
				var a Agent
				var lastSeenUnix int64
				_ = rows.Scan(&a.ID, &a.Name, &a.Hostname, &lastSeenUnix)
				a.LastSeen = time.Unix(lastSeenUnix, 0)
				agents = append(agents, a)
			}
			rows.Close()
		}
		// 获取webhook配置
		webhookData, _ := ioutil.ReadFile("webhook.json")
		var webhooks []Webhook
		_ = json.Unmarshal(webhookData, &webhooks)
		// 检查每个agent
		for _, agent := range agents {
			// 离线判定
			if time.Since(agent.LastSeen) > 30*time.Second {
				if !offlineAlerted[agent.ID] {
					title := "Agent离线告警"
					desp := fmt.Sprintf("Agent %s(%s) 已离线，最后在线时间：%s", agent.Name, agent.ID, agent.LastSeen.Format(time.RFC3339))
					for _, wh := range webhooks {
						if wh.Enabled && wh.Type == "serverchan" && wh.SendKey != "" {
							_, _ = sendServerChan(wh.SendKey, title, desp)
						}
					}
					offlineAlerted[agent.ID] = true
				}
			} else {
				offlineAlerted[agent.ID] = false
			}
			// 高负载判定（10分钟）
			row := db.QueryRow("SELECT timestamp, cpu_usage FROM metrics WHERE agent_id = ? ORDER BY timestamp DESC LIMIT 1", agent.ID)
			var ts int64
			var cpu float64
			_ = row.Scan(&ts, &cpu)
			if cpu > 90 {
				if highLoadStart[agent.ID] == 0 {
					highLoadStart[agent.ID] = ts
				}
				if ts-highLoadStart[agent.ID] >= 600 && !highLoadAlerted[agent.ID] {
					title := "Agent高负载告警"
					desp := fmt.Sprintf("Agent %s(%s) 已高负载10分钟，当前CPU: %.2f%%", agent.Name, agent.ID, cpu)
					for _, wh := range webhooks {
						if wh.Enabled && wh.Type == "serverchan" && wh.SendKey != "" {
							_, _ = sendServerChan(wh.SendKey, title, desp)
						}
					}
					highLoadAlerted[agent.ID] = true
				}
			} else {
				highLoadStart[agent.ID] = 0
				highLoadAlerted[agent.ID] = false
			}
		}
		time.Sleep(60 * time.Second)
	}
}

func testWebhook(c *gin.Context) {
	var wh Webhook
	if err := c.ShouldBindJSON(&wh); err != nil {
		log.Printf("[WebhookTest] 参数无效: %v", err)
		c.JSON(400, gin.H{"error": "参数无效"})
		return
	}
	log.Printf("[WebhookTest] 测试请求: %+v", wh)
	title := "Webhook测试消息"
	desp := "这是一条Webhook测试消息，说明配置已生效。"
	if wh.Type == "serverchan" && wh.SendKey != "" {
		body, err := sendServerChan(wh.SendKey, title, desp)
		if err != nil {
			log.Printf("[WebhookTest] Server酱推送失败: %v", err)
			c.JSON(500, gin.H{"error": "Server酱推送失败", "detail": err.Error()})
			return
		}
		var resp struct{ Code int `json:"code"`; Message string `json:"message"` }
		err2 := json.Unmarshal(body, &resp)
		if err2 != nil {
			log.Printf("[WebhookTest] Server酱响应解析失败: %v, 原始: %s", err2, string(body))
			c.JSON(200, gin.H{"message": "FAIL", "detail": "响应解析失败", "raw": string(body)})
			return
		}
		log.Printf("[WebhookTest] Server酱响应解析: code=%d, message=%s", resp.Code, resp.Message)
		if resp.Code == 0 {
			c.JSON(200, gin.H{"message": "SUCCESS"})
		} else {
			c.JSON(200, gin.H{"message": "FAIL", "detail": resp.Message, "raw": string(body)})
		}
		return
	}
	if wh.Type == "custom" && wh.URL != "" {
		body := map[string]string{"title": title, "desc": desp}
		b, _ := json.Marshal(body)
		log.Printf("[WebhookTest] POST %s, body=%s", wh.URL, string(b))
		resp, err := http.Post(wh.URL, "application/json", strings.NewReader(string(b)))
		if err != nil {
			log.Printf("[WebhookTest] 自定义Webhook推送失败: %v", err)
			c.JSON(500, gin.H{"error": "自定义Webhook推送失败", "detail": err.Error()})
			return
		}
		defer resp.Body.Close()
		log.Printf("[WebhookTest] 自定义Webhook响应状态: %d", resp.StatusCode)
		c.JSON(200, gin.H{"message": "SUCCESS"})
		return
	}
	log.Printf("[WebhookTest] 不支持的Webhook类型或缺少必要参数: %+v", wh)
	c.JSON(400, gin.H{"error": "不支持的Webhook类型或缺少必要参数"})
}