# Kubernetes部署配置

本目录包含zMmoServer的Kubernetes部署配置文件。

## 目录结构

- `gateway/` - GatewayServer的Kubernetes配置
- `game/` - GameServer的Kubernetes配置
- `map/` - MapServer的Kubernetes配置
- `global/` - GlobalServer的Kubernetes配置
- `etcd/` - etcd集群配置（用于服务发现和配置中心）
- `mysql/` - MySQL数据库配置
- `prometheus/` - Prometheus监控配置

## 部署步骤

1. **准备环境**
   - Kubernetes集群（版本1.20+）
   - Helm 3.0+
   - 网络插件（如Calico、Flannel等）

2. **部署基础服务**
   ```bash
   # 部署etcd集群
   kubectl apply -f etcd/
   
   # 部署MySQL
   kubectl apply -f mysql/
   ```

3. **部署游戏服务器**
   ```bash
   # 部署GlobalServer
   kubectl apply -f global/
   
   # 部署GatewayServer
   kubectl apply -f gateway/
   
   # 部署GameServer
   kubectl apply -f game/
   
   # 部署MapServer
   kubectl apply -f map/
   ```

4. **部署监控**
   ```bash
   # 部署Prometheus和Grafana
   kubectl apply -f prometheus/
   ```

## 配置管理

在Kubernetes环境中，配置管理采用以下策略：

1. **环境变量** - 用于基本配置，如服务地址、端口等
2. **ConfigMap** - 用于静态配置文件
3. **Secret** - 用于敏感信息，如数据库密码、令牌密钥等
4. **etcd配置中心** - 用于动态配置和服务发现

## 服务发现

使用etcd作为服务发现和配置中心：

- 每个服务器启动时向etcd注册
- 定期发送心跳保持注册状态
- 其他服务通过etcd发现目标服务

## 监控和日志

- **Prometheus** - 用于指标监控
- **Grafana** - 用于监控面板
- **ELK Stack** - 用于日志收集和分析

## 扩展和缩容

- **GatewayServer** - 使用Deployment和HPA自动扩缩容
- **GameServer** - 使用StatefulSet管理有状态服务
- **MapServer** - 使用Deployment管理无状态服务

## 高可用

- **etcd** - 3节点集群
- **MySQL** - 主从复制
- **GameServer** - 多实例部署
- **GatewayServer** - 多实例负载均衡
