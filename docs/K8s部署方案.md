# Kubernetes 集群部署方案

## 部署时间
2026-03-14

## 服务器环境
- 操作系统：Ubuntu 24.04.4 LTS
- 内核版本：6.8.0-101-generic
- IP地址：192.168.91.128
- 容器运行时：containerd 2.2.2

## 一、前期清理步骤

### 1. 连接服务器并检查当前状态
```bash
# 连接服务器（使用MCP服务）
# 检查服务状态
systemctl status kubelet docker containerd 2>&1 | head -50

# 检查容器状态
docker ps -a

# 检查上传的文件
ls -la /root/ | grep -E "\.(sh|yaml|yml|json)$"

# 检查Kubernetes相关目录
ls -la /var/lib/etcd/ 2>&1 && ls -la /var/lib/kubelet/ 2>&1 && ls -la /etc/kubernetes/ 2>&1
```

### 2. 停止并卸载Kubernetes所有组件
```bash
# 停止服务
systemctl stop docker && systemctl stop containerd
systemctl stop docker.socket

# 禁用服务
systemctl disable docker kubelet containerd

# 卸载Kubernetes组件
apt-get remove -y --allow-change-held-packages kubeadm kubectl kubelet kubernetes-cni
```

### 3. 清理Kubernetes配置文件和数据目录
```bash
rm -rf /etc/kubernetes/ /var/lib/kubelet/ /var/lib/etcd/ /root/.kube/ /opt/cni/bin/ /etc/cni/net.d/
```

### 4. 清理Docker容器网络和上传的文件
```bash
# 清理上传的文件
rm -f /root/*.sh /root/*.yaml /root/*.yml /root/*.json

# 清理容器相关目录
rm -rf /var/lib/calico /var/lib/cni /var/lib/docker /var/lib/containerd
```

### 5. 清理MySQL和Redis数据
```bash
rm -rf /var/lib/redis /var/lib/mysql /var/lib/mysqld
```

### 6. 验证清理完成情况
```bash
# 检查目录是否存在
ls -la /etc/kubernetes/ 2>&1 && ls -la /var/lib/kubelet/ 2>&1 && ls -la /var/lib/etcd/ 2>&1

# 检查服务状态
systemctl status docker containerd kubelet 2>&1 | grep -E "(Active:|Loaded:)"
```

## 二、重新部署Kubernetes集群

### 1. 启动containerd和docker服务
```bash
systemctl start containerd && systemctl start docker
```

### 2. 安装Kubernetes组件
```bash
# 安装依赖
apt-get install -y apt-transport-https curl

# 添加Kubernetes源
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.29/deb/Release.key | sudo gpg --dearmor -o /usr/share/keyrings/kubernetes-archive-keyring.gpg --batch --yes
echo 'deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.29/deb/ /' | tee /etc/apt/sources.list.d/kubernetes.list

# 更新源
apt-get update

# 安装Kubernetes组件
apt-get install -y kubeadm kubelet kubectl

# 锁定版本
apt-mark hold kubeadm kubelet kubectl
```

### 3. 配置containerd
```bash
# 创建containerd配置目录
mkdir -p /etc/containerd

# 生成默认配置
containerd config default > /etc/containerd/config.toml

# 修改配置，使用SystemdCgroup
sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

# 重启containerd
systemctl restart containerd

# 验证containerd状态
systemctl status containerd
```

### 4. 导入Docker镜像到containerd
```bash
# 导出所有docker镜像
docker save $(docker images --format '{{.Repository}}:{{.Tag}}' | grep -v '<none>') -o /tmp/all-images.tar

# 导入到containerd的k8s.io命名空间
ctr -n k8s.io images import /tmp/all-images.tar

# 验证导入成功
ctr -n k8s.io images list | grep pause
```

### 5. 初始化Kubernetes集群
```bash
# 重置kubeadm（如果之前有残留）
kubeadm reset -f

# 清理CNI配置
rm -rf /etc/cni/net.d/* && rm -rf $HOME/.kube/config

# 初始化集群
kubeadm init --apiserver-advertise-address=192.168.91.128 --pod-network-cidr=10.244.0.0/16 --image-repository=registry.aliyuncs.com/google_containers
```

