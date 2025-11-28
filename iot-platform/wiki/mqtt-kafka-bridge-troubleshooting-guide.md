# MQTT-Kafka Bridge 部署故障排查手册

## 📋 项目信息

**项目名称**: MQTT-Kafka Bridge for IoT Platform  
**部署环境**: AWS EKS (Kubernetes)  
**Git 仓库**: git@github.com:misky530/eks.git  
**项目路径**: iot-platform/mqtt-kafka-bridge  
**AWS 账号**: 645890933537  
**区域**: us-east-1  
**集群名称**: iot-platform  
**命名空间**: iot-bridge

---

## 🏗️ 架构概述

```
MQTT Broker (hats.hcs.cn:1883)
    ↓ 订阅主题
MQTT-Kafka Bridge (Go Application)
    ↓ 转发消息
Kafka Cluster (iot-cluster-kafka-bootstrap.kafka:9092)
```

**应用配置**:
- **语言**: Go 1.21
- **容器**: Alpine Linux (18MB 镜像)
- **副本数**: 1 (由于节点容量限制)
- **资源限制**: 50m CPU / 64Mi Memory (requests), 200m CPU / 128Mi Memory (limits)

---

## 🔧 部署流程

### 1. Docker 镜像构建与推送

```bash
# 构建镜像
docker build -t mqtt-kafka-bridge:latest .

# 登录 ECR
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin \
  645890933537.dkr.ecr.us-east-1.amazonaws.com

# 标记镜像
docker tag mqtt-kafka-bridge:latest \
  645890933537.dkr.ecr.us-east-1.amazonaws.com/mqtt-kafka-bridge:latest

# 推送到 ECR
docker push 645890933537.dkr.ecr.us-east-1.amazonaws.com/mqtt-kafka-bridge:latest
```

### 2. ArgoCD 部署

```bash
# 应用 ArgoCD 配置
cd iot-platform/mqtt-kafka-bridge/deployments/kubernetes
kubectl apply -f argocd-app.yaml

# 查看同步状态
kubectl get application -n argocd mqtt-kafka-bridge
```

### 3. 验证部署

```bash
# 查看 Pod 状态
kubectl get pods -n iot-bridge

# 查看日志
kubectl logs -n iot-bridge -l app=mqtt-kafka-bridge

# 查看所有资源
kubectl get all -n iot-bridge
```

---

## ⚠️ 问题与解决方案

### 问题 1: Docker 构建失败 - 缺少 pkg 目录

**错误信息**:
```
COPY pkg/ ./pkg/: not found
```

**原因分析**:
- 空的 `pkg/` 目录被 Git 忽略
- Dockerfile 仍然引用该目录

**解决方案**:
```dockerfile
# 删除 Dockerfile 中的这一行
# COPY pkg/ ./pkg/
```

**学到的经验**:
- 在 Dockerfile 中只复制实际存在且需要的文件
- 对于可选目录，使用条件复制或移除引用

---

### 问题 2: Go 模块校验失败

**错误信息**:
```
verifying github.com/klauspost/compress@v1.17.4/go.mod: checksum mismatch
downloaded: h1:xyz...
go.sum:     h1:abc...
```

**原因分析**:
- `go.sum` 文件包含过期的校验和
- 依赖版本可能已更新

**解决方案**:
```bash
# 删除 go.sum
rm go.sum

# 修改 Dockerfile，在 go mod download 前添加 go mod tidy
RUN go mod tidy
RUN go mod download
RUN go mod verify
```

**Dockerfile 最佳实践**:
```dockerfile
# 第一阶段：构建
FROM golang:1.21-alpine AS builder
WORKDIR /app

# 复制 go.mod 和源代码
COPY go.mod ./
COPY cmd/ ./cmd/

# 整理依赖并下载
RUN go mod tidy
RUN go mod download
RUN go mod verify

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -o bridge ./cmd/bridge

# 第二阶段：运行
FROM alpine:3.19
RUN apk --no-cache add ca-certificates procps
WORKDIR /root/
COPY --from=builder /app/bridge .
CMD ["./bridge"]
```

**学到的经验**:
- 使用 `go mod tidy` 清理和更新依赖
- 在 CI/CD 环境中不要提交 `go.sum`，让构建过程自动生成
- 使用 `go mod verify` 确保依赖完整性

---

### 问题 3: Go 代码编译错误 - 未使用的变量

