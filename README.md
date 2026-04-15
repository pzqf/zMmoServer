# zMmoServer

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

zMmoServer 是一个基于 Go 语言的分布式 MMORPG 游戏服务器，采用微服务架构设计，支持多服务器协同工作。

## 系统架构

```
                    ┌─────────────┐
                    │ GlobalServer│  账号管理、服务器列表、JWT认证
                    │  :8888 HTTP │
                    └──────┬──────┘
                           │ etcd 服务发现
              ┌────────────┼────────────┐
              │            │            │
     ┌────────▼───┐  ┌────▼───────┐    │
     │GatewayServer│  │GatewayServer│   │  1:1 配对
     │  :8001 TCP  │  │ :8002 TCP  │    │
     └──────┬──────┘  └─────┬──────┘    │
            │               │           │
     ┌──────▼──────┐  ┌─────▼──────┐   │
     │ GameServer  │  │ GameServer │   │  玩家数据、逻辑调度
     │ :20001 TCP  │  │ :20002 TCP │   │
     └──────┬──────┘  └─────┬──────┘   │
            │               │           │
     ┌──────▼──────┐  ┌─────▼──────┐   │
     │ MapServer   │  │ MapServer  │   │  地图实例、战斗、AI
     │ :30001 TCP  │  │ :30002 TCP │   │
     └─────────────┘  └────────────┘   │
                                       │
     ┌─────────────┐  ┌──────────────┐ │
     │ MapServer   │  │ MapServer    │ │  跨服地图（cross_group）
     │ :30003 TCP  │  │ :30004 TCP   │ │  多服共享
     │ (mirror)    │  │ (cross_group)│ │
     └─────────────┘  └──────────────┘ │
```

### 服务器组件

| 服务器                  | 职责                                                         | 成熟度 |
| ----------------------- | ------------------------------------------------------------ | ------ |
| **GlobalServer**  | 账号注册/登录、JWT认证、服务器列表管理、服务发现中枢         | 中高   |
| **GatewayServer** | 客户端TCP连接管理、消息路由转发、JWT验证、DDoS防护、防作弊   | 中高   |
| **GameServer**    | 玩家数据管理、游戏逻辑调度、地图服务协调、Outbox/Inbox一致性 | 中     |
| **MapServer**     | 地图实例管理、战斗计算、AI状态机、技能系统、经济系统         | 中     |
| **AdminServer**   | 后台管理、监控（暂未实现）                                   | -      |

### 三种地图模式

| 模式              | 说明                   | 配置文件          |
| ----------------- | ---------------------- | ----------------- |
| `single_server` | 单服地图，仅本服玩家   | config_single.ini |
| `mirror`        | 镜像地图，各服独立副本 | config_mirror.ini |
| `cross_group`   | 世界地图，多服玩家共享 | config_cross.ini  |

## 技术栈

| 类别               | 技术                            |
| ------------------ | ------------------------------- |
| **语言**     | Go 1.25+                        |
| **网络**     | zEngine/zNet (TCP + Protobuf)   |
| **日志**     | zEngine/zLog (zap + lumberjack) |
| **服务发现** | etcd                            |
| **数据库**   | MySQL (go-sql-driver/mysql)     |
| **缓存**     | Redis (go-redis/v9)             |
| **序列化**   | Protocol Buffers (v1.36.10)     |
| **监控**     | Prometheus                      |
| **容器编排** | Kubernetes                      |
| **框架**     | zEngine + zUtil                 |

## 项目结构

