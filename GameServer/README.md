# GameServer - 游戏服务器

## 概述

GameServer（游戏服务器）是MMO游戏服务器架构的核心业务服务器，负责处理游戏逻辑、玩家数据管理、场景管理和游戏状态同步。它是游戏世界的核心，处理所有游戏相关的业务逻辑。

## 核心功能

### 1. 玩家管理
- **玩家登录**：处理玩家登录请求，验证身份信息
- **角色管理**：创建、删除、查询玩家角色
- **角色选择**：支持玩家选择已有角色进入游戏
- **玩家数据**：管理玩家属性、装备、背包等数据

### 2. 场景管理
- **地图加载**：加载和管理游戏地图数据
- **场景切换**：处理玩家在不同场景间的切换
- **视野管理**：管理玩家的视野范围和可见对象
- **碰撞检测**：处理场景中的碰撞检测

### 3. 游戏逻辑
- **战斗系统**：处理战斗计算和伤害判定
- **技能系统**：管理技能释放和效果
- **任务系统**：处理任务接取、完成和奖励
- **NPC交互**：处理与NPC的对话和交易

### 4. 数据持久化
- **数据库操作**：使用GORM进行数据库操作
- **数据缓存**：Redis缓存热点数据
- **数据同步**：定期同步玩家数据到数据库
- **数据备份**：支持数据备份和恢复

### 5. 网络通信
- **Gateway连接**：与GatewayServer建立连接
- **消息处理**：处理来自客户端的游戏消息
- **状态同步**：同步游戏状态给客户端
- **广播机制**：支持全服广播和范围广播

## 系统架构

```
┌─────────────────────────────────────────────────┐
│                   GameServer                     │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │
│  │ 玩家管理    │  │ 场景管理    │  │ 游戏逻辑 │ │
│  │ ├─登录     │  │ ├─地图加载  │  │ ├─战斗   │ │
│  │ ├─角色     │  │ ├─视野管理  │  │ ├─技能   │ │
│  │ └─数据     │  │ └─碰撞检测  │  │ └─任务   │ │
│  └─────────────┘  └─────────────┘  └──────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │
│  │ 网络层      │  │ 数据层      │  │ 工具层   │ │
│  │ ├─Gateway  │  │ ├─MySQL    │  │ ├─配置   │ │
│  │ ├─协议     │  │ ├─Redis    │  │ ├─日志   │ │
│  │ └─消息路由 │  │ └─缓存     │  │ └─监控   │ │
│  └─────────────┘  └─────────────┘  └──────────┘ │
└─────────────────────────────────────────────────┘
                          │
                          ▼
                   ┌─────────────┐
                   │  Gateway    │
                   └─────────────┘
```

## 目录结构

```
GameServer/
├── config/                    # 配置管理
│   └── config.go             # 配置结构和加载
├── connection/               # 连接管理
│   └── connection_manager.go # Gateway连接管理
├── data/                     # 数据定义
│   ├── cache/               # 缓存定义
│   ├── database/            # 数据库模型
│   └── redis/               # Redis定义
├── entity/                   # 实体定义
│   └── player.go            # 玩家实体
├── game/                     # 游戏核心
│   ├── base_server.go       # 基础服务器
│   ├── player_manager.go    # 玩家管理器
│   └── scene_manager.go     # 场景管理器
├── handler/                  # 消息处理器
│   └── player_handler.go    # 玩家消息处理
├── net/                      # 网络层
│   ├── protolayer/         # 协议层
│   │   └── protobuf_protocol.go
│   ├── router/             # 消息路由
│   │   └── packet_router.go
│   └── service/            # 网络服务
│       └── tcp_service.go
├── service/                  # 业务服务
│   └── player_service.go   # 玩家服务
├── session/                  # 会话管理
│   └── session_manager.go  # 会话管理器
├── config.ini               # 配置文件
├── main.go                  # 入口程序
└── README.md                # 本文档
```

## 快速开始

### 1. 环境要求
- Go 1.20+
- MySQL 5.7+
- Redis 6.0+
- 配置文件 config.ini

### 2. 数据库初始化
```sql
-- 创建数据库
CREATE DATABASE GameDB_000101 CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建玩家表
CREATE TABLE players (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    account_id BIGINT NOT NULL,
    name VARCHAR(32) NOT NULL,
    level INT DEFAULT 1,
    exp BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_account (account_id),
    UNIQUE KEY uk_name (name)
);
```

### 3. 编译运行
```bash
# 编译
go build -o gameserver.exe main.go

# 运行
./gameserver.exe
```

