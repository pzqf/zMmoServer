# GlobalServer - 全局服务器

## 概述

GlobalServer（全局服务器）是MMO游戏服务器架构的中心管理服务，负责账号管理、服务器列表维护、服务器注册和心跳监控。它提供HTTP API接口，是客户端登录和服务器管理的入口。

## 核心功能

### 1. 账号管理
- **账号创建**：支持新用户注册账号
- **账号登录**：验证用户身份并返回Token
- **JWT认证**：生成和验证JWT令牌
- **登录状态**：记录用户最后登录时间

### 2. 服务器管理
- **服务器注册**：游戏服务器启动时注册到GlobalServer
- **服务器列表**：提供可用游戏服务器列表
- **心跳监控**：监控游戏服务器在线状态
- **负载均衡**：根据服务器负载分配玩家

### 3. HTTP API服务
- **RESTful API**：提供标准的HTTP API接口
- **健康检查**：服务状态监控接口
- **限流保护**：防止API被滥用
- **CORS支持**：支持跨域请求

### 4. 数据持久化
- **MySQL数据库**：存储账号和服务器信息
- **连接池管理**：高效数据库连接复用
- **数据验证**：确保数据完整性

## 系统架构

```
┌─────────────────────────────────────────────────┐
│                  GlobalServer                    │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │
│  │ HTTP API    │  │ 账号管理    │  │ 服务器   │ │
│  │ ├─/health  │  │ ├─创建     │  │ ├─注册   │ │
│  │ ├─/api/v1  │  │ ├─登录     │  │ ├─心跳   │ │
│  │ └─限流     │  │ └─Token    │  │ └─列表   │ │
│  └─────────────┘  └─────────────┘  └──────────┘ │
│  ┌─────────────┐  ┌─────────────┐               │
│  │ 数据库      │  │ 日志监控    │               │
│  │ ├─MySQL    │  │ ├─日志     │               │
│  │ └─连接池   │  │ └─pprof    │               │
│  └─────────────┘  └─────────────┘               │
└─────────────────────────────────────────────────┘
        │                    │
        ▼                    ▼
  ┌──────────┐        ┌──────────┐
  │  Client  │        │ GameSrv  │
  └──────────┘        └──────────┘
```

## 目录结构

```
GlobalServer/
├── global/              # 全局服务核心
│   └── server.go       # 服务器基础实现
├── handler/            # HTTP处理器
│   └── handler.go      # API处理函数
├── http/               # HTTP服务
│   └── service.go      # Echo HTTP服务实现
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
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    account_id BIGINT NOT NULL UNIQUE,
    account_name VARCHAR(64) NOT NULL UNIQUE,
    password VARCHAR(128) NOT NULL,
    status INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_account_name (account_name)
);

-- 创建游戏服务器表
CREATE TABLE game_servers (
    id INT PRIMARY KEY AUTO_INCREMENT,
    server_id INT NOT NULL UNIQUE,
    server_name VARCHAR(64) NOT NULL,
    server_type INT DEFAULT 1,
    address VARCHAR(64) NOT NULL,
    port INT NOT NULL,
    status INT DEFAULT 1,
    online_count INT DEFAULT 0,
    max_online_count INT DEFAULT 5000,
    region VARCHAR(32),
    version VARCHAR(32),
    last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_server_id (server_id)
);
```

### 3. 编译运行
```bash
# 编译
go build -o globalserver.exe main.go

# 运行（使用默认配置）
./globalserver.exe

# 运行（指定配置文件）
./globalserver.exe /path/to/config.ini
```

### 4. 验证服务
```bash
# 健康检查
curl http://localhost:8082/health

# 创建账号
curl -X POST http://localhost:8082/api/v1/account/create \
  -H "Content-Type: application/json" \
  -d '{"account":"testuser","password":"test123"}'

# 登录
curl -X POST http://localhost:8082/api/v1/account/login \
  -H "Content-Type: application/json" \
  -d '{"account":"testuser","password":"test123"}'

# 获取服务器列表
curl http://localhost:8082/api/v1/server/list
```

## API接口文档

### 1. 健康检查
**GET** `/health`

**响应示例：**
```json
{
  "status": "ok",
  "service": "GlobalServer",
  "time": "2024-01-01T12:00:00Z"
}
```