**错误信息**:
```
cmd/bridge/main.go:121:3: ctx declared and not used
```

**原因分析**:
- Stop() 函数中声明了 `context.WithTimeout` 但未使用
- Go 编译器严格检查未使用的变量

**解决方案**:
```go
// 删除未使用的代码
func (b *Bridge) Stop() error {
    b.logger.Info("Stopping bridge...")
    
    // 删除这两行
    // ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    // defer cancel()
    
    // 关闭连接
    if b.mqttClient.IsConnected() {
        b.mqttClient.Disconnect(250)
    }
    // ...
}
```

**学到的经验**:
- Go 不允许未使用的变量和导入
- 使用 `gofmt` 和 `go vet` 在提交前检查代码
- 在 CI/CD 中添加代码质量检查

---

### 问题 4: ArgoCD 配置错误 - Git 仓库路径不正确

**错误信息**:
```
connection error: desc = transport: Error while dialing: 
dial tcp 172.20.165.178:8081: connect: connection refused
```

**原因分析**:
- ArgoCD `path` 配置错误
- 实际仓库结构：`eks/iot-platform/mqtt-kafka-bridge/deployments/kubernetes`
- 配置中使用：`mqtt-kafka-bridge/deployments/kubernetes`（缺少 `iot-platform/`）

**解决方案**:
```yaml
# argocd-app.yaml
spec:
  source:
    repoURL: https://github.com/misky530/eks
    targetRevision: HEAD
    path: iot-platform/mqtt-kafka-bridge/deployments/kubernetes  # 修正路径
```

```bash
# 修改配置
cd iot-platform/mqtt-kafka-bridge/deployments/kubernetes
sed -i 's|path: mqtt-kafka-bridge/deployments/kubernetes|path: iot-platform/mqtt-kafka-bridge/deployments/kubernetes|g' argocd-app.yaml

# 提交到 Git
git add argocd-app.yaml
git commit -m "Fix ArgoCD application path"
git push
```

**学到的经验**:
- ArgoCD 的 `path` 必须是从 Git 仓库根目录开始的相对路径
- 使用 `tree` 或 `ls -R` 验证目录结构
- 在 ArgoCD UI 中可以看到路径解析错误

**验证方法**:
```bash
# 克隆仓库并验证路径
git clone https://github.com/misky530/eks /tmp/test-repo
ls -la /tmp/test-repo/iot-platform/mqtt-kafka-bridge/deployments/kubernetes
```

---

### 问题 5: Kubernetes ImagePullBackOff - ECR 镜像路径错误

**错误信息**:
```
Failed to pull image "645890933537.dkr.ecr.us-east-1.amazonaws.com/mqtt-kafka-bridge/mqtt-kafka-bridge:latest": 
rpc error: code = NotFound desc = failed to pull and unpack image: not found
```

**原因分析**:
- ECR 仓库名称：`mqtt-kafka-bridge`
- 错误的镜像路径：`mqtt-kafka-bridge/mqtt-kafka-bridge:latest`（重复了仓库名）
- 正确的镜像路径：`mqtt-kafka-bridge:latest`

**解决方案**:
```yaml
# deployment.yaml - 修改前
spec:
  containers:
  - name: bridge
    image: 645890933537.dkr.ecr.us-east-1.amazonaws.com/mqtt-kafka-bridge/mqtt-kafka-bridge:latest

# deployment.yaml - 修改后
spec:
  containers:
  - name: bridge
    image: 645890933537.dkr.ecr.us-east-1.amazonaws.com/mqtt-kafka-bridge:latest
```

```bash
# 修改配置
sed -i 's|mqtt-kafka-bridge/mqtt-kafka-bridge:latest|mqtt-kafka-bridge:latest|g' deployments/kubernetes/deployment.yaml

# 提交到 Git
git add deployments/kubernetes/deployment.yaml
git commit -m "Fix ECR image path"
git push
```

**学到的经验**:
- ECR 镜像完整路径格式：`<account-id>.dkr.ecr.<region>.amazonaws.com/<repository-name>:<tag>`
- 不要在镜像路径中重复仓库名称
- 使用 `aws ecr describe-images` 验证镜像存在

