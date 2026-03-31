# GlobalServer - 全局服务器
## 概述

GlobalServer（全局服务器）是MMO游戏服务器架构的中心管理服务，负责账号管理、服务器列表维护、服务器注册和心跳监控。它提供HTTP API接口，是客户端登录和服务器管理的入口。

**设计目标**：为游戏玩家提供统一的登录入口，为游戏服务器提供集中的管理平台，确保整个游戏生态的稳定运行。

**本服务器架构作为其他服务器的参考模板，体现了模块化、可扩展的设计理念。**

## 设计理念

### 为什么需要GlobalServer？

在MMO游戏中，通常会有多个游戏服务器（GameServer）来承载玩家。为了管理这些服务器并为玩家提供统一的登录体验，我们需要一个中心化的管理服务：

1. **统一入口**：玩家通过GlobalServer进行登录，获得可用的游戏服务器列表
2. **服务器管理**：集中管理所有游戏服务器的注册、状态监控和负载均衡
3. **账号管理**：统一的账号系统，避免每个游戏服务器都维护一套账号体系
4. **服务发现**：通过etcd实现服务注册与发现，确保服务器间的动态发现

### 核心设计原则

1. **模块化设计**：将不同功能拆分为独立模块，便于维护和扩展
2. **分层架构**：清晰的分层结构，职责明确
3. **高可用性**：支持多实例部署，实现负载均衡
4. **可监控性**：集成Prometheus监控，实时掌握系统状态
5. **安全性**：JWT认证保护，防止未授权访问

## 核心功能

### 1. 账号管理
- **账号创建**：支持新用户注册账号，建立游戏账号体系
- **账号登录**：验证用户身份并返回Token，确保安全访问
- **JWT认证**：使用JWT令牌进行身份验证，无状态设计便于水平扩展
- **登录状态**：记录用户最后登录时间，便于数据分析和用户行为跟踪

### 2. 服务器管理
- **服务器注册**：游戏服务器启动时自动注册到GlobalServer，实现动态发现
- **服务器列表**：提供可用游戏服务器列表，支持玩家选择合适的服务器
- **心跳监控**：监控游戏服务器在线状态，及时发现异常服务器
- **负载均衡**：根据服务器负载分配玩家，确保各服务器负载均衡

### 3. HTTP API服务
- **RESTful API**：提供标准的HTTP API接口，便于客户端和其他服务调用
- **健康检查**：服务状态监控接口，用于监控系统运行状态
- **限流保护**：防止API被滥用，保护系统稳定
- **CORS支持**：支持跨域请求，便于前端开发

### 4. 数据持久化
- **MySQL数据库**：存储账号和服务器信息，确保数据持久化
- **连接池管理**：高效数据库连接复用，提高性能
- **数据验证**：确保数据完整性，防止无效数据

### 5. 监控系统
- **Prometheus集成**：标准指标监控，便于系统监控和告警
- **业务指标**：账号、服务器、数据库操作指标，了解业务运行状态
- **系统指标**：运行时间、HTTP请求统计，掌握系统性能
- **健康检查**：服务状态监控，确保系统正常运行
- **Metrics API**：`/metrics` 端点暴露监控数据，便于集成监控系统

## 系统架构

```
┌─────────────────────────────────────────────────────┐
│                 GlobalServer                        │
├─────────────────────────────────────────────────────┤
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐    │
│ │ HTTP API    │ │ 账号管理    │ │ 服务器管理  │    │
│ │├─/health    │ │├─创建       │ │├─注册       │    │
│ │├─/api/v1    │ │├─登录       │ │├─心跳       │    │
│ │└─限流       │ │└─Token      │ │└─列表       │    │
│ └─────────────┘ └─────────────┘ └─────────────┘    │
│ ┌─────────────┐ ┌─────────────┐                    │
│ │ 数据存储    │ │ 日志监控    │                    │
│ │├─MySQL      │ │├─日志       │                    │
│ │└─连接池     │ │└─pprof      │                    │
│ └─────────────┘ └─────────────┘                    │
└─────────────────────────────────────────────────────┘
        │                   │
        │                   │
  ┌──────────┐       ┌──────────┐
  │ Client   │       │ GameSrv  │
  └──────────┘       └──────────┘
```

