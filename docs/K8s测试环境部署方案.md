# Kubernetes 测试环境部署方案

## 1. 环境准备

### 1.1 更新系统包
```bash
sudo apt update
sudo apt upgrade -y
```

### 1.2 安装必要工具
```bash
sudo apt install -y curl apt-transport-https ca-certificates gnupg lsb-release
```

### 1.3 关闭防火墙和 swap
```bash
sudo systemctl stop firewalld 2>/dev/null || true
sudo systemctl disable firewalld 2>/dev/null || true
sudo swapoff -a
sudo sed -i '/swap/s/^/#/' /etc/fstab
```

### 1.4 配置内核参数
```bash
sudo cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward = 1
EOF
sudo sysctl --system
```

### 1.5 设置主机名
```bash
sudo hostnamectl set-hostname k8s-test
echo "127.0.0.1 k8s-test" >> /etc/hosts
```

## 2. 安装 Containerd

### 2.1 添加 Docker GPG 密钥
```bash
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
```

### 2.2 添加 Docker 仓库
```bash
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
```

### 2.3 安装 Containerd
```bash
sudo apt update
sudo apt install -y containerd.io
```

### 2.4 配置 Containerd
```bash
sudo mkdir -p /etc/containerd
sudo containerd config default | sudo tee /etc/containerd/config.toml
```

编辑 `/etc/containerd/config.toml` 文件，修改以下内容：

```toml
# 找到 [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
# 添加以下内容
[plugins."io.containerd.grpc.v1.cri".registry.mirrors]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."k8s.gcr.io"]
    endpoint = ["https://registry.aliyuncs.com/google_containers"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."registry.k8s.io"]
    endpoint = ["https://registry.aliyuncs.com/google_containers"]

# 找到 sandbox_image
# 修改为
sandbox_image = "registry.aliyuncs.com/google_containers/pause:3.10.1"
```

### 2.5 重启 Containerd
```bash
sudo systemctl restart containerd
sudo systemctl enable containerd
```

## 3. 安装 Kubernetes 组件

### 3.1 添加 Kubernetes GPG 密钥
```bash
curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/kubernetes-archive-keyring.gpg
```

### 3.2 添加 Kubernetes 仓库
```bash
echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list > /dev/null
```

### 3.3 安装 Kubernetes 组件
```bash
sudo apt update
sudo apt install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
```

## 4. 预拉取镜像（重要）

### 4.1 拉取所需镜像
```bash
# 控制平面组件
sudo ctr -n k8s.io image pull registry.aliyuncs.com/google_containers/kube-apiserver:v1.29.15
sudo ctr -n k8s.io image pull registry.aliyuncs.com/google_containers/kube-controller-manager:v1.29.15
sudo ctr -n k8s.io image pull registry.aliyuncs.com/google_containers/kube-scheduler:v1.29.15
sudo ctr -n k8s.io image pull registry.aliyuncs.com/google_containers/etcd:3.5.16-0
sudo ctr -n k8s.io image pull registry.aliyuncs.com/google_containers/coredns:v1.11.1
sudo ctr -n k8s.io image pull registry.aliyuncs.com/google_containers/kube-proxy:v1.29.15
sudo ctr -n k8s.io image pull registry.aliyuncs.com/google_containers/pause:3.10.1

# 网络插件镜像
sudo ctr -n k8s.io image pull ghcr.io/flannel-io/flannel:v0.28.1
sudo ctr -n k8s.io image pull ghcr.io/flannel-io/flannel-cni-plugin:v1.9.0-flannel1

# 或者使用 Calico 镜像
sudo ctr -n k8s.io image pull docker.m.daocloud.io/calico/node:v3.28.2
sudo ctr -n k8s.io image pull docker.m.daocloud.io/calico/cni:v3.28.2
sudo ctr -n k8s.io image pull docker.m.daocloud.io/calico/kube-controllers:v3.28.2
```

### 4.2 验证镜像
```bash
sudo ctr -n k8s.io image ls
```

## 5. 初始化 Kubernetes 集群

### 5.1 初始化集群
```bash
sudo kubeadm init \
  --apiserver-advertise-address=192.168.91.128 \
  --pod-network-cidr=10.244.0.0/16 \
  --image-repository=registry.aliyuncs.com/google_containers
```