**初始化输出：**
```
[init] Using Kubernetes version: v1.29.15
[preflight] Running pre-flight checks
[preflight] Pulling images required for setting up a Kubernetes cluster
[preflight] This might take a minute or two, depending on the speed of your internet connection
[preflight] You can also perform this action in beforehand using 'kubeadm config images pull'
[certs] Using certificateDir folder "/etc/kubernetes/pki"
[certs] Generating "ca" certificate and key
[certs] Generating "apiserver" certificate and key
[certs] apiserver serving cert is signed for DNS names [k8s-test kubernetes kubernetes.default kubernetes.default.svc kubernetes.default.svc.cluster.local] and IPs [10.96.0.1 192.168.91.128]
[certs] Generating "apiserver-kubelet-client" certificate and key
[certs] Generating "front-proxy-ca" certificate and key
[certs] Generating "front-proxy-client" certificate and key
[certs] Generating "etcd/ca" certificate and key
[certs] Generating "etcd/server" certificate and key
[certs] etcd/server serving cert is signed for DNS names [k8s-test localhost] and IPs [192.168.91.128 127.0.0.1 ::1]
[certs] Generating "etcd/peer" certificate and key
[certs] etcd/peer serving cert is signed for DNS names [k8s-test localhost] and IPs [192.168.91.128 127.0.0.1 ::1]
[certs] Generating "etcd/healthcheck-client" certificate and key
[certs] Generating "apiserver-etcd-client" certificate and key
[certs] Generating "sa" key and public key
[kubeconfig] Using kubeconfig folder "/etc/kubernetes"
[kubeconfig] Writing "admin.conf" kubeconfig file
[kubeconfig] Writing "super-admin.conf" kubeconfig file
[kubeconfig] Writing "kubelet.conf" kubeconfig file
[kubeconfig] Writing "controller-manager.conf" kubeconfig file
[kubeconfig] Writing "scheduler.conf" kubeconfig file
[etcd] Creating static Pod manifest for local etcd in "/etc/kubernetes/manifests"
[control-plane] Using manifest folder "/etc/kubernetes/manifests"
[control-plane] Creating static Pod manifest for "kube-apiserver"
[control-plane] Creating static Pod manifest for "kube-controller-manager"
[control-plane] Creating static Pod manifest for "kube-scheduler"
[kubelet-start] Writing kubelet environment file with flags to file "/var/lib/kubelet/kubeadm-flags.env"
[kubelet-start] Writing kubelet configuration to file "/var/lib/kubelet/config.yaml"
[kubelet-start] Starting the kubelet
[wait-control-plane] Waiting for the kubelet to boot up the control plane as static Pods from directory "/etc/kubernetes/manifests". This can take up to 4m0s
[apiclient] All control plane components are healthy after 4.001791 seconds
[upload-config] Storing the configuration used in ConfigMap "kubeadm-config" in the "kube-system" Namespace
[kubelet] Creating a ConfigMap "kubelet-config" in namespace kube-system with the configuration for the kubelets in the cluster
[upload-certs] Skipping phase. Please see --upload-certs
[mark-control-plane] Marking the node k8s-test as control-plane by adding the labels: [node-role.kubernetes.io/control-plane node.kubernetes.io/exclude-from-external-load-balancers]
[mark-control-plane] Marking the node k8s-test as control-plane by adding the taints [node-role.kubernetes.io/control-plane:NoSchedule]
[bootstrap-token] Using token: hf0vz0.fcw9g4k8yuqtf683
[bootstrap-token] Configuring bootstrap tokens, cluster-info ConfigMap, RBAC Roles
[bootstrap-token] Configured RBAC rules to allow Node Bootstrap tokens to get nodes
[bootstrap-token] Configured RBAC rules to allow Node Bootstrap tokens to post CSRs in order for nodes to get long term certificate credentials
[bootstrap-token] Configured RBAC rules to allow the csrapprover controller automatically approve CSRs from a Node Bootstrap Token
[bootstrap-token] Configured RBAC rules to allow certificate rotation for all node client certificates in the cluster
[bootstrap-token] Creating the "cluster-info" ConfigMap in the "kube-public" namespace
[kubelet-finalize] Updating "/etc/kubernetes/kubelet.conf" to point to a rotatable kubelet client certificate and key
[addons] Applied essential addon: CoreDNS
[addons] Applied essential addon: kube-proxy

Your Kubernetes control-plane has initialized successfully!

To start using your cluster, you need to run the following as a regular user:

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config

Alternatively, if you are the root user, you can run:

  export KUBECONFIG=/etc/kubernetes/admin.conf

You should now deploy a pod network to the cluster.
Run "kubectl apply -f [podnetwork].yaml" with one of the options listed at:
  https://kubernetes.io/docs/concepts/cluster-administration/addons/

Then you can join any number of worker nodes by running the following on each as root:

kubeadm join 192.168.91.128:6443 --token hf0vz0.fcw9g4k8yuqtf683 \
	--discovery-token-ca-cert-hash sha256:c38a786dc40132f86ffcbaa4a3f45ddebbe4aaadeb27ee571945628044b6dd3e 
```