## 目录结构

```
GlobalServer/
├── global/              # 全局服务核心
│   └── base_server.go   # 服务器基础实现
├── handler/            # HTTP处理器
│   └── handler.go      # API处理函数
├── http/               # HTTP服务
│   └── http_service.go # Echo HTTP服务实现
├── db/                 # 数据库服务
│   └── service.go      # 数据库服务实现
├── config/             # 配置管理
│   └── config.go       # 配置加载和验证
├── metrics/            # 监控指标
│   └── metrics.go      # GlobalServer 特定指标
├── version/            # 版本管理
│   └── version.go      # 版本信息
├── logs/               # 日志目录
│   └── server.log      # 运行日志
├── config.ini          # 配置文件
├── main.go             # 入口程序
├── go.mod              # Go模块定义
└── README.md           # 本文档
```

## 快速开始
### 1. 环境要求
- Go 1.20+
- MySQL 5.7+
- 配置文件 config.ini

### 2. 数据库初始化
```sql
-- 创建全局数据库
CREATE DATABASE global CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建账号表
CREATE TABLE accounts (
    account_id BIGINT PRIMARY KEY,
    account_name VARCHAR(64) NOT NULL UNIQUE,
    password VARCHAR(64) NOT NULL,
    status TINYINT DEFAULT 1,
    created_at DATETIME NOT NULL,
    last_login_at DATETIME NOT NULL,
    INDEX idx_account_name (account_name)
);

-- 创建游戏服务器表
CREATE TABLE game_servers (
    server_id INT PRIMARY KEY,
    server_name VARCHAR(64) NOT NULL,
    server_type VARCHAR(32) NOT NULL,
    group_id INT DEFAULT 0,
    address VARCHAR(128) NOT NULL,
    port INT NOT NULL,
    status TINYINT DEFAULT 1,
    online_count INT DEFAULT 0,
    max_online_count INT DEFAULT 1000,
    region VARCHAR(32),
    version VARCHAR(32),
    last_heartbeat DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    INDEX idx_server_id (server_id)
);
```

### 3. 编译运行
```bash
# 编译
./build.ps1  # Windows
./build.sh   # Linux/Mac

# 运行（使用默认配置）
./bin/global-server.exe

# 运行（指定配置文件）
./bin/global-server.exe
```

### 4. 验证服务
```bash
# 健康检查
curl http://localhost:8888/health

# 创建账号
curl -X POST http://localhost:8888/api/v1/account/create \
  -H "Content-Type: application/json" \
  -d '{"account":"testuser","password":"test123","email":"test@example.com"}'

# 登录
curl -X POST http://localhost:8888/api/v1/account/login \
  -H "Content-Type: application/json" \
  -d '{"account":"testuser","password":"test123"}'

# 获取服务器列表
curl http://localhost:8888/api/v1/server/list
```

## API接口文档

### 1. 健康检查
**GET** `/health`

**响应示例：**
```json
{
  "status": "ok",
  "service": "GlobalServer",
  "version": "0.0.1",
  "build_time": "2026-03-07 22:06:36",
  "git_commit": "abc123",
  "time": "2024-01-01T12:00:00Z"
}
```

### 2. 创建账号
**POST** `/api/v1/account/create`

**请求参数：**
```json
{
  "account": "用户名",
  "password": "密码",
  "email": "邮箱"
}
```

**响应示例：**
```json
{
  "success": true,
  "error_msg": ""
}
```

**错误码：**
- 400: 参数错误
- 409: 账号已存在
- 500: 服务器错误

### 3. 账号登录
**POST** `/api/v1/account/login`

**请求参数：**
```json
{
  "account": "用户名",
  "password": "密码"
}
```

**响应示例：**
```json
{
  "success": true,
  "error_msg": "",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "servers": [
    {
      "server_id": 1,
      "server_name": "GameServer-1",
      "server_type": "normal",
      "group_id": 0,
      "address": "127.0.0.1",
      "port": 9001,
      "status": 1,
      "online_count": 0,
      "max_online_count": 5000,
      "region": "CN",
      "version": "1.0.0"
    }
  ]
}
```