**验证方法**:
```bash
# 列出 ECR 仓库中的镜像
aws ecr describe-images \
  --repository-name mqtt-kafka-bridge \
  --region us-east-1

# 验证镜像 URI
aws ecr describe-repositories \
  --repository-names mqtt-kafka-bridge \
  --region us-east-1 \
  --query 'repositories[0].repositoryUri'
```

---

### 问题 6: Pod 调度失败 - 节点容量不足 ⭐ 最关键问题

**错误信息**:
```
Events:
  Type     Reason            Age   From               Message
  ----     ------            ----  ----               -------
  Warning  FailedScheduling  77s   default-scheduler  
    0/2 nodes are available: 2 Too many pods. 
    preemption: 0/2 nodes are available: 2 No preemption victims found for incoming pod.
```

**原因分析**:
- EKS 集群有 2 个 t3.small 节点
- t3.small 每个节点最多支持 11 个 Pod（AWS ENI 限制）
- 两个节点的 Pod 槽位都已满
- 新 Pod 无法调度

**节点容量计算**:
```
t3.small:
- vCPU: 2
- Memory: 2 GiB
- 最大 Pod 数: 11 (由 ENI 和 IP 地址数量决定)
- 可用 Pod 数 = 11 - 系统 Pod (kube-proxy, CNI 等)
```

**解决方案 1: 减少副本数（临时方案）**:
```bash
# 缩减副本数到 1
kubectl scale deployment mqtt-kafka-bridge -n iot-bridge --replicas=1

# 删除旧的失败 Pod
kubectl delete pod -n iot-bridge -l app=mqtt-kafka-bridge --force --grace-period=0
```

```yaml
# deployment.yaml - 修改副本数
spec:
  replicas: 1  # 从 2 改为 1
```

**解决方案 2: 扩展节点组（长期方案）**:
```bash
# 查看当前节点组配置
aws eks describe-nodegroup \
  --cluster-name iot-platform \
  --nodegroup-name <your-nodegroup-name>

# 增加节点数量
aws eks update-nodegroup-config \
  --cluster-name iot-platform \
  --nodegroup-name <your-nodegroup-name> \
  --scaling-config minSize=2,maxSize=4,desiredSize=3
```

**解决方案 3: 使用更大的实例类型（推荐）**:
```bash
# 创建新的节点组（t3.medium）
aws eks create-nodegroup \
  --cluster-name iot-platform \
  --nodegroup-name iot-platform-t3-medium \
  --instance-types t3.medium \
  --scaling-config minSize=2,maxSize=4,desiredSize=2 \
  --subnets subnet-xxx subnet-yyy \
  --node-role arn:aws:iam::645890933537:role/EKSNodeRole
```

**实例类型对比**:
| 实例类型 | vCPU | Memory | 最大 Pod 数 | 建议场景 |
|---------|------|--------|-----------|---------|
| t3.small | 2 | 2 GiB | 11 | 开发/测试 |
| t3.medium | 2 | 4 GiB | 17 | 小型生产 |
| t3.large | 2 | 8 GiB | 35 | 中型生产 |

**学到的经验**:
- AWS EKS 节点的最大 Pod 数受 ENI 和 IP 地址限制
- 规划集群容量时要考虑系统 Pod 的占用
- 使用 `kubectl describe nodes` 查看节点 Pod 分配情况
- 生产环境建议使用 t3.medium 或更大实例

**容量规划公式**:
```
可用 Pod 槽位 = (最大 Pod 数 × 节点数) - 系统 Pod 数
推荐预留: 至少 20% 的槽位用于扩容
```

**诊断命令**:
```bash
# 查看所有 Pod 分布
kubectl get pods --all-namespaces -o wide | \
  awk '{print $8}' | sort | uniq -c | sort -rn

# 查看节点详细 Pod 列表
kubectl describe nodes | grep -E "^Name:|Non-terminated Pods:" -A 15

# 查看节点资源使用
kubectl top nodes

# 统计总 Pod 数
kubectl get pods --all-namespaces | wc -l
```

---

### 问题 7: ArgoCD Application 删除卡住

**现象**:
- 执行 `kubectl delete application` 后命令卡住
- Application 处于 Terminating 状态

**原因分析**:
- ArgoCD Finalizer 在等待资源清理
- 可能存在资源依赖关系

