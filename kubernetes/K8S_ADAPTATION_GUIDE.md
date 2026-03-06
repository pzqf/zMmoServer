# Kubernetes部署适配指南

本指南详细说明如何将zMmoServer项目修改为适应Kubernetes部署环境。

## 1. 核心修改

### 1.1 配置管理

- **环境变量支持**：修改各服务器的配置读取代码，使其能够从环境变量中读取配置
- **ConfigMap**：使用Kubernetes ConfigMap存储静态配置文件
- **Secret**：使用Kubernetes Secret存储敏感信息
- **etcd配置中心**：使用etcd作为动态配置中心

### 1.2 服务发现

- **etcd服务发现**：使用etcd实现服务注册和发现
- **Kubernetes Service**：使用Headless Service实现服务间的直接通信
- **StatefulSet**：使用StatefulSet管理有状态服务（如GameServer）

### 1.3 容器化

- **Dockerfile**：为每个服务器创建Dockerfile
- **镜像管理**：使用Docker镜像仓库管理容器镜像
- **资源限制**：为容器设置合理的资源限制

### 1.4 监控和日志

- **Prometheus**：集成Prometheus监控
- **Grafana**：使用Grafana创建监控面板
- **ELK Stack**：集成ELK Stack进行日志收集和分析

## 2. 目录结构

```
zMmoServer/
├── kubernetes/              # Kubernetes部署配置
│   ├── gateway/             # GatewayServer配置
│   ├── game/                # GameServer配置
│   ├── map/                 # MapServer配置
│   ├── global/              # GlobalServer配置
│   ├── etcd/                # etcd集群配置
│   ├── mysql/               # MySQL配置
│   ├── prometheus/          # Prometheus配置
│   └── K8S_ADAPTATION_GUIDE.md  # 本指南
├── zMmoShared/
│   ├── discovery/           # 服务发现实现
│   └── configcenter/        # 配置中心实现
├── GatewayServer/
├── GameServer/
├── MapServer/
└── GlobalServer/
```

## 3. 配置修改

### 3.1 GatewayServer

- **环境变量**：支持从环境变量读取配置，如`SERVER_ID`、`TOKEN_SECRET`等
- **服务发现**：向etcd注册服务，发现GameServer
- **负载均衡**：使用Kubernetes Service实现负载均衡

### 3.2 GameServer

- **环境变量**：支持从环境变量读取配置，如`SERVER_ID`、`DB_PASSWORD`等
- **服务发现**：向etcd注册服务，发现GatewayServer和GlobalServer
- **有状态管理**：使用StatefulSet管理有状态服务

### 3.3 MapServer

- **环境变量**：支持从环境变量读取配置
- **服务发现**：向etcd注册服务，发现GameServer
- **自动扩缩容**：使用HorizontalPodAutoscaler实现自动扩缩容

### 3.4 GlobalServer

- **环境变量**：支持从环境变量读取配置
- **服务发现**：向etcd注册服务，被其他服务发现
- **健康检查**：实现HTTP健康检查端点

## 4. 部署步骤

### 4.1 准备环境

1. **Kubernetes集群**：部署Kubernetes集群（版本1.20+）
2. **Helm**：安装Helm 3.0+
3. **网络插件**：安装网络插件（如Calico、Flannel等）

### 4.2 部署基础服务

```bash
# 创建命名空间
kubectl create namespace game

# 部署etcd集群
kubectl apply -f kubernetes/etcd/

# 部署MySQL
kubectl apply -f kubernetes/mysql/
```

### 4.3 部署游戏服务器

```bash
# 部署GlobalServer
kubectl apply -f kubernetes/global/

# 部署GatewayServer
kubectl apply -f kubernetes/gateway/

# 部署GameServer
kubectl apply -f kubernetes/game/

# 部署MapServer
kubectl apply -f kubernetes/map/
```

### 4.4 部署监控

```bash
# 部署Prometheus和Grafana
kubectl apply -f kubernetes/prometheus/
```

## 5. 服务发现和配置中心

### 5.1 服务注册

每个服务器启动时，会向etcd注册自己的服务信息，包括：
- 服务名称
- 服务ID
- 服务地址
- 服务端口
- 元数据

### 5.2 服务发现

其他服务通过etcd发现目标服务，实现动态服务发现。

### 5.3 配置管理

- **静态配置**：使用ConfigMap存储，如配置文件
- **动态配置**：使用etcd存储，支持实时更新
- **敏感信息**：使用Secret存储，如数据库密码、令牌密钥

## 6. 监控和日志

### 6.1 指标监控

- **Prometheus**：收集服务器指标
- **Grafana**：可视化监控面板
- **告警**：配置告警规则

### 6.2 日志管理

- **ELK Stack**：收集、存储和分析日志
- **日志级别**：通过配置文件和环境变量控制
- **日志轮转**：配置日志轮转策略

## 7. 扩展和高可用

### 7.1 水平扩展

- **GatewayServer**：使用Deployment和HPA自动扩缩容
- **MapServer**：使用Deployment和HPA自动扩缩容
- **GameServer**：使用StatefulSet管理，根据负载手动扩缩容

### 7.2 高可用

- **etcd**：3节点集群
- **MySQL**：主从复制
- **GameServer**：多实例部署
- **GatewayServer**：多实例负载均衡

## 8. 最佳实践

### 8.1 配置管理

- 使用环境变量覆盖配置文件中的默认值
- 使用Secret存储敏感信息
- 使用ConfigMap存储静态配置
- 使用etcd存储动态配置

### 8.2 服务发现

- 使用etcd作为服务发现中心
- 使用Headless Service实现服务间的直接通信
- 使用StatefulSet管理有状态服务

### 8.3 监控和日志

- 集成Prometheus和Grafana
- 配置合理的告警规则
- 实现健康检查端点
- 配置日志轮转和保留策略

### 8.4 性能优化

- 设置合理的资源限制
- 优化数据库连接池
- 合理配置服务发现和配置中心
- 使用水平扩展应对高负载

## 9. 故障排查

### 9.1 常见问题

- **服务注册失败**：检查etcd连接和权限
- **服务发现失败**：检查服务注册状态和网络连接
- **配置加载失败**：检查环境变量和ConfigMap
- **数据库连接失败**：检查数据库服务状态和连接字符串

### 9.2 排查工具

- **kubectl**：查看Pod状态和日志
- **etcdctl**：查看etcd中的服务注册信息
- **Prometheus**：查看监控指标
- **Grafana**：查看监控面板

## 10. 总结

通过以上修改，zMmoServer项目可以完全适应Kubernetes部署环境，实现：

- **动态服务发现**：使用etcd和Kubernetes Service
- **集中配置管理**：使用ConfigMap、Secret和etcd
- **自动扩缩容**：使用HorizontalPodAutoscaler
- **高可用**：多实例部署和负载均衡
- **监控和日志**：集成Prometheus和ELK Stack

这些修改不仅提高了系统的可靠性和可扩展性，也简化了运维管理，为大规模部署做好了准备。