**错误码：**
- 400: 参数错误
- 401: 账号或密码错误
- 500: 服务器错误

### 4. 获取服务器列表
**GET** `/api/v1/server/list`

**响应示例：**
```json
{
  "success": true,
  "error_msg": "",
  "servers": [
    {
      "server_id": 1,
      "server_name": "GameServer-1",
      "server_type": "normal",
      "group_id": 0,
      "address": "127.0.0.1",
      "port": 9001,
      "status": 1,
      "online_count": 0,
      "max_online_count": 5000,
      "region": "CN",
      "version": "1.0.0"
    }
  ]
}
```

## 配置文件说明

### [server] - 服务器基本配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| server_id | 服务器ID | 1 |
| server_name | 服务器名称 | GlobalServer |
| worker_id | Snowflake工作机器ID(0-31) | 1 |
| datacenter_id | Snowflake数据中心ID(0-31) | 1 |

### [http] - HTTP服务配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| listen_address | HTTP监听地址 | 0.0.0.0:8888 |
| max_client_count | 最大客户端连接数 | 10000 |
| max_packet_data_size | 最大数据包大小 | 1048576 |
| enabled | 是否启用HTTP服务 | true |

### [database.global] - 数据库配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| host | 数据库主机地址 | localhost |
| port | 数据库端口 | 3306 |
| user | 数据库用户名 | root |
| password | 数据库密码 | 空 |
| dbname | 数据库名称 | global |
| driver | 数据库驱动 | mysql |
| max_pool_size | 连接池最大连接数 | 100 |
| min_pool_size | 连接池最小连接数 | 10 |
| connect_timeout | 连接超时时间(秒) | 30 |

### [log] - 日志配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| level | 日志级别(0=debug, 1=info, 2=warn, 3=error) | 0 |
| console | 是否输出到控制台 | true |
| console_level | 控制台日志级别 | 0 |
| filename | 日志文件路径 | ./logs/server.log |
| max-size | 单个日志文件最大大小(MB) | 100 |
| max-days | 日志保留天数 | 15 |
| max-backups | 日志备份数量 | 10 |
| compress | 是否压缩日志 | true |
| show-caller | 是否显示调用者信息 | true |
| stacktrace | 错误堆栈级别 | 3 |
| sampling | 是否启用采样 | true |
| sampling-initial | 初始采样数量 | 100 |
| sampling-thereafter | 后续采样间隔 | 10 |
| async | 是否异步写入 | true |
| async-buffer-size | 异步缓冲区大小 | 2048 |
| async-flush-interval | 异步刷新间隔(ms) | 50 |

### [pprof] - 性能分析配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| enabled | 是否启用pprof | false |
| listen_address | pprof监听地址 | localhost:6060 |

### [metrics] - 监控指标配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| enabled | 是否启用监控 | true |
| listen_address | 监控服务监听地址 | 0.0.0.0:8889 |

## 架构设计指南

### 1. 服务器架构模式
GlobalServer 采用**分层架构**设计，其他服务器应遵循相同模式：

```
┌─────────────────────────────────────────┐
│             Entry (main.go)            │
├─────────────────────────────────────────┤
│          Service Layer                 │
│ ┌─────────┐ ┌─────────┐ ┌──────────┐   │
│ │ Global  │ │ HTTP    │ │ Metrics  │   │
│ │ Server  │ │ Service │ │ Service  │   │
│ └────┬────┘ └────┬────┘ └────┬─────┘   │
├──────┼───────────┼───────────┼─────────┤
│      └───────────┴───────────┘         │
│          Business Layer                │
│ ┌─────────┐ ┌─────────┐ ┌──────────┐   │
│ │ Handler │ │  DB     │ │ Config   │   │
│ └────┬────┘ └────┬────┘ └────┬─────┘   │
├──────┼───────────┼───────────┼─────────┤
│      └───────────┴───────────┘         │
│          Data Access Layer             │
│ ┌─────────┐ ┌─────────┐ ┌──────────┐   │
│ │ Repository │ │  DAO   │ │ Connector │ │
│ └─────────┘ └─────────┘ └──────────┘   │
└─────────────────────────────────────────┘
```

### 2. 生命周期管理