### 6. 配置kubectl
```bash
mkdir -p $HOME/.kube && cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
```

### 7. 部署网络插件（Calico）
```bash
# 部署Calico
kubectl apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.28.2/manifests/calico.yaml

# 等待Calico Pod启动
sleep 30 && kubectl get pods -n kube-system | grep calico
```

## 三、验证集群状态

### 1. 检查节点状态
```bash
kubectl get nodes -o wide
```

**输出：**
```
NAME       STATUS   ROLES           AGE   VERSION    INTERNAL-IP      EXTERNAL-IP   OS-IMAGE             KERNEL-VERSION      CONTAINER-RUNTIME
k8s-test   Ready    control-plane   25m   v1.29.15   192.168.91.128   <none>        Ubuntu 24.04.4 LTS   6.8.0-101-generic   containerd://2.2.2
```

### 2. 检查所有Pod状态
```bash
kubectl get pods -A
```

**输出：**
```
NAMESPACE     NAME                                       READY   STATUS    RESTARTS   AGE
kube-system   calico-kube-controllers-658958974b-lblv9   1/1     Running   0          40s
kube-system   calico-node-rqf56                          1/1     Running   0          40s
kube-system   coredns-857d9ff4c9-hscl2                   1/1     Running   0          25m
kube-system   coredns-857d9ff4c9-qskss                   1/1     Running   0          25m
kube-system   etcd-k8s-test                              1/1     Running   202        25m
kube-system   kube-apiserver-k8s-test                    1/1     Running   151        25m
kube-system   kube-controller-manager-k8s-test           1/1     Running   169        25m
kube-system   kube-proxy-hsp4v                           1/1     Running   0          25m
kube-system   kube-scheduler-k8s-test                    1/1     Running   171        25m
```

### 3. 测试集群网络功能
```bash
# 创建测试Pod
kubectl run test-pod --image=nginx --restart=Never

# 查看Pod状态
kubectl get pods -o wide

# 测试网络连通性
kubectl exec -it test-pod -- /bin/bash
# 在Pod内执行：ping www.baidu.com

# 删除测试Pod
kubectl delete pod test-pod
```

## 四、配置文件和目录

### 1. 主要配置文件
- **kubeadm配置文件**：/etc/kubernetes/kubeadm.conf
- **kubectl配置文件**：$HOME/.kube/config
- **kubelet配置文件**：/var/lib/kubelet/config.yaml
- **containerd配置文件**：/etc/containerd/config.toml
- **Kubernetes源配置**：/etc/apt/sources.list.d/kubernetes.list

### 2. 重要目录
- **Kubernetes配置目录**：/etc/kubernetes/
- **kubelet数据目录**：/var/lib/kubelet/
- **etcd数据目录**：/var/lib/etcd/
- **CNI配置目录**：/etc/cni/net.d/
- **容器运行时目录**：/var/lib/containerd/
- **Docker目录**：/var/lib/docker/