```
zMmoServer/
├── GlobalServer/        # 全局服（9文件/1808行）
│   ├── global/          # 核心服务器（BaseServer 生命周期管理）
│   ├── handler/         # HTTP 请求处理（账号/登录/服务器列表）
│   ├── http/            # Echo HTTP 服务
│   ├── db/              # 数据库服务
│   ├── gameserverlist/  # 服务器列表管理器
│   └── config/          # 配置加载
├── GatewayServer/       # 网关服（17文件/1719行）
│   ├── gateway/         # 核心服务器
│   ├── client/          # 客户端服务（连接/认证/安全）
│   ├── proxy/           # GameServer 代理
│   ├── gameserver/      # GameServer 连接
│   └── config/          # 配置加载
├── GameServer/          # 游戏服（48文件/10041行）
│   ├── game/            # 游戏逻辑
│   │   ├── player/      # 玩家对象（Actor模型）、管理器
│   │   ├── object/      # 游戏对象体系（GameObject→LivingObject）
│   │   ├── maps/        # 地图服务（MapService + Outbox/Inbox）
│   │   ├── inventory/   # 背包系统
│   │   ├── skill/       # 技能系统
│   │   ├── quest/       # 任务系统
│   │   ├── buff/        # Buff系统
│   │   ├── dungeon/     # 副本系统
│   │   ├── chat/        # 聊天系统
│   │   └── auction/     # 拍卖系统
│   ├── connection/      # 连接管理
│   ├── session/         # 会话管理
│   ├── services/        # 玩家服务（数据库操作）
│   └── gateway/         # Gateway 连接
├── MapServer/           # 地图服（36文件/8189行）
│   ├── server/          # 核心服务器
│   ├── maps/            # 地图核心
│   │   ├── ai/          # AI 状态机（巡逻/追击/攻击/逃跑）
│   │   ├── combat/      # 战斗系统（物理/魔法/真实伤害）
│   │   ├── skill/       # 技能系统（含连招 ComboManager）
│   │   ├── buff/        # Buff 管理
│   │   ├── economy/     # 经济系统（交易/拍卖/商店/货币）
│   │   ├── object/      # 地图对象（Player/Monster/NPC/Item）
│   │   ├── event/       # 事件系统
│   │   ├── dungeon/     # 副本系统
│   │   ├── item/        # 物品管理
│   │   └── task/        # 任务系统
│   └── connection/      # 连接管理
├── zCommon/             # 共享公共库（132文件/19432行）
│   ├── common/id/       # ID 类型定义（20+ 类型化 ID）+ Snowflake
│   ├── config/          # 配置表系统（Excel 加载 + 热更新）
│   ├── consistency/     # 一致性保障（Outbox/Inbox + 事务管理器）
│   ├── crossserver/     # 跨服务器消息（Envelope + RPC + 迁移）
│   ├── db/              # 数据库层（DAO/Repository/Models/Connector）
│   ├── discovery/       # 统一服务发现（etcd）
│   ├── health/          # 健康检查框架
│   ├── metrics/         # 统一 Prometheus 指标
│   ├── net/             # 网络层（Protobuf 协议、路由）
│   ├── pool/            # 对象池（Packet/Event/ByteSlice/TypedPool）
│   ├── protocol/        # Protobuf 生成代码
│   ├── aoi/             # AOI 系统（Grid AOI）
│   ├── parallel/        # 分区并行调度
│   ├── connpool/        # 连接池（RoundRobin + 健康检查）
│   ├── monitor/         # 内存监控 + 告警
│   ├── stresstest/      # 压力测试框架
│   ├── gameevent/       # 游戏事件框架
│   ├── lifecycle/       # 对象生命周期管理
│   └── ...              # 其他共享模块
├── resources/           # 资源文件
│   ├── excel_tables/    # 配置表（15个 Excel 文件）
│   ├── maps/            # 地图文件（25个 JSON 文件）
│   ├── protocol/        # Proto 文件
│   └── etcd/            # etcd TLS 证书
├── kubernetes/          # K8s 部署配置
└── docs/                # 项目文档
```

## 核心设计

### 统一生命周期

所有服务器实现 `zServer.LifecycleHooks` 接口，启动流程标准化：

```
flag解析 → 配置加载 → 日志初始化 → 服务器创建 → Run()
  ├── OnBeforeStart: 拆分为独立 init* 方法
  ├── OnAfterStart:  启动各服务
  └── OnBeforeStop:  优雅关闭
```

状态转换：`Starting` → `Initializing` → `Ready` → `Healthy` → `Draining` → `Stopped`

### ID 体系

- **ServerID**: 6位语义 ID（GroupID(4位) + ServerIndex(2位)），如 `101` = Group `0001` + Index `01`
- **运行时 ID**: Snowflake 生成（PlayerId/ObjectId 等）
- **配置驱动 ID**: int32（MapId/SkillId/QuestId 等）
- **20+ 类型化 ID**: AccountIdType/PlayerIdType/ItemIdType/SkillIdType/MapIdType/GuildIdType/TeamIdType/PetIdType/MountIdType/AchievementIdType 等

### 数据层架构

```
Handler → Service → DAO → DBConnector → MySQL
                  ↕
             Repository（LRU缓存）
```

- **DAO**：同步数据库操作（`QuerySync`/`ExecSync`），核心3个已同步化
- **Repository**：Repository 模式，LRU 缓存热点数据，已移除 Async 方法
- **DBConnector**：统一数据库连接器，支持 MySQL/MongoDB

### 跨服务器通信

