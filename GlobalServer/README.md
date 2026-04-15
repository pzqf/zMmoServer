# GlobalServer - 全局服务器

## 概述

GlobalServer 是 MMO 游戏服务器架构的中心管理服务，负责账号管理、服务器列表维护、服务器注册和心跳监控。它提供 HTTP API 接口，是客户端登录和服务器管理的入口。

## 核心功能

### 1. 账号管理
- 账号创建与登录
- JWT 认证（HS256 签名，7天有效期）
- 登录状态记录

### 2. 服务器管理
- 游戏服务器注册（etcd 动态发现）
- 服务器列表查询（MySQL 静态 + etcd 动态合并）
- 心跳监控与自动清理

### 3. HTTP API 服务
- RESTful API（Echo 框架）
- 健康检查端点
- 请求限流保护
- CORS 支持

### 4. 数据持久化
- MySQL 数据库（账号、服务器信息）
- 连接池管理
- Repository 模式（含 LRU 缓存）

### 5. 监控系统
- Prometheus 指标集成
- 业务指标（账号、服务器、数据库操作）
- 健康检查框架

## 系统架构

```
┌─────────────────────────────────────────────────┐
│                 GlobalServer                     │
├─────────────────────────────────────────────────┤
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐│
│ │ HTTP API    │ │ 账号管理    │ │ 服务器管理  ││
│ │├─/health    │ │├─创建       │ │├─注册       ││
│ │├─/api/v1    │ │├─登录       │ │├─心跳       ││
│ │└─限流       │ │└─Token      │ │└─列表       ││
│ └─────────────┘ └─────────────┘ └─────────────┘│
│ ┌─────────────┐ ┌─────────────┐                │
│ │ 数据存储    │ │ 服务发现    │                │
│ │├─MySQL      │ │├─etcd注册   │                │
│ │├─连接池     │ │├─心跳上报   │                │
│ │└─LRU缓存    │ │└─Watch      │                │
│ └─────────────┘ └─────────────┘                │
└─────────────────────────────────────────────────┘
```

## 目录结构

```
GlobalServer/
├── global/              # 全局服务核心
│   └── base_server.go   # 服务器基础实现（生命周期管理）
├── handler/            # HTTP处理器
│   └── handler.go      # API处理函数
├── http/               # HTTP服务
│   └── http_service.go # Echo HTTP服务实现
├── db/                 # 数据库服务
│   └── service.go      # 数据库服务实现
├── gameserverlist/     # 服务器列表管理器
├── config/             # 配置管理
├── metrics/            # 监控指标
└── version/            # 版本信息
```

## 生命周期

GlobalServer 遵循统一的 `zServer.LifecycleHooks` 接口：

```
OnBeforeStart:
  ├── initHealthManager()      # 初始化健康检查管理器
  ├── initDatabase()           # 初始化DBManager、ID生成器
  ├── initMetrics()            # 初始化Prometheus指标
  ├── initHTTPService()        # 初始化Echo HTTP服务
  ├── initServiceDiscovery()   # etcd注册、ServerListManager、心跳
  └── 注册组件

OnAfterStart:
  ├── 启动DB服务
  ├── 启动Metrics服务
  └── 启动HTTP服务

OnBeforeStop:
  ├── 停止HTTP服务
  ├── 停止DB服务
  └── 关闭DBManager
```

## 数据层架构

```
Handler → Repository → DAO → DBConnector → MySQL
            ↕
         LRU缓存
```

- **AccountDAO**：同步数据库操作
- **GameServerDAO**：同步数据库操作
- **AccountRepository**：LRU 缓存热点数据
- **GameServerRepository**：LRU 缓存热点数据
- **DBConnector**：统一数据库连接器

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| POST | `/api/v1/account/create` | 创建账号 |
| POST | `/api/v1/account/login` | 账号登录 |
| GET | `/api/v1/server/list` | 获取服务器列表 |

### 创建账号

```bash
curl -X POST http://localhost:8888/api/v1/account/create \
  -H "Content-Type: application/json" \
  -d '{"account":"testuser","password":"test123"}'
```

### 账号登录

```bash
curl -X POST http://localhost:8888/api/v1/account/login \
  -H "Content-Type: application/json" \
  -d '{"account":"testuser","password":"test123"}'
```

## 配置文件

| 配置段 | 说明 | 关键配置项 |
|--------|------|-----------|
| [server] | 服务器基本配置 | server_id, worker_id, datacenter_id |
| [http] | HTTP服务配置 | listen_address, max_client_count |
| [database.global] | 数据库配置 | host, port, user, dbname, max_pool_size |
| [etcd] | etcd配置 | Endpoints, CACertPath |
| [log] | 日志配置 | level, filename, max_size |
| [metrics] | 监控配置 | enabled, listen_address |
| [pprof] | 性能分析配置 | enabled, listen_address |

## 快速开始

```bash
# 编译
go build -o globalserver.exe main.go

# 运行
./globalserver.exe -config config.ini
```

## 监控指标

- 账号注册/登录次数
- 服务器列表查询次数
- 数据库操作延迟
- HTTP 请求统计
- 运行时间

## 版本历史

### v1.1.0
- BaseServer 重构：拆分为独立初始化方法
- DAO 同步化：AccountDAO/GameServerDAO 移除回调模式
- GameServerRepository 新增（含LRU缓存）
- JWT 依赖清理（统一使用 v5）
- 启动流程标准化

### v1.0.0
- 初始版本发布
- 账号注册/登录
- 服务器列表管理
- JWT 认证
- Prometheus 监控