所有服务器必须实现 `LifecycleHooks` 接口：
```go
type LifecycleHooks interface {
    OnBeforeStart() error  // 初始化组件
    OnAfterStart() error   // 启动服务
    OnBeforeStop()         // 优雅关闭
}
```

**启动顺序：**
1. `OnBeforeStart` - 初始化所有组件（DB、HTTP、Metrics等）
2. `OnAfterStart` - 启动所有服务
3. 等待退出信号
4. `OnBeforeStop` - 优雅关闭所有资源

### 3. 配置管理规范

**统一配置结构：**
```go
type Config struct {
    Server   ServerConfig        // 服务器配置
    HTTP     HTTPConfig          // HTTP配置
    Log      zLog.Config         // 日志配置（使用zEngine标准配置）
    Database db.DBConfig         // 数据库配置（使用zCommon标准配置）
    Metrics  MetricsConfig       // 监控配置
}
```

**配置加载：**
```go
func LoadConfig(filePath string) (*Config, error) {
    // 1. 加载INI配置文件
    // 2. 映射到配置结构体
    // 3. 验证配置有效性
    // 4. 返回配置对象
}
```

### 4. 数据库访问层设计

**Repository模式：**
```
Handler -> Repository -> DAO -> Connector -> Database
```

**按需初始化：**
```go
// 定义需要的Repository类型
var RepoTypeMyServer = []db.RepoType{
    db.RepoTypeAccount,
    db.RepoTypePlayer,
    // ... 只初始化需要的
}

// 初始化时指定
sharedDB.InitDBManagerWithRepos(dbConfigs, RepoTypeMyServer)
```

### 5. 监控指标集成

**必须集成的指标：**
- HTTP请求指标（次数、延迟）
- 业务指标（根据服务器类型定义）
- 系统指标（运行时间）

**指标定义：**
```go
type Metrics struct {
    *sharedMetrics.ServerMetrics  // 基础指标
    
    // 业务指标
    myCounter prometheus.Counter
    myHistogram prometheus.Histogram
}
```

### 6. 版本管理

**版本文件：**
```go
// version/version.go
const Version = "0.0.1"
var BuildTime = ""
var GitCommit = ""
```

**构建脚本：**
- `build.ps1` - Windows构建
- `build.sh` - Linux/Mac构建

**自动注入版本信息：**
```bash
# 构建时自动更新version.go
./build.ps1
```

### 7. 日志规范

**统一使用zEngine/zLog：**
```go
// 初始化
zLog.InitLogger(cfg.GetLogConfig())

// 使用
zLog.Info("message", zap.String("key", value))
zLog.Error("message", zap.Error(err))
```

**日志级别：**
- Debug: 调试信息
- Info: 一般信息（默认）
- Warn: 警告信息
- Error: 错误信息

### 8. 错误处理规范

**分层错误处理：**
- DAO层：返回原始错误
- Repository层：包装业务错误
- Handler层：返回客户端友好的错误

**错误响应格式：**
```json
{
  "success": false,
  "error_msg": "用户友好的错误信息"
}
```

## 代码架构分析

### 1. 启动流程
```
main.go
  ├── 加载配置 (config.LoadConfig)
  ├── 初始化日志 (zLog.InitLogger)
  ├── 创建服务器 (global.NewBaseServer)
  ├── 注入日志 (SetLogger)
  └── 运行服务器 (Run)
        ├── OnBeforeStart
        │   ├── 设置服务器ID（从配置读取）
        │   ├── 初始化DBManager（按需初始化Repository）
        │   ├── 初始化ID生成器
        │   ├── 初始化Metrics服务
        │   ├── 初始化HTTPService
        │   ├── 初始化DBService
        │   └── 注册组件
        ├── 启动服务
        ├── OnAfterStart
        │   ├── 启动DB服务
        │   ├── 启动Metrics服务
        │   └── 启动HTTP服务
        └── 等待退出信号
              └── OnBeforeStop
                    ├── 停止HTTP服务
                    ├── 停止DB服务
                    └── 关闭DBManager
```