初始化成功后会显示类似输出：
```
Your Kubernetes control-plane has initialized successfully!

To start using your cluster, you need to run the following as a regular user:

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config

...
```

### 5.2 配置 kubectl
```bash
mkdir -p ~/.kube
sudo cp /etc/kubernetes/admin.conf ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config
```

### 5.3 验证集群初始化
```bash
# 等待控制平面组件启动（约1-2分钟）
sleep 120

# 查看节点状态
kubectl get nodes -o wide

# 查看控制平面组件状态
kubectl get pods -n kube-system -o wide
```

## 6. 部署网络插件

### 6.1 方案一：部署 Flannel 网络插件（推荐，更简单）
```bash
kubectl apply -f https://github.com/coreos/flannel/raw/master/Documentation/kube-flannel.yml
```

### 6.2 方案二：部署 Calico 网络插件
如果使用 Calico，需要修改镜像仓库地址：

```bash
# 下载 Calico 配置文件
wget https://raw.githubusercontent.com/projectcalico/calico/v3.28.2/manifests/calico.yaml -O calico.yaml

# 修改镜像仓库（将 docker.io/calico/ 改为 docker.m.daocloud.io/calico/）
sed -i 's/docker.io\/calico\//docker.m.daocloud.io\/calico\//g' calico.yaml

# 部署 Calico
kubectl apply -f calico.yaml
```

### 6.3 验证网络插件部署
```bash
# 等待网络插件启动（约1-2分钟）
sleep 120

# 查看节点状态（应该变为 Ready）
kubectl get nodes -o wide

# 查看所有 Pod 状态
kubectl get pods -A -o wide
```

## 7. 允许控制平面节点运行 Pod（可选，单节点集群）

如果是单节点测试环境，需要移除控制平面节点的污点：

```bash
kubectl taint nodes --all node-role.kubernetes.io/control-plane-
```

## 8. 测试集群

### 8.1 部署测试 Pod
```bash
cat > nginx-test.yaml << EOF
apiVersion: v1
kind: Pod
metadata:
  name: nginx-test
  labels:
    app: nginx
spec:
  containers:
  - name: nginx
    image: nginx:alpine
    ports:
    - containerPort: 80
  tolerations:
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
EOF

kubectl apply -f nginx-test.yaml
```

### 8.2 检查测试 Pod
```bash
# 等待 Pod 启动
sleep 60

# 查看 Pod 状态
kubectl get pods -o wide

# 查看 Pod 详细信息
kubectl describe pod nginx-test
```

### 8.3 清理测试 Pod
```bash
kubectl delete -f nginx-test.yaml
```

## 9. 集群状态检查

### 9.1 查看节点状态
```bash
kubectl get nodes -o wide
```

### 9.2 查看所有 Pod 状态
```bash
kubectl get pods -A -o wide
```

### 9.3 查看集群信息
```bash
kubectl cluster-info
```

### 9.4 查看系统资源使用
```bash
# 查看节点资源使用
kubectl top nodes

# 查看 Pod 资源使用
kubectl top pods -A
```

注意：如果 `kubectl top` 命令不可用，需要先部署 metrics-server。

## 10. 集群清理和重置

### 10.1 完全重置集群（保留镜像）
```bash
# 1. 停止 kubelet
sudo systemctl stop kubelet

# 2. 停止 containerd
sudo systemctl stop containerd

# 3. 重置 kubeadm
sudo kubeadm reset -f

# 4. 清理所有配置和数据目录（重要！）
sudo rm -rf /etc/kubernetes/*
sudo rm -rf /var/lib/etcd/*
sudo rm -rf /var/lib/kubelet/*
sudo rm -rf /etc/cni/net.d/*
sudo rm -rf /var/lib/cni/*
sudo rm -rf /opt/cni/bin/*

# 5. 清理网络接口
sudo ip link delete cni0 2>/dev/null || true
sudo ip link delete flannel.1 2>/dev/null || true
sudo ip link delete tunl0 2>/dev/null || true

# 6. 清理 iptables 规则
sudo iptables -F
sudo iptables -t nat -F
sudo iptables -t mangle -F
sudo iptables -X
sudo iptables -t nat -X
sudo iptables -t mangle -X

# 7. 重启 containerd
sudo systemctl restart containerd
sleep 3

# 8. 重启 kubelet
sudo systemctl restart kubelet
sleep 3

# 9. 验证镜像仍在
export CONTAINER_RUNTIME_ENDPOINT=unix:///var/run/containerd/containerd.sock
sudo crictl images
```