**解决方案**:
```bash
# 方案 1: 等待自然删除（推荐）
# 通常会在 30-60 秒内完成

# 方案 2: 强制删除 Finalizer
kubectl patch application mqtt-kafka-bridge -n argocd \
  -p '{"metadata":{"finalizers":[]}}' \
  --type=merge

# 方案 3: 重新创建（删除后立即创建）
kubectl delete application -n argocd mqtt-kafka-bridge
kubectl apply -f argocd-app.yaml
```

**学到的经验**:
- ArgoCD 使用 Finalizer 确保资源清理
- 删除时要有耐心等待
- 避免频繁删除和重建 Application

---

## 🎯 最佳实践总结

### 1. Docker 镜像构建

**✅ 推荐做法**:
- 使用多阶段构建减小镜像体积
- 在构建阶段使用 `go mod tidy` 和 `go mod verify`
- 不要在镜像中包含源代码（除非必要）
- 使用 Alpine 作为运行时基础镜像

**❌ 避免做法**:
- 不要提交 `go.sum` 到版本控制（在 CI/CD 中生成）
- 不要在 Dockerfile 中硬编码版本号
- 不要复制不存在的目录

### 2. ECR 镜像管理

**✅ 推荐做法**:
```bash
# 使用语义化版本标签
docker tag app:latest ${ECR_REPO}:v1.0.0
docker tag app:latest ${ECR_REPO}:latest

# 推送多个标签
docker push ${ECR_REPO}:v1.0.0
docker push ${ECR_REPO}:latest
```

**镜像命名规范**:
```
格式: <account-id>.dkr.ecr.<region>.amazonaws.com/<repository>:<tag>
示例: 645890933537.dkr.ecr.us-east-1.amazonaws.com/mqtt-kafka-bridge:v1.0.0
```

### 3. Kubernetes 部署

**✅ 推荐配置**:
```yaml
# 资源限制
resources:
  requests:
    cpu: 50m
    memory: 64Mi
  limits:
    cpu: 200m
    memory: 128Mi

# 健康检查
livenessProbe:
  exec:
    command: ["pgrep", "bridge"]
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  exec:
    command: ["pgrep", "bridge"]
  initialDelaySeconds: 5
  periodSeconds: 10

# 副本数
replicas: 1  # 根据节点容量调整
```

### 4. ArgoCD 配置

**✅ 推荐配置**:
```yaml
spec:
  source:
    repoURL: https://github.com/username/repo
    targetRevision: HEAD
    path: full/path/from/repo/root  # 使用完整路径
  
  destination:
    server: https://kubernetes.default.svc
    namespace: app-namespace
  
  syncPolicy:
    automated:
      prune: true      # 自动删除不在 Git 中的资源
      selfHeal: true   # 自动修复配置漂移
    syncOptions:
      - CreateNamespace=true  # 自动创建命名空间
```

### 5. 容量规划

**节点规划建议**:
```
开发环境: 2-3 个 t3.small 节点
测试环境: 2-4 个 t3.medium 节点
生产环境: 3+ 个 t3.large 或 t3.xlarge 节点
```

**Pod 密度计算**:
```python
# 计算可用 Pod 槽位
def calculate_pod_capacity(instance_type, node_count):
    max_pods_per_node = {
        't3.small': 11,
        't3.medium': 17,
        't3.large': 35,
        't3.xlarge': 58
    }
    
    total_slots = max_pods_per_node[instance_type] * node_count
    system_pods = 5 * node_count  # 每节点约 5 个系统 Pod
    available_slots = total_slots - system_pods
    
    return available_slots

# 示例
print(calculate_pod_capacity('t3.small', 2))   # 12 个可用槽位
print(calculate_pod_capacity('t3.medium', 2))  # 24 个可用槽位
```

---

## 📊 故障排查流程图

```
部署失败
    ↓
检查 ArgoCD Application 状态
    ├─ OutOfSync → 检查 Git 路径配置
    ├─ Degraded → 检查 Pod 状态
    └─ Healthy → 部署成功
         ↓
检查 Pod 状态
    ├─ ImagePullBackOff → 检查镜像路径和 ECR 权限
    ├─ CrashLoopBackOff → 检查应用日志
    ├─ Pending → 检查节点资源和调度器
    └─ Running → 检查应用日志
         ↓
检查应用日志
    ├─ MQTT 连接失败 → 检查网络和 MQTT Broker
    ├─ Kafka 连接失败 → 检查 Kafka 集群状态
    └─ 正常运行 → 部署成功
```

---

## 🔍 常用诊断命令