### 4. 配置文件
详见 [config.ini](#配置文件说明) 部分的详细说明。

## 配置文件说明

### [Server] - 服务器基本配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| ServerName | 服务器名称 | GameServer |
| ServerID | 服务器ID | 1 |
| ListenAddr | 监听地址 | 0.0.0.0:9001 |
| MaxConnections | 最大连接数 | 10000 |
| ConnectionTimeout | 连接超时时间(秒) | 300 |
| HeartbeatInterval | 心跳间隔(秒) | 30 |

### [Database] - 数据库配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| DBType | 数据库类型 | mysql |
| DBHost | 数据库主机 | 127.0.0.1 |
| DBPort | 数据库端口 | 3306 |
| DBName | 数据库名称 | GameDB_000101 |
| DBUser | 数据库用户 | root |
| DBPassword | 数据库密码 | root |
| MaxOpenConns | 最大连接数 | 100 |
| MaxIdleConns | 最大空闲连接 | 10 |
| ConnMaxLifetime | 连接最大生命周期(秒) | 3600 |

### [Gateway] - Gateway配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| GatewayAddr | Gateway地址 | 127.0.0.1:8081 |
| GatewayConnectTimeout | 连接超时(秒) | 10 |

### [Logging] - 日志配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| LogLevel | 日志级别 | info |
| LogFile | 日志文件路径 | logs/server.log |
| LogMaxSize | 单个日志文件最大大小(MB) | 100 |
| LogMaxBackups | 日志文件保留数量 | 5 |
| LogMaxAge | 日志保留天数 | 30 |

### [Metrics] - 监控配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| Enabled | 是否启用监控 | true |
| MetricsAddr | 监控服务地址 | 0.0.0.0:9092 |

## 关键组件详解

### BaseServer（基础服务器）
- 服务器生命周期管理
- 组件初始化和销毁
- 信号处理和优雅关闭
- 依赖注入容器管理

### ConnectionManager（连接管理器）
- 管理与Gateway的连接
- 连接状态监控
- 自动重连机制
- 消息发送队列

### SessionManager（会话管理器）
- 玩家会话管理
- 会话状态跟踪
- 会话超时处理
- 多设备登录控制

### PlayerService（玩家服务）
- 玩家数据CRUD操作
- 玩家登录验证
- 角色列表管理
- 角色创建和删除

### PacketRouter（消息路由器）
- 消息分发和路由
- 处理器注册
- 消息过滤和拦截
- 错误处理

### SceneManager（场景管理器）
- 场景加载和卸载
- 玩家进出场景
- AOI（兴趣区域）管理
- 场景事件分发

## 消息处理流程

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  Client  │───▶│ Gateway  │───▶│ GameSrv  │───▶│ Handler  │
└──────────┘    └──────────┘    └──────────┘    └────┬─────┘
                                                     │
                                                     ▼
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  Client  │◀───│ Gateway  │◀───│ GameSrv  │◀───│ Service  │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
```

1. **接收消息**：从Gateway接收客户端消息
2. **协议解析**：解析Protobuf协议数据
3. **路由分发**：根据消息ID路由到对应处理器
4. **业务处理**：执行业务逻辑
5. **数据操作**：读写数据库或缓存
6. **响应返回**：构建响应消息返回给客户端

## 数据流

### 玩家登录流程
```
1. 接收登录请求（账号ID + Token）
2. 验证Token有效性
3. 查询玩家角色列表
4. 返回角色列表给客户端
5. 等待玩家选择角色
6. 加载角色数据
7. 进入游戏场景
8. 同步初始状态
```

### 角色创建流程
```
1. 接收创建角色请求
2. 验证角色名合法性
3. 检查角色名是否重复
4. 创建角色数据
5. 保存到数据库
6. 返回创建结果
```

## 性能优化

### 1. 数据库优化
- 连接池管理
- 索引优化
- 读写分离
- 分库分表

### 2. 缓存策略
- Redis缓存热点数据
- 本地缓存频繁访问数据
- 缓存更新策略
- 缓存穿透防护

### 3. 异步处理
- 异步数据库写入
- 消息队列处理
- 定时任务调度
- 协程池管理

### 4. 内存优化
- 对象池复用
- 内存预分配
- GC优化
- 内存泄漏检测

## 监控指标

GameServer提供以下监控指标：
- 在线玩家数
- 每秒处理消息数
- 数据库操作延迟
- 缓存命中率
- 场景负载情况
- 内存使用情况
- Goroutine数量

## 故障排查

### 常见问题

1. **数据库连接失败**
   - 检查数据库配置
   - 确认网络连通性
   - 检查数据库权限

2. **Gateway连接失败**
   - 检查Gateway地址配置
   - 确认Gateway服务状态
   - 查看防火墙设置

3. **内存泄漏**
   - 检查对象池使用
   - 分析内存分配
   - 检查goroutine泄漏

4. **性能瓶颈**
   - 分析CPU和内存使用
   - 检查数据库慢查询
   - 优化热点代码

## 开发指南

### 添加新的消息处理器
1. 在`handler`包中创建处理器
2. 实现消息处理方法
3. 在`packet_router.go`中注册路由
4. 更新协议定义（如需要）

### 添加新的实体
1. 在`entity`包中定义实体结构
2. 在`data/database`中定义数据库模型
3. 在`service`中实现业务逻辑
4. 添加对应的处理器

### 扩展场景功能
1. 在`game`包中扩展SceneManager
2. 实现场景加载逻辑
3. 添加场景事件处理
4. 更新AOI管理

## 版本历史

### v1.0.0
- 初始版本发布
- 支持玩家登录和角色管理
- 实现基础网络通信
- 集成数据库和缓存
- 支持Protobuf协议

## 贡献指南

欢迎提交Issue和Pull Request！

## 许可证

MIT License
