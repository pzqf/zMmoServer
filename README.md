# zMmoServer

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat\&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

zMmoServer 是一个基于 Go 语言的分布式 MMORPG 游戏服务器，采用微服务架构设计，支持多服务器协同工作。
它建立在游戏引擎 [zEngine](https://github.com/pzqf/zEngine) 与工具库 [zUtil](https://github.com/pzqf/zUtil) 之上，
既是一套**可运行的分布式服务端**，也是**学习"MMO 服务端怎么分层、跨服怎么通信、状态怎么保证一致"的参考实现**。

## 项目定位与适用范围

**它是什么**：一套四层分布式游戏服务端（账号/网关/游戏/地图）的**参考实现 + 骨架**，把跨服通信、
AOI 视野、Actor 单写者、Outbox/Inbox 一致性、服务发现、优雅生命周期这些"难做对"的部分做出了可运行、
有测试/E2E 背书的样板。业务层刻意保持最小（够验证架构即可，不堆数值玩法）。

**适合谁**：想学/借鉴分布式游戏服务端架构的开发者；想在一套现成骨架上长自己玩法的团队；
想研究"跨进程一致性/AOI/单写者 Actor 在 Go 里怎么落地"的人。

**不适合**：开箱即用的成品游戏（业务/玩法是最小闭环，需要自行扩建）；不想碰 etcd/MySQL 的轻量场景；
把它当"引擎"用——引擎在 [zEngine](https://github.com/pzqf/zEngine)，本仓是长在引擎之上的服务端。

**成熟度**：版本 `0.0.x`，成熟前不发 `1.0`。**状态以证据为准**——每条"完成"挂一个通过的测试或跨进程真
E2E，权威清单见 [`docs/项目评估报告.md`](docs/项目评估报告.md)。文档导航见 [`docs/README.md`](docs/README.md)。

## 系统架构

```
                    ┌─────────────┐
                    │ GlobalServer│  账号管理、服务器列表、JWT认证
                    │  :8888 HTTP │
                    └──────┬──────┘
                           │ etcd 服务发现
              ┌────────────┼────────────┐
              │            │            │
     ┌────────▼───┐  ┌────▼───────┐     │
     │GatewayServer│  │GatewayServer│   │  1:1 配对
     │  :8001 TCP  │  │ :8002 TCP  │    │
     └──────┬──────┘  └─────┬──────┘    │
            │               │           │
     ┌──────▼──────┐  ┌─────▼──────┐    │
     │ GameServer  │  │ GameServer │    │  玩家数据、逻辑调度
     │ :20001 TCP  │  │ :20002 TCP │    │
     └──────┬──────┘  └─────┬──────┘    │
            │               │           │
     ┌──────▼──────┐  ┌─────▼──────┐    │
     │ MapServer   │  │ MapServer  │    │  地图实例、战斗、AI
     │ :30001 TCP  │  │ :30002 TCP │    │
     └─────────────┘  └────────────┘    │
                                        │
     ┌─────────────┐  ┌──────────────┐  │
     │ MapServer   │  │ MapServer    │  │  跨服地图（cross_group）
     │ :30003 TCP  │  │ :30004 TCP   │  │  多服共享
     │ (mirror)    │  │ (cross_group)│  │
     └─────────────┘  └──────────────┘  │
```

### 服务器组件

| 服务器               | 职责                                   | 成熟度 |
| ----------------- | ------------------------------------ | --- |
| **GlobalServer**  | 账号注册/登录、JWT认证、服务器列表管理、服务发现中枢         | 中高  |
| **GatewayServer** | 客户端TCP连接管理、消息路由转发、JWT验证、DDoS防护、防作弊   | 中高  |
| **GameServer**    | 玩家数据管理、游戏逻辑调度、地图服务协调、Outbox/Inbox一致性 | 中   |
| **MapServer**     | 地图实例管理、战斗计算、AI状态机、技能系统、经济系统          | 中   |
| **AdminServer**   | 后台管理、监控（暂未实现）                        | -   |

### 三种地图模式

| 模式              | 说明          | 配置文件               |
| --------------- | ----------- | ------------------ |
| `single_server` | 单服地图，仅本服玩家  | config\_single.ini |
| `mirror`        | 镜像地图，各服独立副本 | config\_mirror.ini |
| `cross_group`   | 世界地图，多服玩家共享 | config\_cross.ini  |

## 技术栈

| 类别       | 技术                              |
| -------- | ------------------------------- |
| **语言**   | Go 1.25+                        |
| **网络**   | zEngine/zNet (TCP + Protobuf)   |
| **日志**   | zEngine/zLog (zap + lumberjack) |
| **服务发现** | etcd                            |
| **数据库**  | MySQL (go-sql-driver/mysql)     |
| **缓存**   | Redis (go-redis/v9)             |
| **序列化**  | Protocol Buffers (v1.36.10)     |
| **监控**   | Prometheus                      |
| **容器编排** | Kubernetes                      |
| **框架**   | zEngine + zUtil                 |

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
│   │   ├── loot/        # 掉落系统（配置表驱动）
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

## 成熟度与验证（真相优先）

> **状态以证据为准，不以主观打分为准。** 本项目正经历一轮**成熟化改造**（Phase 0–4），
> 目标 = 上线级 + 可复用开源框架。权威状态见：
> - [`docs/项目评估报告.md`](docs/项目评估报告.md) — **能力 → 证据**清单（每条完成挂一个通过的测试或跨进程真 E2E）。
> - [`docs/成熟化改造-执行计划.md`](docs/成熟化改造-执行计划.md) — 进度台账（勾选框 = 进度真相）。
>
> 改造已达成（均有测试/E2E 背书）：Game↔Map 跨服断链修复 + 全 protobuf 契约、attack 真实伤害闭环、
> **AOI 视野回程**（曾潜伏 5 个真实缺陷）、Actor 监督（panic 恢复）、断线自动重连、连接去重、
> zMetrics 接入 zNet、**Outbox/Inbox 持久化**（进程重启不丢在途消息，真 MySQL 验证）、
> dispatcher 崩溃隔离等。8 模块 `build+vet` 全绿。
>
> ⚠️ 下方「能力清单」是**功能盘点**，不等同于"已充分验证"——许多能力在改造前是"实现了但从没真跑过"
> 的潜伏状态，其真实可用性以上述评估报告的证据为准。

### 能力清单（功能盘点）

- [x] 四服务器框架搭建（Global/Gateway/Game/Map）
- [x] 统一生命周期管理（BaseServer 重构，独立 init\* 方法）
- [x] 统一启动流程（flag→config→log→server）
- [x] 网络通信模块（TCP + Protobuf）
- [x] 服务发现与注册（etcd，统一使用 zCommon/discovery）
- [x] 统一配置文件管理（INI + Excel 配置表 + 热更新）
- [x] 账号注册/登录（JWT + SHA256）
- [x] 服务器列表管理（静态MySQL + 动态etcd合并）
- [x] DDoS 防护（无锁设计）
- [x] 防作弊检测（IP管理、行为频率统计）
- [x] Prometheus 监控指标（各服务器统一）
- [x] 健康检查框架
- [x] Actor 并发模型（玩家对象）
- [x] 游戏对象体系（GameObject → LivingObject → Player）
- [x] 战斗系统（物理/魔法/真实伤害，暴击判定）
- [x] AI 状态机（Idle/Patrol/Chase/Attack/Flee/Return/Dead）
- [x] 技能系统（含连招 ComboManager）
- [x] 经济系统（交易/拍卖/商店/货币）
- [x] 统一通信模式（BaseMessage/CrossServerMessage/Envelope）
- [x] 数据一致性机制（Outbox/Inbox + 事务管理器）
- [x] ID 类型规范（20+ 类型化 ID + Snowflake）
- [x] DAO 同步化（PlayerDAO/AccountDAO/GameServerDAO）
- [x] Repository 精简（移除 Async 方法，新增 GameServerRepository）
- [x] 代码冗余消除（统一 metrics/health/discovery/config/container/utils/request）
- [x] 对象池扩展（Event/ByteSlice/SizedBytePool/TypedPool）
- [x] Map分区并行（PartitionScheduler）
- [x] Gateway多连接池（ConnectionPool + RoundRobin）
- [x] 内存监控 + 告警
- [x] 压力测试框架
- [x] AOI 系统集成到 MapServer（Grid AOI + 视野推送）
- [x] 玩家移动同步完整流程（AOI事件驱动 + 客户端推送）
- [x] 技能释放完整流程（技能配置→伤害计算→效果施加→冷却管理）
- [x] 怪物生成与 AI 集成（PlayerQuerier接口 + 配置表驱动）
- [x] GameServer 与 MapServer 职责边界明确化（Actor消息路由 + 会话绑定）
- [x] 技能效果完整实现（伤害/治疗/Buff增益/控制Debuff）
- [x] Buff 属性修正系统（攻击/防御/HP/MP/速度修正 + DoT/HoT处理）
- [x] 交易结算逻辑（CompleteTrade/completeAuction 货币转移）
- [x] 怪物重生机制（配置表驱动 + 定时重生）
- [x] 掉落系统（LootGroup配置表 + 掉落率计算 + 物品生成）
- [x] 装备属性加成接入战斗计算（Equipment属性汇总 + CombatSystem集成）
- [x] 游戏主循环（MapServer 100ms Tick + AI/Buff/Player/Skill/Event 更新）
- [x] Gateway 健康状态联动（GameServer连接状态 → Gateway健康状态）
- [x] Gateway etcd TLS 支持
- [x] zUtil/zConfig 配置工具函数统一（GetStringWithDefault/GetEnv等）
- [x] GlobalServer 数据库连接复用（DBService复用DBManager连接器）

#### 业务层建设（2026-07-25，定向放开"最小闭环"，均有测试/E2E 背书）

> 原则仍是"够验证架构即可、不堆数值"。以下每块均以真实证据收尾（全客户端 E2E 或确定性单测 + t1 `-race`）。

- [x] 副本/镜像/跨服实例地图生命周期做实（销毁统一走 MapManager：StopActor + 停 spawnLoop + 清理；修复实例地图 goroutine 泄漏，含 cleanup 护栏测试）
- [x] 物品 + 仓库（item.proto 协议 + Warehouse；全客户端四进程 E2E 数量一致性验证）
- [x] 技能（skill.proto + 接既有 SkillManager；E2E 学习/查询/升级/释放，冷却校验生效）
- [x] 场景同步增强：战斗**血量/死亡**经 AOI 广播给视野内所有玩家（双客户端 PvP E2E：攻防两端实时收到血量变化 + 死亡）
- [x] 场景同步增强：技能 **buff 增/删**经 AOI 广播（确定性集成单测 + t1 `-race`）
- [x] 掉落拾取闭环（就近拾取 → MapServer 物品权威移除 → 跨服 grant 入背包；E2E 种子掉落→拾取→背包）
- [x] 世界聊天（玩家发起 → 全服在线扇出 → 多客户端接收；双客户端 E2E）
- [x] 组队（跨玩家共享花名册 + 成员子集广播；建/加/离，队长顺位；双客户端 E2E）
- [x] 玩家间交易（两方有状态事务 + 原子金币交换带回滚；双客户端 E2E 成交/回滚双路径，直查 DB 校验）
- [x] 邮件系统（**离线持久异步投递** + 领取金币附件；A 发给离线 B → B 登录领取 → 金币到账并持久，直查 DB）
- [x] 持久化补齐：背包/技能/仓库落库（登录载入 + 登出存盘，原生 SQL 对齐真实表，重登直查 DB 校验一致）
- [x] **修复**：登录时从 DB 载入玩家属性到 actor（金币/等级/经验/钻石）——此前遗漏致 actor 恒 0 且**登出会把金币清零落库（数据丢失）**；直查 DB 校验成交后余额正确

#### 持久化现状（2026-07-25 系统排查）

> 结论：唯一的数据损坏类 bug（玩家金币/等级登出清零）已修复。其余业务态为「内存态、按最小闭环刻意延后持久化」，非 bug（不损坏 DB），但跨登录不保留。

| 状态 | 载入 | 落库 | 说明 |
|------|------|------|------|
| 玩家属性 level/exp/gold/diamond | ✅ | ✅ savePlayer(周期+登出) | 已正确（修复载入缺口） |
| 背包 / 技能 | ✅ | ✅ 登出存盘 | **已落库**：登录从 player\_items/player\_skills 载入、登出「先清后插」写回（原生 SQL 对齐真实表；重登直查 DB 校验一致） |
| 仓库 | ✅ | ✅ 登出存盘 | **已落库**：自动建 player\_warehouse 表 + 登录载入/登出写回（直查 DB 校验） |
| 邮件 | ✅ 持久 | ✅ | **离线持久投递**：发件即落 player\_mails（收件人离线也收），登录拉取 + 领取金币附件（直查 DB：离线→领取金币到账并持久） |
| buff | ❌ | ❌ | 本就瞬时，合理 |
| 队伍 / 交易会话 | — | — | 会话态，本就不持久（正确） |

### 待开发

- [ ] 剩余8个 DAO 同步化（auction/login\_log/player\_buff 等）
- [ ] GatewayServer 多 GameServer 负载均衡
- [ ] 背包/物品/装备完整实现（装备属性同步到MapServer）
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

### 验证（门禁 + 测试）

```bash
# 全模块 build+vet 门禁（多 module，无 go.work）
powershell -ExecutionPolicy Bypass -File scripts/ci-check.ps1

# 引擎 + 契约 + 领域 + 持久化 测试
go -C ../zEngine test ./zActor/ ./zNet/ -count=1
go -C zCommon test ./crossserver/ ./game/ -count=1
# 持久化需 MySQL，未设 DSN 则自动 skip
ZMMO_TEST_MYSQL_DSN="root:pwd@tcp(host:3306)/db?parseTime=true" go -C zCommon test ./consistency/
```

**最小可运行示例 / 端到端**：`GameClient`（Go 版测试客户端，无需 Unity）跑通
登录→创角→进图→移动→攻击→AOI 全流程；E2E 编排步骤见
[`docs/开发约定与工作流.md`](docs/开发约定与工作流.md)。持久化可用 `ZMMO_PERSIST_CONSISTENCY=1` 开启。

### 配置文件

| 服务器           | 配置文件               | 说明                                               |
| ------------- | ------------------ | ------------------------------------------------ |
| GlobalServer  | config.ini         | HTTP/Database/Redis/Etcd/Metrics/Pprof           |
| GatewayServer | config.ini         | TCP/Security/DDoS/Compression/Etcd/Metrics/Pprof |
| GameServer    | config.ini         | TCP/Database/Etcd/Metrics/Pprof                  |
| MapServer     | config\_single.ini | 单服地图模式                                           |
| MapServer     | config\_mirror.ini | 镜像地图模式                                           |
| MapServer     | config\_cross.ini  | 世界地图模式                                           |
| MapServer     | config\_test.ini   | 测试配置                                             |

## 文档

完整文档导航（面向使用者 vs 内部工作台账的分类）见 [`docs/README.md`](docs/README.md)。常用入口：

| 文档 | 用途 |
|------|------|
| [`docs/README.md`](docs/README.md) | **文档索引**：区分面向使用者的说明 与 内部演进台账 |
| [`AGENTS.md`](AGENTS.md) | 跨工具（AI/编辑器/人）仓库入口，指向下列权威文档 |
| [`docs/开发约定与工作流.md`](docs/开发约定与工作流.md) | 开发方法、验证纪律、环境约定（AI 无关、可移植的单一真相源）|
| [`docs/成熟化改造-执行计划.md`](docs/成熟化改造-执行计划.md) | 进度台账（勾选框 = 进度真相）|
| [`docs/项目评估报告.md`](docs/项目评估报告.md) | 能力 → 证据清单（可验证事实）|
| [`docs/协议契约.md`](docs/协议契约.md) | 跨服 Game↔Map 线格式与 ID 契约 |
| [`docs/框架成熟化改造方案.md`](docs/框架成熟化改造方案.md) | 战略、关键决策、领域逻辑分层 |
| [`docs/架构设计方案.md`](docs/架构设计方案.md) | 架构设计 |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | 贡献指南 |

## 贡献

欢迎贡献。开始前请读 [`CONTRIBUTING.md`](CONTRIBUTING.md) 与
[`docs/开发约定与工作流.md`](docs/开发约定与工作流.md)——核心原则：**每个"完成"必须挂一个通过的测试或跨进程真 E2E**，
不凭主观判断。提交前跑 `scripts/ci-check.ps1` 确保 8 模块全绿。

## 许可证

[MIT License](LICENSE) © 2026

***

*最后更新: 2026-07-25*
