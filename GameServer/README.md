# GameServer - 游戏服务器

## 概述

GameServer 是 MMO 游戏服务器架构的核心业务服务器，负责处理游戏逻辑、玩家数据管理、场景协调和游戏状态同步。它通过 TCP 与 GatewayServer 和 MapServer 通信，通过 etcd 进行服务注册与发现。

## 核心功能

### 1. 玩家管理
- 玩家登录/登出、角色创建/删除
- 玩家数据持久化（MySQL）
- 会话管理与多设备登录控制

### 2. 场景协调
- 地图服务协调（MapService + MapServerManager）
- 与 MapServer 的连接管理和消息路由
- 玩家地图切换

### 3. 游戏逻辑
- 战斗系统、技能系统、任务系统
- Buff 系统、副本系统
- 聊天系统、拍卖系统
- 游戏对象体系（GameObject → LivingObject → Player）

### 4. 网络通信
- 与 GatewayServer 的 TCP 连接（GatewayProxy）
- 与 MapServer 的 TCP 连接（ConnectionManager）
- Protobuf 协议编解码

### 5. 服务发现
- 基于 etcd 的服务注册与发现（zCommon/discovery）
- 心跳上报、MapServer 自动发现与连接
- 负载感知的服务器选择

## 系统架构

```
┌─────────────────────────────────────────────────┐
│                   GameServer                     │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │
│  │ 玩家管理    │  │ 场景协调    │  │ 游戏逻辑 │ │
│  │ ├─PlayerMgr │  │ ├─MapService│  │ ├─战斗   │ │
│  │ ├─Session   │  │ ├─MapServer │  │ ├─技能   │ │
│  │ └─PlayerDAO │  │   Manager  │  │ └─任务   │ │
│  └─────────────┘  └─────────────┘  └──────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │
│  │ 网络层      │  │ 数据层      │  │ 服务发现 │ │
│  │ ├─TCP      │  │ ├─MySQL    │  │ ├─etcd   │ │
│  │ ├─Protobuf │  │ ├─PlayerDAO│  │ ├─心跳   │ │
│  │ └─Router   │  │ └─Cache    │  │ └─发现   │ │
│  └─────────────┘  └─────────────┘  └──────────┘ │
└─────────────────────────────────────────────────┘
        │                    │
        ▼                    ▼
 ┌─────────────┐     ┌─────────────┐
 │  Gateway    │     │  MapServer  │
 └─────────────┘     └─────────────┘
```

## 目录结构

```
GameServer/
├── config/                    # 配置管理
│   └── config.go             # 配置结构和加载
├── connection/               # 连接管理（MapServer连接）
│   └── connection_manager.go
├── crossserver/              # 跨服消息
│   └── message.go
├── game/                     # 游戏核心
│   ├── base_server.go       # 基础服务器（生命周期管理）
│   ├── player/              # 玩家对象和管理器
│   ├── object/              # 游戏对象体系
│   ├── maps/                # 地图服务协调
│   ├── inventory/           # 背包系统
│   ├── skill/               # 技能系统
│   ├── quest/               # 任务系统
│   ├── buff/                # Buff系统
│   ├── dungeon/             # 副本系统
│   ├── chat/                # 聊天系统
│   └── auction/             # 拍卖系统
├── gateway/                  # Gateway连接
│   ├── connection_service.go
│   └── proxy/gateway_proxy.go
├── handler/                  # 消息处理器
├── health/                   # 健康检查
├── metrics/                  # Prometheus监控
├── net/                      # 网络层
│   ├── protolayer/          # Protobuf协议
│   └── service/tcp_service.go
├── services/                 # 业务服务
│   └── player_service.go
├── session/                  # 会话管理
└── version/                  # 版本信息
```

## 生命周期

GameServer 遵循统一的 `zServer.LifecycleHooks` 接口：

```
OnBeforeStart:
  ├── initServerID()          # 解析服务器ID
  ├── initDatabase()          # 初始化MySQL连接、DAO、Service
  ├── initPlayerComponents()  # 初始化玩家管理器、会话管理器
  ├── initMapComponents()     # 初始化地图服务、监控
  ├── initNetworkComponents() # 初始化TCP服务、Gateway连接
  ├── initServiceDiscovery()  # etcd注册、心跳、MapServer发现
  └── registerComponents()    # 注册到DI容器

OnAfterStart:
  ├── 启动TCP服务
  ├── 启动Metrics
  ├── 启动MapService
  ├── 启动Gateway连接
  └── 启动健康检查

OnBeforeStop:
  ├── 停止TCP服务
  ├── 停止MapService
  └── 注销服务发现
```

## 数据层架构

```
Handler → PlayerService → PlayerDAO → DBConnector → MySQL
                         ↕
                    PlayerRepository（含LRU缓存）
```

- **PlayerDAO**：同步数据库操作，使用 `QuerySync`/`ExecSync`
- **PlayerRepository**：Repository 模式，LRU 缓存热点数据
- **DBConnector**：统一数据库连接器，支持 MySQL/MongoDB

## 配置文件

| 配置段 | 说明 | 关键配置项 |
|--------|------|-----------|
| [Server] | 服务器基本配置 | ServerID, ListenAddr, MaxConnections |
| [Database] | MySQL配置 | DBHost, DBPort, DBName, DBUser |
| [Etcd] | etcd配置 | Endpoints, CACertPath |
| [Log] | 日志配置 | level, filename, max_size |
| [Metrics] | 监控配置 | Enabled, MetricsAddr |

## 快速开始

```bash
# 编译
go build -o gameserver.exe main.go

# 运行
./gameserver.exe -config config.ini
```

## 监控指标

- 在线玩家数
- 每秒处理消息数
- 数据库操作延迟
- MapServer 连接状态
- 内存使用情况
- Goroutine 数量

## 版本历史

### v1.1.0
- BaseServer 重构：拆分为独立初始化方法
- 服务发现统一：直接使用 zCommon/discovery
- PlayerDAO 同步化：移除回调模式
- 启动流程标准化

### v1.0.0
- 初始版本发布
- 支持玩家登录和角色管理
- 实现基础网络通信
- 集成数据库和缓存
