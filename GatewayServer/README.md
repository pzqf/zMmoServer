# GatewayServer - 网关服务器

## 概述

GatewayServer 是 MMO 游戏服务器架构的接入层，负责处理客户端 TCP 连接、安全验证、数据转发和负载均衡。它是客户端与游戏服务器之间的桥梁，提供统一的接入点。

## 核心功能

### 1. 客户端连接管理
- TCP 连接处理（支持高并发，默认最大 10000 连接）
- 连接状态监控与心跳检测
- 工作池模式（WorkerPool）提高并发性能

### 2. 安全防护
- **DDoS 防护**：IP 连接频率限制、数据包频率限制、流量限制、自动封禁
- **Token 验证**：JWT 令牌验证机制
- **IP 封禁**：使用 etcd 存储封禁列表，支持实时同步和自动过期
- **防作弊**：行为统计、异常检测、自动处理

### 3. 数据转发
- 协议解析与路由转发
- Snappy 数据压缩
- GameServer 代理（GameServerProxy）

### 4. 服务发现与管理
- 基于 etcd 的服务注册与发现（zCommon/discovery）
- 心跳机制与服务上报
- GameServer 连接管理

## 系统架构

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ TCP
       ▼
┌──────────────────┐
│   GatewayServer  │
│  ├─ 连接管理     │
│  │  ├─ 会话管理   │
│  │  └─ 工作池     │
│  ├─ 安全系统     │
│  │  ├─ DDoS防护  │
│  │  ├─ IP封禁    │
│  │  └─ 防作弊    │
│  ├─ 认证系统     │
│  │  ├─ Token管理 │
│  │  └─ 认证处理   │
│  ├─ 消息处理     │
│  │  ├─ 消息解析   │
│  │  └─ 消息转发   │
│  └─ 服务管理     │
│     ├─ etcd发现   │
│     └─ 心跳上报   │
└──────┬───────────┘
       │ TCP
       ▼
┌──────────────────┐
│   GameServer     │
└──────────────────┘
```

## 目录结构

```
GatewayServer/
├── client/                # 客户端相关模块
│   ├── auth/              # 认证管理（Token管理、认证处理器）
│   ├── connection/        # 连接管理（连接管理器、事件处理）
│   ├── security/          # 安全管理（防作弊、IP封禁）
│   ├── message_handler.go # 消息处理器
│   └── service.go         # 客户端服务
├── common/                # 共享接口
├── config/                # 配置管理
├── gameserver/            # GameServer连接
│   └── connection_service.go
├── gateway/               # 网关核心
│   └── base_server.go     # 基础服务器实现
├── metrics/               # 监控指标
├── proxy/                 # 代理服务
│   └── game_server_proxy.go
├── report/                # 服务上报
└── version/               # 版本信息
```

## 生命周期

GatewayServer 遵循统一的 `zServer.LifecycleHooks` 接口：

```
OnBeforeStart:
  ├── initServerID()       # 解析服务器ID
  ├── initNetServer()      # 初始化TCP网络服务
  ├── initEtcdClient()     # 初始化etcd客户端
  ├── initServices()       # 初始化客户端服务、GameServer连接、上报服务
  └── registerComponents() # 注册到DI容器

OnAfterStart:
  ├── 启动TCP服务
  ├── 启动Metrics
  └── 启动服务上报

OnBeforeStop:
  ├── 停止TCP服务
  ├── 关闭etcd连接
  └── 注销服务发现
```

## 配置文件

| 配置段 | 说明 | 关键配置项 |
|--------|------|-----------|
| [Server] | 服务器基本配置 | ServerID, ListenAddr, MaxConnections, UseWorkerPool |
| [Security] | 安全配置 | TokenExpiry, MaxLoginAttempts, BanDuration |
| [GameServer] | GameServer配置 | GameServerID, GameServerAddr |
| [Etcd] | etcd配置 | Endpoints, CACertPath |
| [ddos] | DDoS防护配置 | max_conn_per_ip, max_packets_per_ip, ban_duration |
| [compression] | 压缩配置 | enabled, threshold |
| [Log] | 日志配置 | level, filename, max_size |
| [Metrics] | 监控配置 | Enabled, MetricsAddr |

## 快速开始

```bash
# 编译
go build -o gatewayserver.exe main.go

# 运行
./gatewayserver.exe -config config.ini
```

## 监控指标

- 当前连接数
- 每秒请求数（QPS）
- 平均响应时间
- DDoS 拦截次数
- 压缩比率
- 工作池使用情况
- IP 封禁数量
- 作弊检测次数

## 版本历史

### v1.3.0
- BaseServer 重构：拆分为独立初始化方法
- 服务发现统一：直接使用 zCommon/discovery
- 启动流程标准化

### v1.2.0
- 重构客户端模块，采用模块化设计
- 实现etcd存储IP封禁
- 添加防作弊系统
- 优化服务发现机制

### v1.1.0
- 添加工作池模式支持
- 优化连接管理（使用zMap.TypedMap）
- 改进心跳上报机制

### v1.0.0
- 初始版本发布
- 支持基础连接管理
- 实现DDoS防护
- 支持数据压缩
- 集成JWT验证