### 3. 静态Pod清单
- **kube-apiserver**：/etc/kubernetes/manifests/kube-apiserver.yaml
- **kube-controller-manager**：/etc/kubernetes/manifests/kube-controller-manager.yaml
- **kube-scheduler**：/etc/kubernetes/manifests/kube-scheduler.yaml
- **etcd**：/etc/kubernetes/manifests/etcd.yaml

### 4. 配置文件详细内容

#### containerd配置文件 (/etc/containerd/config.toml)
```toml
version = 2
root = "/var/lib/containerd"
state = "/run/containerd"
imports = ["/etc/containerd/conf.d/*.toml"]

grpc {
  address = "/run/containerd/containerd.sock"
  tcp_address = ""
  tcp_tls_cert = ""
  tcp_tls_key = ""
  uid = 0
  gid = 0
  max_recv_message_size = 16777216
  max_send_message_size = 16777216
}

debug = false

metrics {
  address = ""
  grpc_histogram = false
}

tls {
  debug = false
  cert_file = ""
  key_file = ""
  roots_file = ""
  client_ca_file = ""
  
}

plugins {
  io.containerd.gc.v1.scheduler {
    pause_threshold = 0.02
    deletion_threshold = 0
    mutation_threshold = 100
    schedule_delay = "0s"
    startup_delay = "100ms"
  }
  io.containerd.runtime.v1.linux {
    shim = "containerd-shim"
    runtime = "runc"
    runtime_root = ""
    no_shim = false
    shim_debug = false
  }
  io.containerd.runtime.v2.task {
    platforms = ["linux/amd64"]
  }
  io.containerd.monitor.v1.cgroups {
    no_prometheus = false
  }
  io.containerd.service.v1.diff-service {
    default = "walking"
  }
  io.containerd.grpc.v1.cri {
    disable_tcp_service = true
    stream_server_address = "127.0.0.1"
    stream_server_port = "0"
    stream_idle_timeout = "4h0m0s"
    enable_selinux = false
    selinux_category_range = 1024
    sandbox_image = "registry.aliyuncs.com/google_containers/pause:3.10"
    stats_collect_period = 10
    systemd_cgroup = true
    enable_tls_streaming = false
    max_container_log_line_size = 16384
    disable_cgroup = false
    disable_apparmor = false
    restrict_oom_score_adj = false
    max_concurrent_downloads = 3
    disable_proc_mount = false
    unset_seccomp_profile = ""
    tolerate_missing_hugetlb_controller = true
    disable_hugetlb_controller = false
    ignore_image_defined_volumes = false
    disable_compression = false
    [plugins."io.containerd.grpc.v1.cri".containerd]
      snapshotter = "overlayfs"
      default_runtime_name = "runc"
      no_pivot = false
      disable_snapshot_annotations = false
      discard_unpacked_layers = false
      [plugins."io.containerd.grpc.v1.cri".containerd.default_runtime]
        runtime_type = ""
        runtime_engine = ""
        runtime_root = ""
        privileged_without_host_devices = false
        base_runtime_spec = ""
      [plugins."io.containerd.grpc.v1.cri".containerd.untrusted_workload_runtime]
        runtime_type = ""
        runtime_engine = ""
        runtime_root = ""
        privileged_without_host_devices = false
        base_runtime_spec = ""
      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"
          runtime_engine = ""
          runtime_root = ""
          privileged_without_host_devices = false
          base_runtime_spec = ""
    [plugins."io.containerd.grpc.v1.cri".cni]
      bin_dir = "/opt/cni/bin"
      conf_dir = "/etc/cni/net.d"
      max_conf_num = 1
      conf_template = ""
    [plugins."io.containerd.grpc.v1.cri".registry]
      config_path = ""
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
          endpoint = ["https://docker.mirrors.ustc.edu.cn", "https://registry-1.docker.io"]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."gcr.io"]
          endpoint = ["https://gcr.mirrors.ustc.edu.cn"]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."k8s.gcr.io"]
          endpoint = ["https://gcr.mirrors.ustc.edu.cn/google-containers/"]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."quay.io"]
          endpoint = ["https://quay.mirrors.ustc.edu.cn"]
}

proxy_plugins {}

opt {
  "io.containerd.nri.v1.nri" = false
}
```

