# zMmoServer

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

一套基于 Go 的**分布式 MMORPG 服务端参考实现**，采用 **realm（分区/分服）架构**。它建立在游戏引擎
[zEngine](https://github.com/pzqf/zEngine) 与工具库 [zUtil](https://github.com/pzqf/zUtil) 之上，
既是可运行的四层服务端，也是**学习"MMO 服务端怎么分层、跨服怎么通信、玩家状态怎么保证不出错"的样板**。

## 这是什么

把 MMO 服务端里**难做对**的部分——跨服通信、AOI 视野、单写者 Actor、玩家状态的权威分层与事件回流、
Outbox/Inbox 一致性、服务发现、优雅生命周期——做成了**可运行、有测试/E2E 背书**的骨架。
业务层刻意保持最小（够验证架构即可，不堆数值玩法），方便你在上面长自己的游戏。

- **适合**：想学/借鉴分布式游戏服务端架构的人；想在现成骨架上做玩法的团队。
- **不适合**：开箱即用的成品游戏（玩法是最小闭环）；把它当引擎用（引擎在 [zEngine](https://github.com/pzqf/zEngine)）。
- **成熟度**：`0.0.x`，成熟前不发 `1.0`。业务层最小、以架构为重。

> **第一次来？** 先读 [`docs/架构.md`](docs/架构.md)——从零把这套架构讲清楚（心智模型 + 为什么这么设计）。

## 架构一图

```
                    ┌──────────────┐
                    │ GlobalServer │  账号 / 服务器列表 / 服务发现（无游戏态）
                    └──────┬───────┘
                           │ etcd（按 GroupID 分组）
        客户端 ──TCP──▶ ┌──▼──────────┐
                        │GatewayServer│  连接 / 鉴权 / 路由（无游戏态）
                        └──────┬──────┘ 1:1
                        ┌──────▼──────┐
                        │ GameServer  │  玩家逻辑 + 持久数据权威（落库）
                        └──────┬──────┘ 1:N
                 ┌─────────────┼─────────────┐
            ┌────▼────┐   ┌────▼────┐   ┌────▼────┐
            │MapServer│   │MapServer│   │MapServer│  空间 / 战斗 / AOI / tick
            └─────────┘   └─────────┘   └─────────┘  瞬时模拟权威（不落库）
```

**这一整套 = 一个"服（realm）"。** 每个服自成一个完整世界，扩容就是再开一套（+ 独立数据库），
彼此隔离——不是全区全服。跨服玩法走临时实例。

## 核心设计（一句话版，详见 [`docs/架构.md`](docs/架构.md)）

- **realm 架构**：每服一个独立世界，加服扩容、天然隔离；跨服 = 临时实例 + 事件回流。
- **玩家的两个化身 + 权威分层**：GameServer 持有**持久权威**（背包/货币/等级，唯一落库者）；
  MapServer 持有**瞬时权威**（血/位/buff）。战斗产生的持久变更由 MapServer **事件回流**给
  GameServer 落库（幂等），**MapServer 绝不写库**——从设计上杜绝双写冲突。
- **地图单写者 Actor**：每张地图一条 goroutine，网络命令与帧更新都排队串行执行，从根上消除数据竞争。
- **AOI 视野**：只把你视野内实体的变化推给你；客户端视野由 MapServer AOI 单一权威驱动。
- **realm 内伸缩**：分线（热点图开多层分摊）/ 无缝（相邻分区透明交接）/ 实例（副本/战场/跨服）。
- **一致性**：跨进程消息带 `requestID` 幂等去重；关键在途消息走 Outbox/Inbox 持久化，进程重启不丢。

## 技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.25+ |
| 网络 | zEngine/zNet（TCP + Protobuf） |
| 服务发现 | etcd |
| 存储 | MySQL、Redis |
| 序列化 | Protocol Buffers |
| 监控 | Prometheus |
| 部署 | Kubernetes |

## 目录结构

```
zMmoServer/
├── GlobalServer/    # 全局服：账号 / 服务器列表 / 服务发现
├── GatewayServer/   # 网关服：连接 / 鉴权 / 路由
├── GameServer/      # 游戏服：玩家逻辑 + 持久数据权威（player Actor / 背包 / 技能 / 任务…）
├── MapServer/       # 地图服：地图实例 / AOI / 战斗 / AI / 分线·无缝·实例
├── GameClient/      # Go 版测试客户端（无需 Unity，驱动端到端 E2E）
├── zCommon/         # 共享库：crossserver（跨服）/ aoi / consistency / discovery / db / net …
├── resources/       # 配置表(Excel) / 地图(JSON) / proto
├── kubernetes/      # K8s 部署
└── docs/            # 文档（见下方导航）
```

> 引擎（`zEngine`）与工具库（`zUtil`）是独立仓库；本仓通过 go.mod `replace` 指向本地同级目录。

## 快速开始

```bash
# 依赖：Go 1.25+、etcd、MySQL（各服 config.ini 里配连接）

# 编译（多 module，各自 go.mod，用 go -C 逐模块）
go -C GlobalServer  build -o ../bin/global_server.exe  .
go -C GatewayServer build -o ../bin/gateway_server.exe .
go -C GameServer    build -o ../bin/game_server.exe    .
go -C MapServer     build -o ../bin/map_server.exe     .

# 启动顺序：etcd / MySQL 就绪后 → Global → Gateway → Game → Map

# 全模块 build+vet 门禁（无 go.work）
powershell -ExecutionPolicy Bypass -File scripts/ci-check.ps1
```

**跑通端到端**：`GameClient`（Go 测试客户端，无需 Unity）一条命令跑登录 → 创角 → 进图 → 移动 →
攻击 → AOI 全流程：`go -C GameClient build -o ../bin/gclient.exe ./cmd`，再 `gclient.exe -mode full`。

## 文档导航

| 文档 | 讲什么 | 什么时候看 |
|------|--------|-----------|
| [docs/架构.md](docs/架构.md) | **从零讲清整套架构**（心智模型 + 为什么这么设计） | 想理解系统怎么搭的（先读） |
| [docs/游戏开发指南.md](docs/游戏开发指南.md) | 在框架上**开发游戏功能**：七条架构铁律 + 加功能配方 | 要加系统/玩法/地图/协议 |
| [docs/realm架构与跨服设计.md](docs/realm架构与跨服设计.md) | 分服/跨服的方向与取舍（realm-based，非全区全服） | 规划分服/跨服 |
| [docs/realm架构-详细设计.md](docs/realm架构-详细设计.md) | 分线/无缝/实例/回流/建服的可执行深化 + 实现状态 | 动手做 realm 伸缩/跨服/开服 |
| [docs/协议契约.md](docs/协议契约.md) | 跨服 Game↔Map 的线格式与 ID 契约 | 对接/扩展跨服消息 |
| [docs/通信模式设计方案.md](docs/通信模式设计方案.md) | 服务间通信模式与消息流 | 想理解请求如何跨服流转 |
| [docs/规范.md](docs/规范.md) | 目录结构 / 命名 / 配置表规范 | 往仓库加代码/配置 |
| [docs/游戏设计.md](docs/游戏设计.md) | 游戏背景与玩法设计（业务参考） | 关心玩法示例 |
| [docs/K8s部署方案.md](docs/K8s部署方案.md) | Kubernetes 集群部署 | 要部署到集群 |
| [docs/开发约定与工作流.md](docs/开发约定与工作流.md) | 开发方法与验证纪律（贡献前必读） | 准备贡献代码 |

跨工具（AI/编辑器/人）统一入口见 [`AGENTS.md`](AGENTS.md)。

## 贡献

欢迎贡献。开始前请读 [`CONTRIBUTING.md`](CONTRIBUTING.md) 与 [`docs/开发约定与工作流.md`](docs/开发约定与工作流.md)。
核心原则：**每个"完成"必须挂一个通过的测试或一次跨进程真 E2E**，不凭主观判断；提交前跑 `scripts/ci-check.ps1` 全模块绿。

## 许可证

[MIT License](LICENSE) © 2026
