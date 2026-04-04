# zMmoServer

## 项目简介
zMmoServer 是一个基于 Go 语言开发的分布式 MMORPG 游戏服务器框架，采用微服务架构设计，支持多服务器协同工作，为大型多人在线游戏提供稳定、高效的服务支持。

### 最新特性
- **统一通信模式**：实现了 BaseMessage 和 CrossServerMessage 统一消息结构
- **智能路由**：基于玩家-地图映射的消息路由机制
- **数据一致性**：实现了 Inbox/Outbox 机制确保消息可靠投递
- **并发安全**：使用 zMap.TypedMap 确保高并发场景下的数据安全
- **服务发现优化**：MapServer 注册时包含负责的地图ID列表

## 系统架构

### 服务器组件
- **GlobalServer**：全局服务器，负责跨服管理、账号认证、全局配置
- **GatewayServer**：网关服务器，负责客户端连接管理、消息转发、防作弊
- **GameServer**：游戏服务器，负责玩家数据管理、游戏逻辑处理、地图服务器映射
- **MapServer**：地图服务器，负责地图管理、AI、技能、碰撞检测等
- **AdminServer**：管理服务器（暂未实现），负责后台管理、监控等

### 核心功能
- **分布式架构**：基于微服务设计，支持服务器水平扩展
- **服务发现**：基于 etcd 实现服务注册与发现
- **配置管理**：统一的配置文件管理，支持 Excel 配置表
- **网络通信**：基于 TCP + Protocol Buffers 的高效通信
- **数据持久化**：支持 MySQL 和 MongoDB 存储
- **防作弊机制**：客户端行为分析、异常操作检测
- **监控系统**：服务器状态跟踪、告警规则、心跳上报
- **AI 系统**：怪物 AI、路径寻路、行为决策
- **技能系统**：技能释放、伤害计算、特效处理
- **社交系统**：组队、公会、交易等
- **经济系统**：货币、商店、拍卖等
- **成就系统**：成就任务、奖励发放
- **宠物系统**：宠物养成、技能、战斗
- **坐骑系统**：坐骑养成、属性加成
- **活动系统**：限时活动、奖励发放
- **副本系统**：副本管理、Boss 战
- **通信模式**：统一消息结构、智能路由、数据一致性
## 技术栈

- **语言**：Go 1.25+
- **Web 框架**：gorilla/websocket
- **日志**：go.uber.org/zap + lumberjack
- **配置管理**：gopkg.in/ini.v1, Kubernetes ConfigMap/Secret
- **数据�?*：MySQL (go-sql-driver/mysql), MongoDB (mongo-driver)
- **服务发现**：etcd, Kubernetes Service
- **监控**：Prometheus, Grafana
- **容器编排**：Kubernetes
- **序列�?*：Protocol Buffers (google.golang.org/protobuf)
- **本地模块**：zEngine, zUtil

## 项目结构

```
zMmoServer/
├── AdminServer/          # 管理服（暂未实现�?├── GameServer/           # 游戏�?├── GatewayServer/        # 网关�?├── GlobalServer/         # 全局�?├── MapServer/            # 地图�?├── docs/                 # 文档
├── kubernetes/           # Kubernetes 部署配置
├── resources/            # 资源文件
�?  ├── excel_tables/     # 配置�?�?  ├── maps/             # 地图文件
�?  └── protocol/         # 协议文件
├── testclient/           # 测试客户�?├── zCommon/           # 共享�?└── README.md             # 项目文档
```

## 开发进度
### 已完成功能
- [x] 服务器框架搭建
- [x] 网络通信模块
- [x] 服务发现与注册
- [x] 配置文件管理
- [x] 防作弊机制
- [x] 监控系统
- [x] 线程模型优化（支持工作池模式）
- [x] 分布式锁实现
- [x] 基础游戏系统（技能、AI、Buff等）
- [x] 统一通信模式（BaseMessage/CrossServerMessage）
- [x] 智能消息路由（基于玩家-地图映射）
- [x] 数据一致性机制（Inbox/Outbox）
- [x] 服务发现优化（MapServer地图ID注册）
- [x] 并发安全数据结构（zMap.TypedMap）

### 待开发功�?- [ ] AdminServer 实现
- [ ] 跨服功能完善
- [ ] 数据库分片与读写分离
- [ ] Kubernetes 部署优化
- [ ] 性能测试与优�?
## 快速开�?
### 编译服务�?
```bash
# 编译 GlobalServer
cd GlobalServer
go build -o bin/global_server.exe ./main.go

# 编译 GatewayServer
cd ../GatewayServer
go build -o bin/gateway_server.exe ./main.go

# 编译 GameServer
cd ../GameServer
go build -o bin/game_server.exe ./main.go

# 编译 MapServer
cd ../MapServer
go build -o bin/map_server.exe ./main.go
```

### 运行服务�?
1. 启动 etcd 服务（用于服务发现）
2. 配置各服务器�?`config.ini` 文件
3. 按顺序启动服务器：GlobalServer �?GatewayServer �?GameServer �?MapServer

### 配置文件

各服务器的配置文件位于各自目录下�?`config.ini`，主要配置项包括�?- 服务器基本信息（类型、ID、名称等�?- 网络配置（监听地址、端口等�?- 数据库配置（连接信息、池大小等）
- 日志配置（级别、文件路径等�?- 其他服务地址（如 etcd 地址、其他服务器地址等）

## 开发规�?
请参�?`project_rules.md` 文件，了解项目的代码规范、架构设计原则和开发约束�?
## 贡献指南

1. Fork 本项�?2. 创建特性分�?(`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开�?Pull Request

## 许可�?
本项目采�?MIT 许可证，详情请参�?LICENSE 文件�?
## 联系方式

如有问题或建议，欢迎通过 GitHub Issues 或邮件联系我们�?
---

**最后更新时间**：2026-04-03