#### Kubernetes源配置文件 (/etc/apt/sources.list.d/kubernetes.list)
```
deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.29/deb/ /
```

## 五、部署 etcdkeeper 服务

### 1. 清理现有 etcdkeeper 资源
```bash
# 查看现有的 etcdkeeper Pod
kubectl get pods -A | grep etcdkeeper

# 删除现有的 etcdkeeper Pod
kubectl delete pod etcdkeeper-57bdbcff8c-8mxtx etcdkeeper-cc4c75cdb-m75xj -n kube-system

# 查看并删除 etcdkeeper Deployment
kubectl get deployment -n kube-system | grep etcdkeeper
kubectl delete deployment etcdkeeper -n kube-system

# 查看并删除 etcdkeeper Service
kubectl get svc -n kube-system | grep etcdkeeper
kubectl delete svc etcdkeeper -n kube-system
```

### 2. 重新部署 etcdkeeper

#### 2.1 创建 Deployment 配置
```bash
cat > etcdkeeper-deployment.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: etcdkeeper
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: etcdkeeper
  template:
    metadata:
      labels:
        app: etcdkeeper
    spec:
      tolerations:
      - key: node-role.kubernetes.io/control-plane
        operator: Exists
        effect: NoSchedule
      containers:
      - name: etcdkeeper
        image: deltaprojects/etcdkeeper:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
        env:
        - name: ETCD_HOST
          value: "etcd-cluster.kube-system.svc.cluster.local"
        - name: ETCD_PORT
          value: "2379"
EOF
```

#### 2.2 创建 Service 配置
```bash
cat > etcdkeeper-service.yaml << 'EOF'
apiVersion: v1
kind: Service
metadata:
  name: etcdkeeper
  namespace: kube-system
spec:
  selector:
    app: etcdkeeper
  ports:
  - port: 8080
    targetPort: 8080
  type: NodePort
EOF
```

#### 2.3 应用配置
```bash
kubectl apply -f etcdkeeper-deployment.yaml
kubectl apply -f etcdkeeper-service.yaml
```

### 3. 验证部署结果
```bash
# 查看 Pod 状态
kubectl get pods -n kube-system | grep etcdkeeper

# 查看 Service 状态
kubectl get svc etcdkeeper -n kube-system

# 查看详细日志
kubectl logs -f deployment/etcdkeeper -n kube-system
```

### 4. 访问 etcdkeeper
- **访问地址**：`http://<节点IP>:<NodePort>`
- **示例**：`http://192.168.91.128:32079`
- **功能**：通过 Web 界面管理和监控 etcd 集群

### 5. 故障排查
- **Pod 无法调度**：添加对控制平面节点污点的容忍度（已在配置中添加）
- **ImagePullBackOff**：确保本地有 `deltaprojects/etcdkeeper:latest` 镜像，或修改 `imagePullPolicy` 为 `Never`
- **无法连接 etcd**：检查 etcd 服务地址是否正确，确保 etcd 集群正常运行

## 六、常用命令

### 1. 集群状态查看
- **查看节点状态**：`kubectl get nodes -o wide`
- **查看Pod状态**：`kubectl get pods -A`
- **查看服务状态**：`kubectl get services -A`
- **查看集群组件状态**：`kubectl get componentstatuses`

### 2. 日志查看
- **查看kubelet日志**：`journalctl -u kubelet -f`
- **查看容器日志**：`kubectl logs <pod-name> -n <namespace>`
- **查看事件**：`kubectl get events -A --sort-by='.lastTimestamp'`

### 3. 集群管理
- **重启kubelet**：`systemctl restart kubelet`
- **查看集群信息**：`kubectl cluster-info`
- **检查集群健康状态**：`kubectl get --raw='/healthz'`

## 六、注意事项

