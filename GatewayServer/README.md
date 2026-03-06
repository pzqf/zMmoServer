# GatewayServer - 网关服务器

## 概述

GatewayServer（网关服务器）是MMO游戏服务器架构的核心组件，负责处理客户端连接、安全验证、数据转发和负载均衡。它是客户端与游戏服务器之间的桥梁，提供统一的接入点。

## 核心功能

### 1. 客户端连接管理
- **TCP连接处理**：支持高并发客户端连接（默认最大10000连接）
- **连接状态监控**：实时监控连接状态和活动
- **心跳检测**：自动检测并清理超时连接
- **连接限制**：防止单一IP过度连接

### 2. 安全防护
- **DDoS防护**：
  - IP连接频率限制
  - 数据包频率限制
  - 流量限制
  - 自动封禁恶意IP
- **Token验证**：JWT令牌验证机制
- **登录保护**：限制登录尝试次数，防止暴力破解

### 3. 数据转发
- **协议解析**：解析客户端数据包
- **数据压缩**：支持Snappy压缩算法，减少网络传输
- **路由转发**：将请求转发到对应的GameServer
- **响应返回**：将GameServer的响应返回给客户端

### 4. 负载均衡
- **多GameServer支持**：可连接多个GameServer实例
- **智能路由**：根据玩家ID或服务器负载进行路由
- **故障转移**：自动检测并切换到可用服务器

## 系统架构

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ TCP
       ▼
┌──────────────────┐
│   GatewayServer  │
│  ├─ DDoS防护     │
│  ├─ 连接管理     │
│  ├─ Token验证    │
│  ├─ 数据压缩     │
│  └─ 消息转发     │
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
├── config/                 # 配置管理
│   └── config.go          # 配置结构和加载
├── connection/            # 连接管理
│   └── connection_manager.go
├── gateway/               # 网关核心
│   └── base_server.go     # 基础服务器实现
├── metrics/               # 监控指标
│   └── metrics.go         # 性能监控
├── net/                   # 网络工具
│   └── compression_manager.go  # 压缩管理
├── protocol/              # 协议处理
│   └── protocol_handler.go     # 协议编解码
├── proxy/                 # 代理服务
│   └── game_server_proxy.go    # GameServer代理
├── service/               # 业务服务
│   └── tcp_service.go     # TCP服务实现
├── token/                 # Token管理
│   └── token_manager.go   # JWT令牌管理
├── config.ini             # 配置文件
├── main.go                # 入口程序
└── README.md              # 本文档
```

## 快速开始

### 1. 环境要求
- Go 1.20+
- 配置文件 config.ini

### 2. 编译运行
```bash
# 编译
go build -o gatewayserver.exe main.go

# 运行
./gatewayserver.exe
```

### 3. 配置文件
详见 [config.ini](#配置文件说明) 部分的详细说明。

## 配置文件说明

### [Server] - 服务器基本配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| ServerName | 服务器名称 | GatewayServer |
| ServerID | 服务器ID | 1 |
| ListenAddr | 监听地址 | 0.0.0.0:8081 |
| MaxConnections | 最大连接数 | 10000 |
| ConnectionTimeout | 连接超时时间(秒) | 300 |
| HeartbeatInterval | 心跳间隔(秒) | 30 |

### [Security] - 安全配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| TokenSecret | JWT密钥 | your_secret_key_here |
| TokenExpiry | Token过期时间(秒) | 86400 |
| MaxLoginAttempts | 最大登录尝试次数 | 5 |
| BanDuration | 封禁时长(秒) | 3600 |

### [GameServer] - GameServer配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| GameServerID | GameServer ID | 1 |
| GameServerAddr | GameServer地址 | 127.0.0.1:9001 |
| GameServerConnectTimeout | 连接超时(秒) | 10 |

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
| MetricsAddr | 监控服务地址 | 0.0.0.0:9091 |

### [ddos] - DDoS防护配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| max_conn_per_ip | 单IP最大连接数 | 10 |
| conn_time_window | 连接统计时间窗口(秒) | 60 |
| max_packets_per_ip | 单IP最大数据包数 | 100 |
| packet_time_window | 数据包统计时间窗口(秒) | 1 |
| max_bytes_per_ip | 单IP最大流量(字节) | 10485760 (10MB) |
| traffic_time_window | 流量统计时间窗口(秒) | 3600 |
| ban_duration | IP封禁时长(秒) | 86400 |

### [net_compression] - 网络压缩配置
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| enabled | 是否启用压缩 | true |
| threshold | 压缩阈值(字节) | 1024 |
| level | 压缩级别 | 1 |
| min_quality | 最低网络质量 | 0 |
| max_quality | 最高网络质量 | 100 |

## 关键组件详解

### ConnectionManager（连接管理器）
- 管理所有客户端连接
- 维护连接状态和活动记录
- 提供连接查询和统计功能

### ProtocolHandler（协议处理器）
- 处理消息编解码
- 支持自定义协议格式
- 消息长度校验

### TokenManager（Token管理器）
- JWT令牌生成和验证
- 支持Token过期检测
- 安全密钥管理

### GameServerProxy（GameServer代理）
- 管理与GameServer的连接
- 消息转发和路由
- 自动重连机制

### CompressionManager（压缩管理器）
- Snappy压缩算法
- 智能压缩判断
- 网络质量自适应

## 性能优化

### 1. 连接优化
- 使用连接池管理连接
- 异步消息处理
- 批量数据发送

### 2. 内存优化
- 对象池复用
- 缓冲区管理
- 垃圾回收优化

### 3. 网络优化
- 数据压缩减少带宽
- 心跳包优化
- 批量消息合并

## 监控指标

GatewayServer提供以下监控指标：
- 当前连接数
- 每秒请求数(QPS)
- 平均响应时间
- 网络流量统计
- DDoS拦截次数
- 压缩比率

## 故障排查

### 常见问题

1. **连接数达到上限**
   - 检查MaxConnections配置
   - 查看是否有异常连接
   - 考虑增加服务器资源

2. **DDoS误拦截**
   - 调整ddos配置阈值
   - 检查流量模式
   - 添加白名单

3. **与GameServer连接失败**
   - 检查GameServer地址配置
   - 确认网络连通性
   - 查看防火墙设置

## 开发指南

### 添加新的消息处理
1. 在`protocol`包中定义消息类型
2. 在`service/tcp_service.go`中添加处理逻辑
3. 更新配置文件（如需要）

### 扩展DDoS防护
1. 修改`config.ini`中的ddos配置
2. 在`net`包中添加自定义防护逻辑
3. 重新编译部署

## 版本历史

### v1.0.0
- 初始版本发布
- 支持基础连接管理
- 实现DDoS防护
- 支持数据压缩
- 集成JWT验证

## 贡献指南

欢迎提交Issue和Pull Request！

## 许可证

MIT License