### Pod 相关
```bash
# 查看 Pod 状态
kubectl get pods -n iot-bridge
kubectl describe pod -n iot-bridge <pod-name>
kubectl logs -n iot-bridge <pod-name> --tail=100
kubectl logs -n iot-bridge <pod-name> --previous  # 查看上一次运行的日志

# 进入 Pod 调试
kubectl exec -it -n iot-bridge <pod-name> -- sh

# 查看 Pod 事件
kubectl get events -n iot-bridge --sort-by='.lastTimestamp'
```

### 节点相关
```bash
# 查看节点状态
kubectl get nodes
kubectl describe nodes

# 查看节点资源使用
kubectl top nodes

# 查看节点上的 Pod 分布
kubectl get pods --all-namespaces -o wide | grep <node-name>

# 查看节点容量
kubectl describe node <node-name> | grep -A 5 "Allocatable"
```

### ArgoCD 相关
```bash
# 查看 Application 状态
kubectl get application -n argocd
kubectl describe application -n argocd <app-name>

# 查看 ArgoCD 日志
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-application-controller

# 手动触发同步
kubectl patch application <app-name> -n argocd \
  --type merge \
  -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'
```

### ECR 相关
```bash
# 列出仓库
aws ecr describe-repositories --region us-east-1

# 列出镜像
aws ecr describe-images \
  --repository-name mqtt-kafka-bridge \
  --region us-east-1

# 查看镜像详情
aws ecr batch-get-image \
  --repository-name mqtt-kafka-bridge \
  --image-ids imageTag=latest \
  --region us-east-1 \
  --query 'images[0].imageManifest' \
  --output text | jq .
```

---

## 🚀 快速修复脚本

### 脚本 1: 清理并重新部署

```bash
#!/bin/bash
# cleanup-redeploy.sh

set -e

NAMESPACE="iot-bridge"
APP_NAME="mqtt-kafka-bridge"

echo "🔧 开始清理和重新部署..."

# 1. 删除 ArgoCD Application
echo "删除 ArgoCD Application..."
kubectl delete application -n argocd ${APP_NAME} || true
sleep 5

# 2. 删除命名空间中的所有资源
echo "清理命名空间 ${NAMESPACE}..."
kubectl delete deployment,service,configmap,secret -n ${NAMESPACE} -l app=${APP_NAME} || true
sleep 5

# 3. 重新创建 ArgoCD Application
echo "重新创建 ArgoCD Application..."
kubectl apply -f deployments/kubernetes/argocd-app.yaml

# 4. 等待同步
echo "等待 ArgoCD 同步..."
sleep 15

# 5. 检查状态
echo "=== ArgoCD Application 状态 ==="
kubectl get application -n argocd ${APP_NAME}

echo "=== Pod 状态 ==="
kubectl get pods -n ${NAMESPACE}

echo "✅ 完成！"
```

### 脚本 2: 健康检查

```bash
#!/bin/bash
# health-check.sh

NAMESPACE="iot-bridge"
APP_NAME="mqtt-kafka-bridge"

echo "🔍 健康检查开始..."

# 1. ArgoCD 状态
echo "=== ArgoCD Application ==="
kubectl get application -n argocd ${APP_NAME}

# 2. Pod 状态
echo ""
echo "=== Pods ==="
kubectl get pods -n ${NAMESPACE} -l app=${APP_NAME}

# 3. Service 状态
echo ""
echo "=== Services ==="
kubectl get svc -n ${NAMESPACE}

# 4. 最近的事件
echo ""
echo "=== 最近事件 ==="
kubectl get events -n ${NAMESPACE} --sort-by='.lastTimestamp' | tail -10

# 5. 应用日志（最后 20 行）
echo ""
echo "=== 应用日志 ==="
POD_NAME=$(kubectl get pods -n ${NAMESPACE} -l app=${APP_NAME} -o jsonpath='{.items[0].metadata.name}')
if [ ! -z "$POD_NAME" ]; then
    kubectl logs -n ${NAMESPACE} ${POD_NAME} --tail=20
else
    echo "未找到运行中的 Pod"
fi

echo ""
echo "✅ 健康检查完成！"
```

### 脚本 3: 容量检查