### 2. 账号注册流程
```
Client
  └── POST /api/v1/account/create
        └── handler.HandleAccountCreate
              ├── 解析请求参数
              ├── 验证参数完整性
              ├── 检查账号是否存在 (AccountRepository.GetByName)
              ├── 生成账号ID (Snowflake)
              ├── 创建账号对象
              ├── 保存到数据库 (AccountRepository.Create)
              ├── 记录注册指标
              └── 返回响应
```

### 3. 账号登录流程
```
Client
  └── POST /api/v1/account/login
        └── handler.HandleAccountLogin
              ├── 解析请求参数
              ├── 验证参数完整性
              ├── 查询账号 (AccountRepository.GetByName)
              ├── 验证密码
              ├── 更新最后登录时间
              ├── 获取服务器列表 (GameServerRepository.GetAll)
              ├── 生成JWT Token
              ├── 记录登录指标
              └── 返回登录响应
```

### 4. 数据库访问层架构
```
Handler
  └── Repository (AccountRepository)
        └── DAO (AccountDAO)
              └── Connector (MySQLConnector)
                    └── Database (MySQL)
```

## 关键技术点

### 1. 生命周期管理
- 使用 `LifecycleHooks` 接口管理服务器生命周期
- 在 `OnBeforeStart` 中初始化所有组件
- 在 `OnAfterStart` 中启动所有服务
- 在 `OnBeforeStop` 中优雅关闭所有资源

### 2. 数据库连接池
- 使用自定义连接池管理数据库连接
- 支持最大/最小连接数配置
- 支持连接超时设置
- 异步查询执行避免阻塞主线程

### 3. 缓存策略
- 使用 LRU 缓存存储热点数据
- 缓存过期时间：5分钟
- 缓存容量：1000条记录
- 数据更新时同步更新缓存

### 4. JWT认证
- 使用 HS256 算法签名
- Token 有效期：7天
- 包含账号ID和账号名
- 使用固定密钥：zMmoServerSecretKey

### 5. 错误处理
- 统一的错误响应格式
- 详细的错误日志记录
- 分层错误处理（DAO/Repository/Handler）
- 客户端友好的错误信息

## 性能优化

### 1. 数据库优化
- 连接池复用数据库连接
- 异步查询避免阻塞
- 按需初始化Repository（只初始化需要的）
- 合理的索引设计

### 2. HTTP优化
- 请求限流保护
- 连接复用
- 响应压缩
- 超时控制

### 3. 监控优化
- Prometheus指标收集
- 业务指标埋点
- 性能数据收集

## 安全考虑

### 1. 密码安全
- 密码明文存储（当前实现）
- 建议使用 bcrypt 等哈希算法

### 2. 接口安全
- JWT Token 认证
- 请求限流
- CORS 跨域控制
- 参数验证

### 3. 数据安全
- SQL 注入防护（使用参数化查询）
- 数据库连接加密
- 敏感数据脱敏

## 监控与运维
### 1. 健康检查
- HTTP 健康检查接口 (`/health`)
- 返回版本信息、构建时间、Git提交
- 数据库连接状态检查

### 2. 日志监控
- 结构化日志输出
- 日志级别控制
- 日志轮转和压缩
- 错误堆栈跟踪

### 3. 监控系统
- **Prometheus集成**：标准指标监控
- **Metrics API**：`/metrics` 端点暴露监控数据
- **业务指标**：账号操作、服务器管理、数据库操作
- **系统指标**：运行时间、HTTP请求统计
- **网络指标**：连接状态、延迟、吞吐量

### 4. 性能监控
- pprof 性能分析
- 数据库查询性能
- HTTP 请求性能
- 内存使用情况

## 负载均衡部署

### 1. Nginx 负载均衡配置