### 10.2 完全清理（包括镜像）
```bash
# 执行上面的步骤 1-5，然后：

# 删除所有镜像
export CONTAINER_RUNTIME_ENDPOINT=unix:///var/run/containerd/containerd.sock
sudo crictl rmi --prune
```

## 11. 镜像版本清单

| 组件 | 镜像版本 | 镜像来源 | 用途 |
|------|---------|----------|------|
| kube-apiserver | v1.29.15 | registry.aliyuncs.com/google_containers | API 服务器 |
| kube-controller-manager | v1.29.15 | registry.aliyuncs.com/google_containers | 控制器管理器 |
| kube-scheduler | v1.29.15 | registry.aliyuncs.com/google_containers | 调度器 |
| etcd | 3.5.16-0 | registry.aliyuncs.com/google_containers | 分布式存储 |
| coredns | v1.11.1 | registry.aliyuncs.com/google_containers | DNS 服务 |
| kube-proxy | v1.29.15 | registry.aliyuncs.com/google_containers | 网络代理 |
| pause | 3.10.1 | registry.aliyuncs.com/google_containers | Pod 基础设施 |
| flannel | v0.28.1 | ghcr.io/flannel-io | 网络插件 |
| flannel-cni-plugin | v1.9.0-flannel1 | ghcr.io/flannel-io | Flannel CNI 插件 |
| calico/node | v3.28.2 | docker.m.daocloud.io/calico | Calico 节点 |
| calico/cni | v3.28.2 | docker.m.daocloud.io/calico | Calico CNI 插件 |
| calico/kube-controllers | v3.28.2 | docker.m.daocloud.io/calico | Calico 控制器 |

## 12. 关键注意事项

### 12.1 预拉取镜像（非常重要）
- **必须在初始化集群前预拉取所有所需镜像**
- 这样可以避免网络问题导致初始化失败
- 使用 `ctr -n k8s.io image pull` 拉取镜像到 containerd 的 k8s.io 命名空间

### 12.2 镜像仓库配置
- 使用阿里云镜像仓库作为 Google Container Registry 的镜像
- Flannel 镜像需要从 ghcr.io 拉取，或提前准备好
- Calico 镜像建议使用 docker.m.daocloud.io 镜像仓库

### 12.3 集群初始化参数
- `--apiserver-advertise-address`: 设置为节点的实际 IP 地址
- `--pod-network-cidr`: 必须与网络插件的 CIDR 匹配（Flannel 使用 10.244.0.0/16）
- `--image-repository`: 指定镜像仓库，避免从海外拉取

### 12.4 控制平面组件稳定性
- 初始化后需要等待 1-2 分钟让控制平面组件完全启动
- 组件可能会有多次重启，这是正常现象
- 如果持续崩溃重启，检查：
  - 系统资源（内存、CPU）
  - etcd 数据目录权限
  - 网络连通性
  - containerd 状态

### 12.5 网络插件部署
- 必须部署网络插件后节点才会变为 Ready 状态
- Flannel 更简单，适合测试环境
- Calico 功能更强大，但配置更复杂
- 网络插件部署后需要等待 1-2 分钟

### 12.6 单节点集群
- 测试环境通常使用单节点集群
- 需要移除控制平面节点的污点才能运行普通 Pod
- 使用 `kubectl taint` 命令移除污点

### 12.7 系统资源要求
- 最低要求：2 CPU 核心，4GB 内存
- 推荐配置：4 CPU 核心，8GB 内存
- 确保有足够的磁盘空间（至少 20GB）

## 13. 故障排查

### 13.1 控制平面组件持续重启
```bash
# 查看 kubelet 日志
sudo journalctl -u kubelet -n 100 --no-pager

# 查看组件日志
export CONTAINER_RUNTIME_ENDPOINT=unix:///var/run/containerd/containerd.sock
sudo crictl logs <container-id>

# 查看 etcd 日志
sudo crictl logs $(sudo crictl ps | grep etcd | awk '{print $1}')

# 查看 apiserver 日志
sudo crictl logs $(sudo crictl ps | grep kube-apiserver | awk '{print $1}')
```

