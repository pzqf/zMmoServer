# zMmoServer

## 项目简介
zMmoServer 是一个基于 Go 语言开发的分布式 MMORPG 游戏服务器框架，采用微服务架构设计，支持多服务器协同工作，为大型多人在线游戏提供稳定、高效的服务支持。

## 系统架构

### 服务器组件
- **GlobalServer**：全局服务器，负责跨服管理、账号认证、全局配置等
- **GatewayServer**：网关服务器，负责客户端连接管理、消息转发
- **GameServer**：游戏服务器，负责玩家数据管理、游戏逻辑处理
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

## 技术栈

- **语言**：Go 1.25+
- **Web 框架**：gorilla/websocket
- **日志**：go.uber.org/zap + lumberjack
- **配置管理**：gopkg.in/ini.v1, Kubernetes ConfigMap/Secret
- **数据库**：MySQL (go-sql-driver/mysql), MongoDB (mongo-driver)
- **服务发现**：etcd, Kubernetes Service
- **监控**：Prometheus, Grafana
- **容器编排**：Kubernetes
- **序列化**：Protocol Buffers (google.golang.org/protobuf)
- **本地模块**：zEngine, zUtil

## 项目结构

```
zMmoServer/
├── AdminServer/          # 管理服（暂未实现）
├── GameServer/           # 游戏服
├── GatewayServer/        # 网关服
├── GlobalServer/         # 全局服
├── MapServer/            # 地图服
├── docs/                 # 文档
├── kubernetes/           # Kubernetes 部署配置
├── resources/            # 资源文件
│   ├── excel_tables/     # 配置表
│   ├── maps/             # 地图文件
│   └── protocol/         # 协议文件
├── testclient/           # 测试客户端
├── zMmoShared/           # 共享包
└── README.md             # 项目文档
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

### 待开发功能
- [ ] AdminServer 实现
- [ ] 跨服功能完善
- [ ] 数据库分片与读写分离
- [ ] Kubernetes 部署优化
- [ ] 性能测试与优化

## 快速开始

### 编译服务器

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

### 运行服务器

1. 启动 etcd 服务（用于服务发现）
2. 配置各服务器的 `config.ini` 文件
3. 按顺序启动服务器：GlobalServer → GatewayServer → GameServer → MapServer

### 配置文件

各服务器的配置文件位于各自目录下的 `config.ini`，主要配置项包括：
- 服务器基本信息（类型、ID、名称等）
- 网络配置（监听地址、端口等）
- 数据库配置（连接信息、池大小等）
- 日志配置（级别、文件路径等）
- 其他服务地址（如 etcd 地址、其他服务器地址等）

## 开发规范

请参考 `project_rules.md` 文件，了解项目的代码规范、架构设计原则和开发约束。

## 贡献指南

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

## 许可证

本项目采用 MIT 许可证，详情请参阅 LICENSE 文件。

## 联系方式

如有问题或建议，欢迎通过 GitHub Issues 或邮件联系我们。

---

**最后更新时间**：2026-03-06