GlobalServer 支持多实例部署，使用 Nginx 进行负载均衡。以下是完整的 Nginx 配置示例：
```nginx
# Nginx 负载均衡配置
upstream global_servers {
    # 使用最少连接数策略，适合长连接场景
    least_conn;
    
    # GlobalServer 实例列表
    # max_fails: 失败次数阈值，超过则标记为不可用
    # fail_timeout: 失败超时时间
    # max_conns: 最大连接数
    server 192.168.91.128:8888 max_fails=3 fail_timeout=30s max_conns=10000;
    server 192.168.91.129:8888 max_fails=3 fail_timeout=30s max_conns=10000;
    server 192.168.91.130:8888 max_fails=3 fail_timeout=30s max_conns=10000;
    
    # 备用服务器（可选）
    # server 192.168.91.131:8888 backup;
}

server {
    listen 80;
    listen [::]:80;
    server_name global.example.com;
    
    # 访问日志
    access_log /var/log/nginx/global_access.log;
    error_log /var/log/nginx/global_error.log;
    
    # 客户端请求体大小限制
    client_max_body_size 10M;
    
    # 超时设置
    proxy_connect_timeout 30s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;
    
    location / {
        # 代理到 GlobalServer 集群
        proxy_pass http://global_servers;
        
        # 传递真实客户端信息
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket 支持（如果需要）
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
    
    # 健康检查端点（可选，用于监控）
    location /health {
        proxy_pass http://global_servers;
        access_log off;
    }
}
```

### 2. 多实例配置要点
#### 2.1 JWT 密钥配置
**重要**：所有 GlobalServer 实例必须使用相同的 JWT 密钥：
```ini
# config.ini - 所有实例必须相同
[server]
jwt_secret = your-secure-random-key-here
```

#### 2.2 Server ID 配置
每个实例必须使用不同的 `server_id`：
```ini
# 实例 1
[server]
server_id = 1

# 实例 2
[server]
server_id = 2

# 实例 3
[server]
server_id = 3
```

#### 2.3 日志文件配置
每个实例使用不同的日志文件，避免冲突：
```ini
# config.ini
[log]
# 支持占位符：{server_id} 会被替换为实际的服务器ID
filename = ./logs/global_server_{server_id}.log
```

实际生成的日志文件：
- `./logs/global_server_1.log`
- `./logs/global_server_2.log`
- `./logs/global_server_3.log`

#### 2.4 端口配置

**HTTP 服务**：所有实例使用相同端口（8888）
```ini
[http]
listen_address = 0.0.0.0:8888
```

**Metrics 服务**：每个实例使用不同端口或只在一个实例上启用
```ini
# 实例 1
[metrics]
listen_address = 0.0.0.0:8889

# 实例 2
[metrics]
listen_address = 0.0.0.0:8890

# 实例 3
[metrics]
listen_address = 0.0.0.0:8891
```

**Pprof 服务**：每个实例使用不同端口或只在调试实例上启用
```ini
# 实例 1
[pprof]
listen_address = localhost:6060

# 实例 2
[pprof]
listen_address = localhost:6061

# 实例 3
[pprof]
listen_address = localhost:6062
```

### 3. 健康检查配置
#### 3.1 GlobalServer 健康检查
GlobalServer 提供 `/health` 端点用于健康检查：

```bash
# 健康检查示例
curl http://192.168.91.128:8888/health
```

响应示例：
```json
{
  "status": "ok",
  "service": "GlobalServer",
  "version": "0.0.1",
  "build_time": "2026-03-07T12:00:00Z",
  "git_commit": "abc123",
  "time": "2026-03-08T12:00:00Z"
}
```

#### 3.2 Nginx 健康检查
Nginx 可以配置健康检查，自动剔除故障实例：
```nginx
upstream global_servers {
    least_conn;
    
    # 启用健康检查
    # check_interval: 检查间隔（秒）
    # rise: 成功次数阈值
    # fall: 失败次数阈值
    # timeout: 超时时间
    # type: 检查类型（http、tcp等）
    server 192.168.91.128:8888 max_fails=3 fail_timeout=30s;
    server 192.168.91.129:8888 max_fails=3 fail_timeout=30s;
    server 192.168.91.130:8888 max_fails=3 fail_timeout=30s;
}
```

**注意**：Nginx 商业版支持主动健康检查，开源版使用被动检查（max_fails）。

#### 3.3 监控系统集成
使用 Prometheus + Grafana 监控多实例：

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'globalserver'
    static_configs:
      - targets: 
          - '192.168.91.128:8889'
          - '192.168.91.129:8890'
          - '192.168.91.130:8891'