### 13.2 节点状态为 NotReady
```bash
# 检查网络插件是否部署
kubectl get pods -n kube-flannel  # 或 kube-system 如果用 Calico

# 查看 kubelet 日志
sudo journalctl -u kubelet -n 50 --no-pager

# 检查 CNI 配置
ls -la /etc/cni/net.d/
```

### 13.3 镜像拉取失败
```bash
# 检查 containerd 配置
sudo cat /etc/containerd/config.toml | grep -A 10 "registry.mirrors"

# 手动拉取镜像测试
export CONTAINER_RUNTIME_ENDPOINT=unix:///var/run/containerd/containerd.sock
sudo ctr -n k8s.io image pull registry.aliyuncs.com/google_containers/pause:3.10.1

# 检查镜像是否存在
sudo crictl images
```

### 13.4 kubectl 无法连接
```bash
# 检查 kubeconfig 文件
ls -la ~/.kube/config

# 重新复制 kubeconfig
sudo cp /etc/kubernetes/admin.conf ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config

# 检查 apiserver 是否运行
export CONTAINER_RUNTIME_ENDPOINT=unix:///var/run/containerd/containerd.sock
sudo crictl ps | grep kube-apiserver
```

## 14. 部署检查清单

部署完成后，请确认以下项目：

- [ ] 所有控制平面组件正常运行（etcd、apiserver、controller-manager、scheduler）
- [ ] 节点状态为 Ready
- [ ] 网络插件 Pod 正常运行
- [ ] CoreDNS Pod 正常运行
- [ ] 可以成功部署测试 Pod
- [ ] 测试 Pod 可以正常访问
- [ ] `kubectl` 命令可以正常执行
- [ ] 集群资源使用在合理范围内

## 15. 部署记录示例

### 2026-03-13 成功部署记录（方案2：深度排查）
- **部署环境**: Ubuntu 24.04.4 LTS, 8 CPU, 8GB RAM
- **Kubernetes 版本**: v1.29.15
- **容器运行时**: containerd 2.2.2
- **网络插件**: Flannel v0.25.5 (docker.m.daocloud.io 镜像)
- **镜像仓库**: registry.aliyuncs.com/google_containers
- **部署结果**: 成功

**遇到的关键问题及解决方案**：

1. **etcd 旧数据导致启动慢**
   - 问题：etcd 有 125MB 旧数据，启动需要 2-3 分钟，kube-apiserver 20秒超时
   - 解决：完全重置集群时必须删除 `/var/lib/etcd/*`

2. **控制平面组件持续重启**
   - 问题：etcd 重启 136 次，kube-apiserver 重启 83 次
   - 解决：完全清理所有配置和数据目录后重新初始化

3. **CNI 插件缺失**
   - 问题：缺少 bridge、loopback 等 CNI 插件，CoreDNS 无法启动
   - 解决：需要从 containerd 快照或 Docker 快照复制完整的 CNI 插件

4. **kube-proxy 和 CoreDNS 缺失**
   - 问题：kubeadm init 最后一步因 API 服务器连接失败而跳过
   - 解决：使用 `kubeadm init phase addon all` 手动重新安装

**关键经验总结**：
- 重置集群时必须清理 **所有** 配置和数据目录，包括 `/var/lib/etcd/*`、`/var/lib/kubelet/*`、`/etc/kubernetes/*`
- 必须确保 CNI 插件完整（包括 bridge、loopback、portmap 等）
- 如果 kubeadm init 不完整，使用 `kubeadm init phase addon all` 补装
- 控制平面组件初始多次重启是正常现象，需耐心等待 2-5 分钟

---

### 2026-03-13 部署记录（旧版）
- **部署环境**: Ubuntu 24.04.4 LTS, 8 CPU, 8GB RAM
- **Kubernetes 版本**: v1.29.15
- **容器运行时**: containerd 2.2.2
- **网络插件**: Flannel v0.28.1
- **镜像仓库**: registry.aliyuncs.com/google_containers
- **部署结果**: 成功
- **备注**: 预拉取所有镜像后初始化，控制平面组件经过多次重启后稳定运行

---

**文档版本**: v2.1
**最后更新**: 2026-03-13