### 1. 网络问题
- **镜像拉取失败**：如果遇到`ImagePullBackOff`错误，检查网络连接或使用国内镜像源
- **网络插件部署**：Flannel在国内网络环境下载较慢，推荐使用Calico
- **防火墙设置**：确保服务器防火墙允许Kubernetes所需的端口
- **网络连通性**：部署后测试Pod网络连通性，确保集群内部和外部网络正常

### 2. 存储问题
- **etcd数据**：定期备份etcd数据，避免数据丢失
- **磁盘空间**：确保服务器有足够的磁盘空间，特别是/var/lib/目录
- **数据目录权限**：确保各数据目录权限正确，避免权限不足导致服务启动失败

### 3. 服务管理
- **自动启动**：启用kubelet服务开机自启：`systemctl enable kubelet`
- **服务监控**：建议部署Prometheus和Grafana进行监控
- **服务状态检查**：定期检查各组件服务状态，确保正常运行

### 4. 安全注意事项
- **证书管理**：Kubernetes证书默认有效期为1年，需要定期更新
- **RBAC配置**：根据实际需求配置适当的RBAC权限
- **节点安全**：定期更新服务器系统和组件版本
- **敏感信息**：避免在配置文件中存储敏感信息，使用Secret管理

### 5. 故障排查
- **节点NotReady**：检查网络插件状态和kubelet日志
- **Pod调度失败**：检查节点资源是否充足
- **API服务器无响应**：检查etcd状态和网络连接
- **容器启动失败**：检查容器日志和镜像拉取状态
- **kubeadm初始化失败**：检查containerd配置和镜像源设置
- **网络插件部署失败**：检查CNI配置和网络策略

### 6. 本次部署遇到的问题及解决方案
- **SSH连接失败**：使用`mcp_ssh-mcp-server_ssh_list_connections`查看活动连接，直接使用现有连接
- **包管理工具错误**：Ubuntu系统使用`apt-get`替代`yum`或`dnf`
- **containerd配置错误**：修改`SystemdCgroup`为`true`，并设置国内镜像源
- **pause镜像拉取失败**：修改containerd配置，使用阿里云镜像源
- **Flannel镜像拉取超时**：改用Calico网络插件解决
- **kubeadm初始化失败**：确保containerd配置正确、镜像已导入、kubelet服务正常运行

### 7. 最佳实践
- **镜像管理**：提前下载所需镜像，避免部署过程中因网络问题导致失败
- **配置备份**：定期备份Kubernetes配置文件和etcd数据
- **版本管理**：使用固定版本的Kubernetes组件，避免版本不兼容问题
- **文档记录**：详细记录部署过程和配置变更，便于后续维护
- **资源规划**：根据实际需求合理规划集群资源，避免资源不足

### 8. 部署前检查清单
- [ ] 服务器硬件满足Kubernetes最低要求
- [ ] 网络连接正常，能够访问镜像源
- [ ] 磁盘空间充足，特别是/var/lib/目录
- [ ] 系统内核版本满足要求
- [ ] 容器运行时已正确安装和配置
- [ ] 防火墙规则已正确配置
- [ ] 时间同步服务正常运行
- [ ] 所有依赖包已安装

### 9. 部署后验证清单
- [ ] 集群节点状态为Ready
- [ ] 控制平面组件全部Running
- [ ] 网络插件部署成功
- [ ] CoreDNS服务正常运行
- [ ] Pod网络连通性测试通过
- [ ] 集群API服务器可正常访问
- [ ] 节点资源使用情况正常

## 七、版本信息

- **Kubernetes版本**：v1.29.15
- **Containerd版本**：2.2.2
- **Calico版本**：v3.28.2
- **CoreDNS版本**：v1.11.1
- **etcd版本**：3.5.16-0

## 八、部署成功验证

✅ **集群状态**：Ready
✅ **控制平面组件**：全部Running
✅ **网络插件**：Calico Running
✅ **CoreDNS**：Running
✅ **节点状态**：Ready
✅ **网络连通性**：正常

集群已成功部署并运行正常，可以开始部署应用程序。