```bash
#!/bin/bash
# capacity-check.sh

echo "📊 集群容量检查..."

# 1. 节点资源
echo "=== 节点资源使用 ==="
kubectl top nodes 2>/dev/null || echo "Metrics Server 未安装"

# 2. Pod 分布
echo ""
echo "=== Pod 分布 ==="
kubectl get pods --all-namespaces -o wide | \
  awk 'NR>1 {print $8}' | sort | uniq -c | sort -rn

# 3. 节点 Pod 容量
echo ""
echo "=== 节点 Pod 容量 ==="
for node in $(kubectl get nodes -o jsonpath='{.items[*].metadata.name}'); do
    echo "Node: $node"
    kubectl describe node $node | grep -A 5 "Allocated resources:" | grep pods
    echo ""
done

# 4. 总 Pod 数
echo "=== 总 Pod 数 ==="
TOTAL_PODS=$(kubectl get pods --all-namespaces --no-headers | wc -l)
echo "集群总 Pod 数: $TOTAL_PODS"

echo ""
echo "✅ 容量检查完成！"
```

---

## 📚 参考资料

### AWS 文档
- [EKS 节点实例类型](https://docs.aws.amazon.com/eks/latest/userguide/choosing-instance-type.html)
- [ECR 用户指南](https://docs.aws.amazon.com/ecr/latest/userguide/)
- [EKS Pod 网络限制](https://docs.aws.amazon.com/eks/latest/userguide/pod-networking.html)

### Kubernetes 文档
- [Pod 调度](https://kubernetes.io/docs/concepts/scheduling-eviction/)
- [资源管理](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- [健康检查](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)

### ArgoCD 文档
- [ArgoCD 最佳实践](https://argo-cd.readthedocs.io/en/stable/user-guide/best_practices/)
- [同步策略](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-options/)

---

## 📝 检查清单

### 部署前检查
- [ ] Docker 镜像已构建并推送到 ECR
- [ ] Git 仓库配置正确
- [ ] ArgoCD Application 路径正确
- [ ] 节点有足够的 Pod 槽位
- [ ] 命名空间已创建或配置为自动创建
- [ ] 资源限制合理设置

### 部署后检查
- [ ] ArgoCD Application 状态为 Synced 和 Healthy
- [ ] Pod 状态为 Running
- [ ] 应用日志显示正常启动
- [ ] MQTT 连接成功
- [ ] Kafka 连接成功
- [ ] 健康检查通过

### 故障排查清单
- [ ] 检查 ArgoCD Application 状态
- [ ] 检查 Pod 状态和事件
- [ ] 检查应用日志
- [ ] 检查镜像路径是否正确
- [ ] 检查节点资源是否充足
- [ ] 检查网络连接（MQTT、Kafka）

---

## 🎓 经验总结

### 1. 小步快跑
- 每次只修改一个配置
- 验证后再进行下一步
- 保持 Git 提交的原子性

### 2. 充分利用日志
- Pod 日志是第一诊断工具
- 事件日志提供调度和资源信息
- ArgoCD 日志帮助理解同步过程

### 3. 容量规划很重要
- 提前规划节点资源
- 预留 20% 的容量用于扩容
- 监控资源使用趋势

### 4. 自动化是关键
- 使用 ArgoCD 自动同步
- 编写健康检查脚本
- 建立告警机制

### 5. 文档化所有问题
- 记录错误信息和解决方案
- 建立故障排查手册
- 分享经验给团队

---

## 🔗 相关项目文件

```
eks/
└── iot-platform/
    └── mqtt-kafka-bridge/
        ├── cmd/bridge/main.go          # 应用主程序
        ├── Dockerfile                   # Docker 构建配置
        ├── go.mod                       # Go 依赖管理
        ├── deployments/
        │   └── kubernetes/
        │       ├── deployment.yaml      # Kubernetes 部署配置
        │       ├── service.yaml         # Service 配置
        │       └── argocd-app.yaml      # ArgoCD Application 配置
        ├── deploy-simple.sh             # 一键部署脚本
        ├── DEPLOY_WINDOWS.md            # Windows 部署指南
        └── QUICK_DEPLOY.md              # 快速部署指南
```

---

## 📞 支持

如遇到问题，按以下顺序排查：
1. 查看本故障排查手册
2. 检查应用日志和事件
3. 运行健康检查脚本
4. 查阅 AWS 和 Kubernetes 官方文档

---

**文档版本**: 1.0  
**最后更新**: 2024-11-28  
**维护者**: DevOps Team