### 2. 创建账号
**POST** `/api/v1/account/create`

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
  "success": true
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
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "servers": [
    {
      "server_id": 1,
      "server_name": "GameServer_1",
      "server_type": 1,
      "address": "127.0.0.1",
      "port": 9001,
      "status": 1,
      "online_count": 100,
      "max_online_count": 5000,
      "region": "cn-east",
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
  "servers": [
    {
      "server_id": 1,
      "server_name": "GameServer_1",
      "server_type": 1,
      "address": "127.0.0.1",
      "port": 9001,
      "status": 1,
      "online_count": 100,
      "max_online_count": 5000,
      "region": "cn-east",
      "version": "1.0.0"
    }
  ]
}
```

### 5. 服务器注册
**POST** `/api/v1/server/register`

**请求参数：**
```json
{
  "server_id": 1,
  "server_name": "GameServer_1",
  "server_type": 1,
  "address": "127.0.0.1",
  "port": 9001,
  "max_online_count": 5000,
  "region": "cn-east",
  "version": "1.0.0"
}
```

**响应示例：**
```json
{
  "success": true,
  "server_id": 1
}
```

### 6. 服务器心跳
**POST** `/api/v1/server/heartbeat`

**请求参数：**
```json
{
  "server_id": 1,
  "online_count": 100,
  "status": 1
}
```

**响应示例：**
```json
{
  "success": true
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
| listen_address | 监听地址 | 0.0.0.0:8082 |
| max_client_count | 最大客户端数 | 10000 |
| max_packet_data_size | 最大数据包大小(字节) | 1048576 |

### [log] - 日志配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| level | 日志级别 | 0(info) |
| console | 是否输出到控制台 | true |
| filename | 日志文件路径 | ./logs/server.log |
| max-size | 日志文件最大大小(MB) | 100 |
| max-days | 日志文件最大保存天数 | 15 |
| max-backups | 日志文件最大备份数 | 10 |
| compress | 是否压缩日志文件 | true |
| show-caller | 显示调用者信息 | true |
| stacktrace | 栈跟踪深度 | 3 |
| sampling | 是否启用采样 | true |
| sampling-initial | 采样初始数量 | 100 |
| sampling-thereafter | 采样后续数量 | 10 |
| async | 是否启用异步写入 | true |
| async-buffer-size | 异步缓冲区大小 | 2048 |
| async-flush-interval | 异步刷新间隔(ms) | 50 |

### [database.global] - 数据库配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| host | 数据库主机地址 | 192.168.91.128 |
| port | 数据库端口 | 3306 |
| user | 数据库用户名 | root |
| password | 数据库密码 | potato |
| dbname | 数据库名称 | global |
| driver | 数据库驱动类型 | mysql |
| max_pool_size | 连接池最大连接数 | 100 |
| min_pool_size | 连接池最小连接数 | 10 |
| connect_timeout | 连接超时时间(秒) | 30 |

### [pprof] - 性能分析配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| enabled | 是否启用pprof | false |
| listen_address | pprof监听地址 | localhost:6060 |

## 关键组件详解

### GlobalServer（全局服务器）
- 服务器生命周期管理
- 数据库连接管理
- 服务状态维护

### HTTP Service（HTTP服务）
- Echo框架HTTP服务
- 路由注册和分发
- 中间件配置（日志、恢复、CORS、限流）

### Handler（处理器）
- 账号创建和登录处理
- 服务器注册和心跳处理
- 服务器列表查询处理

### DBManager（数据库管理器）
- 数据库连接池管理
- 账号数据访问
- 服务器数据访问

## 工作流程

### 玩家登录流程
```
1. 客户端调用 /api/v1/account/login
2. GlobalServer验证账号密码
3. 生成JWT Token
4. 查询可用游戏服务器列表
5. 返回Token和服务器列表
6. 客户端选择服务器，携带Token连接Gateway
```

### 服务器注册流程
```
1. GameServer启动时调用 /api/v1/server/register
2. GlobalServer保存服务器信息到数据库
3. GameServer定期调用 /api/v1/server/heartbeat
4. GlobalServer更新服务器状态和在线人数
5. 客户端登录时获取最新的服务器列表
```

## 性能优化

### 1. HTTP服务优化
- 连接复用
- 请求限流
- 响应压缩
- 异步处理

### 2. 数据库优化
- 连接池管理
- 索引优化
- 查询优化
- 读写分离（可扩展）

### 3. 日志优化
- 异步写入
- 采样机制
- 日志轮转
- 压缩存储

## 监控指标

GlobalServer提供以下监控：
- HTTP请求QPS
- 平均响应时间
- 数据库连接数
- 活跃游戏服务器数
- 总注册账号数
- 今日登录人数

## 故障排查

### 常见问题

1. **数据库连接失败**
   - 检查数据库配置
   - 确认网络连通性
   - 检查数据库权限

2. **API请求超时**
   - 检查服务器负载
   - 查看数据库慢查询
   - 调整超时配置

3. **服务器注册失败**
   - 检查server_id是否重复
   - 确认数据库连接正常
   - 查看错误日志

4. **Token验证失败**
   - 检查密钥配置
   - 确认Token未过期
   - 验证Token格式

## 开发指南

### 添加新的API接口
1. 在`protocol`包中定义请求和响应结构
2. 在`handler/handler.go`中实现处理函数
3. 在`http/service.go`的`registerRoutes`中注册路由
4. 更新API文档

### 扩展数据库模型
1. 在`zMmoShared/db/models`中定义模型
2. 在`zMmoShared/db`中添加Repository
3. 在Handler中调用新的Repository方法

### 添加中间件
1. 在`http/service.go`中添加中间件
2. 可以使用Echo内置中间件或自定义中间件

## 版本历史

### v1.0.0
- 初始版本发布
- 支持账号创建和登录
- 支持服务器注册和心跳
- 提供服务器列表API
- 集成JWT认证

## 贡献指南

欢迎提交Issue和Pull Request！

## 许可证

MIT License
