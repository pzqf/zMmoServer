# Kubernetes部署配置

本目录包含zMmoServer的Kubernetes部署配置文件。

> ⚠️ **重要：这些清单是「示意 / 参考」性质，不能直接 `kubectl apply` 上线。** 它们与当前运行时模型存在已知不一致，直接部署会**静默失败**：
>
> - **ServerID 注入不匹配**：`game/game-statefulset.yaml` 用 `SERVER_ID="$(POD_NAME)"`（如 `game-server-0`），但代码把 ServerID 当 **6 位数字**（GroupID+ServerIndex，`id.ParseServerIDInt`）解析——pod 名非数字，起服解析即失败。`gateway` 同理。
> - **Gateway 多副本 vs 1:1 配对**：`gateway/gateway-deployment.yaml` 是 `replicas: 2` + HPA(2–10)，但 **Gateway↔GameServer 目前严格 1:1**（按单一 ServerID 配对）。多副本会都指向同一个 GameServer、共用同一 ServerID → 冲突。多 GameServer 负载均衡尚未实现（见根 `README.md` 待办）。
> - **Helm**：下方「部署步骤」提到 Helm 3.0+，但仓库内**没有 Helm chart**，实际只有原生 YAML（`kubectl apply -f`）。
>
> 当前真机跑法仍是**本地手动起 4 个进程**（见根 `README.md` 快速开始）。要真正上 K8s，需先补齐：编排层注入**数字化 ServerID**、多 GameServer 负载均衡、以及（如需要）Helm 化——这些属部署工程，尚未做。下面的清单可作为编写正式部署时的骨架参考。

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
