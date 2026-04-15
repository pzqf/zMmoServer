# MapServer - 地图服务器

## 概述

MapServer 是 MMO 游戏服务器架构的地图实例服务器，负责地图逻辑、战斗计算、AI 状态机、经济系统和场景管理。它接收来自 GameServer 的请求，处理地图内的所有实时逻辑。

## 核心功能

### 1. 地图管理
- 地图实例创建与销毁
- 三种地图模式：单服（single_server）、镜像（mirror）、跨服（cross_group）
- 地图资源加载（JSON 格式）
- 玩家进出地图管理

### 2. 战斗系统
- 物理/魔法/真实伤害计算
- 暴击判定
- 技能释放与连招（ComboManager）
- Buff/Debuff 管理

### 3. AI 系统
- 状态机驱动（Idle/Patrol/Chase/Attack/Flee/Return/Dead）
- 怪物生成管理（SpawnManager）
- NPC 交互

### 4. 经济系统
- 交易系统
- 拍卖系统
- 商店系统
- 货币管理

### 5. 任务系统
- 任务配置加载
- 任务状态管理

### 6. 网络通信
- TCP 服务（与 GameServer 通信）
- Protobuf 协议编解码
- 消息去重与超时管理

## 系统架构

```
┌─────────────────────────────────────────────────┐
│                   MapServer                      │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │
│  │ 地图管理    │  │ 战斗系统    │  │ AI系统   │ │
│  │ ├─MapManager│  │ ├─CombatSys │  │ ├─AI Mgr │ │
│  │ ├─Map实例   │  │ ├─SkillMgr  │  │ ├─Spawn  │ │
│  │ └─资源加载  │  │ └─BuffMgr   │  │ └─NPC    │ │
│  └─────────────┘  └─────────────┘  └──────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │
│  │ 经济系统    │  │ 任务系统    │  │ 网络层   │ │
│  │ ├─交易     │  │ ├─TaskMgr  │  │ ├─TCP    │ │
│  │ ├─拍卖     │  │ └─TaskCfg  │  │ ├─Protobuf│ │
│  │ ├─商店     │  │            │  │ └─去重   │ │
│  │ └─货币     │  │            │  │          │ │
│  └─────────────┘  └─────────────┘  └──────────┘ │
└─────────────────────────────────────────────────┘
        │
        ▼
 ┌─────────────┐
 │  GameServer  │
 └─────────────┘
```

## 目录结构

```
MapServer/
├── config/                # 配置管理（多模式配置）
│   └── config.go
├── connection/            # 连接管理
│   └── connection_manager.go
├── maps/                  # 地图核心
│   ├── map.go            # 地图实例
│   ├── map_manager.go    # 地图管理器
│   ├── map_resource_loader.go
│   ├── player_game_server_manager.go
│   ├── spawn_manager.go  # 怪物生成
│   ├── ai/              # AI 状态机
│   ├── combat/          # 战斗系统
│   ├── skill/           # 技能系统（含连招）
│   ├── buff/            # Buff 管理
│   ├── economy/         # 经济系统（交易/拍卖/商店/货币）
│   ├── object/          # 地图对象（Player/Monster/NPC/Item）
│   ├── event/           # 事件系统
│   ├── dungeon/         # 副本系统
│   ├── item/            # 物品管理
│   └── task/            # 任务系统
├── server/               # 核心服务器
│   └── base_server.go
├── health/               # 健康检查
├── metrics/              # Prometheus 监控
├── net/service/          # TCP 网络服务
├── auth/                 # 防作弊
└── version/              # 版本信息
```

## 生命周期

MapServer 遵循统一的 `zServer.LifecycleHooks` 接口：

```
OnBeforeStart:
  ├── 解析服务器ID
  ├── initComponents()          # 初始化容器、Metrics、健康检查、
  │                             # 去重管理、超时管理、连接管理、
  │                             # 地图管理器、TCP服务、服务发现
  └── 注册组件到容器

OnAfterStart:
  ├── 启动TCP服务
  ├── 启动Metrics
  └── 启动健康检查

OnBeforeStop:
  ├── 停止TCP服务
  └── 注销服务发现
```

## 三种地图模式

| 模式 | 配置文件 | 说明 |
|------|----------|------|
| single_server | config_single.ini | 单服地图，仅本服玩家 |
| mirror | config_mirror.ini | 镜像地图，各服独立副本 |
| cross_group | config_cross.ini | 世界地图，多服玩家共享 |

## 配置文件

| 配置段 | 说明 | 关键配置项 |
|--------|------|-----------|
| [Server] | 服务器基本配置 | ServerID, ListenAddr, MaxConnections |
| [Map] | 地图配置 | MapMode, MapIDs |
| [Etcd] | etcd配置 | Endpoints, CACertPath |
| [Log] | 日志配置 | level, filename, max_size |
| [Metrics] | 监控配置 | Enabled, MetricsAddr |

## 快速开始

```bash
# 编译
go build -o mapserver.exe main.go

# 运行（单服模式）
./mapserver.exe -config config_single.ini

# 运行（镜像模式）
./mapserver.exe -config config_mirror.ini

# 运行（跨服模式）
./mapserver.exe -config config_cross.ini
```

## 监控指标

- 地图实例数量
- 在线玩家数（按地图）
- 怪物数量
- 战斗事件频率
- AI 决策延迟
- 内存使用情况

## 版本历史

### v1.1.0
- BaseServer 重构：拆分为独立初始化方法
- 启动流程标准化

### v1.0.0
- 初始版本发布
- 支持三种地图模式
- 实现战斗系统
- 实现 AI 状态机
- 实现经济系统