- **Envelope 协议**: Magic `0x5A4D4F4F`，40字节元数据头
- **消息路由**: CrossRouter/ServerRouter/RequestRouter
- **一致性保障**: Outbox/Inbox 模式 + 事务管理器
- **跨服 RPC**: RPCEndpoint + RPCService 封装

### 配置文件规范

- 所有 INI 文件使用统一的大写驼峰命名
- 每个配置项上方有中文注释说明
- 支持 `{ServerID}` 占位符
- 所有服务器包含统一的 Server/Log/Metrics/Pprof/Etcd 配置段

## 开发进度

### 已完成

- [X] 四服务器框架搭建（Global/Gateway/Game/Map）
- [X] 统一生命周期管理（BaseServer 重构，独立 init* 方法）
- [X] 统一启动流程（flag→config→log→server）
- [X] 网络通信模块（TCP + Protobuf）
- [X] 服务发现与注册（etcd，统一使用 zCommon/discovery）
- [X] 统一配置文件管理（INI + Excel 配置表 + 热更新）
- [X] 账号注册/登录（JWT + SHA256）
- [X] 服务器列表管理（静态MySQL + 动态etcd合并）
- [X] DDoS 防护（无锁设计）
- [X] 防作弊检测（IP管理、行为频率统计）
- [X] Prometheus 监控指标（各服务器统一）
- [X] 健康检查框架
- [X] Actor 并发模型（玩家对象）
- [X] 游戏对象体系（GameObject → LivingObject → Player）
- [X] 战斗系统（物理/魔法/真实伤害，暴击判定）
- [X] AI 状态机（Idle/Patrol/Chase/Attack/Flee/Return/Dead）
- [X] 技能系统（含连招 ComboManager）
- [X] 经济系统（交易/拍卖/商店/货币）
- [X] 统一通信模式（BaseMessage/CrossServerMessage/Envelope）
- [X] 数据一致性机制（Outbox/Inbox + 事务管理器）
- [X] ID 类型规范（20+ 类型化 ID + Snowflake）
- [X] DAO 同步化（PlayerDAO/AccountDAO/GameServerDAO）
- [X] Repository 精简（移除 Async 方法，新增 GameServerRepository）
- [X] 代码冗余消除（统一 metrics/health/discovery/config/container/utils/request）
- [X] 对象池扩展（Event/ByteSlice/SizedBytePool/TypedPool）
- [X] Map分区并行（PartitionScheduler）
- [X] Gateway多连接池（ConnectionPool + RoundRobin）
- [X] 内存监控 + 告警
- [X] 压力测试框架

### 待开发

- [ ] 剩余8个 DAO 同步化（auction/login_log/player_buff 等）
- [ ] AOI 系统集成到 MapServer
- [ ] 玩家移动同步完整流程
- [ ] 技能释放完整流程
- [ ] 怪物生成与 AI 集成
- [ ] GameServer 与 MapServer 职责边界明确化
- [ ] GatewayServer 多 GameServer 负载均衡
- [ ] 背包/物品/装备完整实现
- [ ] 技能效果完整实现（Buff/Debuff/DoT/HoT）
- [ ] 任务系统完整实现
- [ ] 副本系统完整实现
- [ ] 跨服地图功能验证
- [ ] 数据一致性压力测试
- [ ] 单元测试补充（核心功能 > 80% 覆盖率）
- [ ] AdminServer 实现

## 快速开始

### 编译

```bash
# 编译所有服务器
cd zMmoServer
go build -o bin/global_server.exe  ./GlobalServer/main.go
go build -o bin/gateway_server.exe ./GatewayServer/main.go
go build -o bin/game_server.exe    ./GameServer/main.go
go build -o bin/map_server.exe     ./MapServer/main.go
```

### 运行

1. 启动 etcd 服务
2. 启动 MySQL
3. 按顺序启动：GlobalServer → GatewayServer → GameServer → MapServer

### 配置文件

| 服务器        | 配置文件          | 说明                                             |
| ------------- | ----------------- | ------------------------------------------------ |
| GlobalServer  | config.ini        | HTTP/Database/Redis/Etcd/Metrics/Pprof           |
| GatewayServer | config.ini        | TCP/Security/DDoS/Compression/Etcd/Metrics/Pprof |
| GameServer    | config.ini        | TCP/Database/Etcd/Metrics/Pprof                  |
| MapServer     | config_single.ini | 单服地图模式                                     |
| MapServer     | config_mirror.ini | 镜像地图模式                                     |
| MapServer     | config_cross.ini  | 世界地图模式                                     |
| MapServer     | config_test.ini   | 测试配置                                         |

## 许可证

MIT License

---

*最后更新: 2026-04-15*