```

### 4. 部署检查清单
在部署多实例负载均衡前，请确认以下事项：

- [ ] 所有实例使用相同的 JWT 密钥
- [ ] 每个实例配置不同的 server_id
- [ ] 每个实例使用不同的日志文件名
- [ ] Metrics 端口配置（每个实例不同或只一个实例启用）
- [ ] Pprof 端口配置（每个实例不同或只调试实例启用）
- [ ] MySQL 数据库可被所有实例访问
- [ ] Redis 可被所有实例访问
- [ ] Nginx 配置正确，包含所有实例地址
- [ ] 防火墙规则允许 Nginx 访问所有实例
- [ ] 测试健康检查端点 `/health`
- [ ] 测试故障转移（停止一个实例，验证 Nginx 自动切换）
- [ ] 测试负载均衡（发送多个请求，验证分发到不同实例）
- [ ] 监控系统配置正确

### 5. 故障转移测试

测试故障转移流程：
1. **正常状态**：所有实例运行，Nginx 分发请求
2. **停止实例**：停止一个 GlobalServer 实例
3. **观察切换**：Nginx 自动将请求分发到其他实例
4. **恢复实例**：启动停止的实例
5. **验证恢复**：Nginx 自动恢复请求分发

### 6. 负载均衡策略

Nginx 支持多种负载均衡策略：
#### 6.1 轮询（Round Robin）
```nginx
upstream global_servers {
    # 默认策略，按顺序分发
    server 192.168.91.128:8888;
    server 192.168.91.129:8888;
    server 192.168.91.130:8888;
}
```

#### 6.2 最少连接（Least Connections）
```nginx
upstream global_servers {
    # 推荐策略，适合长连接场景
    least_conn;
    server 192.168.91.128:8888;
    server 192.168.91.129:8888;
    server 192.168.91.130:8888;
}
```

#### 6.3 IP 哈希（IP Hash）
```nginx
upstream global_servers {
    # 同一客户端总是访问同一服务器
    ip_hash;
    server 192.168.91.128:8888;
    server 192.168.91.129:8888;
    server 192.168.91.130:8888;
}
```

**推荐**：使用 `least_conn` 策略，适合游戏服务器的长连接场景。

### 7. 性能优化建议

#### 7.1 连接保持
```nginx
upstream global_servers {
    least_conn;
    server 192.168.91.128:8888;
    server 192.168.91.129:8888;
    
    # 保持连接
    keepalive 32;
    keepalive_timeout 60s;
}
```

#### 7.2 缓冲区优化
```nginx
server {
    location / {
        proxy_pass http://global_servers;
        
        # 缓冲区设置
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
        proxy_busy_buffers_size 8k;
    }
}
```

#### 7.3 压缩
```nginx
server {
    location / {
        proxy_pass http://global_servers;
        
        # 启用压缩
        gzip on;
        gzip_types text/plain application/json;
        gzip_min_length 1000;
    }
}
```

## 常见问题

### 1. 数据库连接失败
- 检查数据库配置是否正确
- 检查数据库服务是否启动
- 检查网络连接是否正常
- 检查防火墙设置

### 2. 账号注册失败
- 检查账号是否已存在
- 检查数据库表结构是否正确
- 检查数据库连接是否正常
- 查看错误日志获取详细信息

### 3. 登录失败
- 检查账号密码是否正确
- 检查账号是否存在
- 检查数据库连接是否正常
- 查看错误日志获取详细信息

### 4. 服务器列表为空
- 检查 game_servers 表是否有数据
- 检查数据库查询是否正常
- 检查数据库连接是否正常

## 更新日志

### v0.0.1 (2026-03-07)
- 初始版本发布
- 实现账号注册和登录功能
- 实现服务器列表管理
- 实现服务器注册和心跳
- 新增监控系统
- 集成 Prometheus 指标
- 实现业务指标监控
- 实现系统和网络指标
- 增加 Metrics API 端点
- 优化配置管理
- 完善错误处理和日志记录
- 修复数据库表结构不匹配问题
- 修复账号注册返回值问题
- 按需初始化Repository优化
- 服务器ID配置优化
- 版本管理自动化

## 开发团队
- 架构设计：zEngine Team
- 开发维护：zMmoServer Team

## 许可证
MIT License

## 联系方式

- 项目主页：https://github.com/pzqf/zMmoServer
- 问题反馈：https://github.com/pzqf/zMmoServer/issues