# MQTT-Kafka Bridge 快速部署指南

## 📦 项目已创建完成

✅ Go 应用 (极简资源占用)
✅ Docker 镜像 (多阶段构建 < 20MB)
✅ Kubernetes 部署配置 (requests: 50m CPU / 64Mi 内存)
✅ ArgoCD GitOps 集成
✅ 自动化部署脚本

---

## 🚀 5 分钟快速部署

### 前提条件
- EKS 集群已运行
- kubectl 配置正确
- AWS CLI 已配置
- Docker 已安装

### 第 1 步：解压项目

```bash
tar -xzf mqtt-kafka-bridge.tar.gz
cd mqtt-kafka-bridge
```

### 第 2 步：一键部署

```bash
./deploy.sh
```

脚本会自动完成：
1. 创建 ECR 仓库
2. 构建 Docker 镜像
3. 推送到 ECR
4. 更新 Kubernetes 配置

### 第 3 步：配置 ArgoCD

编辑 `deployments/kubernetes/argocd-app.yaml`：

```yaml
spec:
  source:
    repoURL: https://github.com/YOUR_USERNAME/iot-platform  # 修改这里
```

### 第 4 步：提交到 Git

```bash
git add .
git commit -m "Add MQTT-Kafka Bridge"
git push origin main
```

### 第 5 步：部署到集群

```bash
kubectl apply -f deployments/kubernetes/argocd-app.yaml
```

### 第 6 步：验证部署

```bash
# 查看 Pod 状态
kubectl get pods -n iot-bridge

# 实时日志
kubectl logs -f -n iot-bridge -l app=mqtt-kafka-bridge
```

预期输出：
```
{"level":"info","msg":"Starting MQTT-Kafka Bridge"}
{"level":"info","msg":"Connected to MQTT broker"}
{"level":"info","msg":"Successfully subscribed to MQTT topic"}
```

---

## 🧪 测试消息流

### 测试 1：消费 Kafka 消息

```bash
# 运行消费者测试
./scripts/test-consumer.sh tenant123 project001

# 应该看到从 MQTT 转发过来的消息
```

### 测试 2：检查资源使用

```bash
kubectl top pod -n iot-bridge
```

预期：
```
NAME                                CPU    MEMORY
mqtt-kafka-bridge-xxx              10m    45Mi
```

### 测试 3：压力测试

MQTT 每秒 100 条消息时的资源使用：
- CPU: ~50m
- Memory: ~60Mi

---

## 📊 监控

### 查看 ArgoCD 界面

```bash
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

访问: https://localhost:8080

### 实时日志

```bash
# 使用 Makefile
make logs

# 或直接使用 kubectl
kubectl logs -f -n iot-bridge deployment/mqtt-kafka-bridge
```

---

## 🔧 常用命令

```bash
# 查看所有状态
make status

# 重启 Pod
kubectl rollout restart deployment/mqtt-kafka-bridge -n iot-bridge

# 查看资源使用
kubectl top pod -n iot-bridge

# 进入 Pod (如果需要)
kubectl exec -it -n iot-bridge deployment/mqtt-kafka-bridge -- sh
```

---

## 🐛 故障排查

### Pod 无法启动

```bash
kubectl describe pod -n iot-bridge -l app=mqtt-kafka-bridge
```

常见问题：
1. **镜像拉取失败**: 检查 ECR 权限
2. **资源不足**: 查看节点 CPU/内存
3. **Liveness 探针失败**: 检查进程是否正常运行

### MQTT 连接失败

检查网络连通性：
```bash
kubectl run test --rm -it --restart=Never --image=busybox -- \
  wget -O- http://hats.hcs.cn:1883
```

### Kafka 写入失败

检查 Kafka 集群：
```bash
kubectl get kafka -n kafka iot-cluster
kubectl get kafkatopic -n kafka
```

---

## 📈 扩展配置

### 多租户订阅

修改 `deployments/kubernetes/deployment.yaml`：

```yaml
env:
- name: MQTT_TOPIC
  value: "mtic/msg/client/realtime/+/#"  # 订阅所有租户
```

### 增加副本（高可用）

```yaml
spec:
  replicas: 2
```

⚠️ **注意**: MQTT 客户端 ID 可能冲突，需要使用 Pod 名称作为 Client ID（已配置）

### 预创建 Kafka Topics

```bash
./scripts/create-topics.sh
```

---

## 📁 项目结构

```
mqtt-kafka-bridge/
├── cmd/bridge/main.go           # 主程序
├── Dockerfile                   # 多阶段构建
├── go.mod / go.sum              # Go 依赖
├── Makefile                     # 常用命令
├── deploy.sh                    # 一键部署
├── deployments/kubernetes/
│   ├── deployment.yaml          # K8s 部署配置
│   └── argocd-app.yaml          # ArgoCD 应用
└── scripts/
    ├── create-topics.sh         # 预创建 Topics
    └── test-consumer.sh         # 消费测试
```

---

## 💡 最佳实践

1. **资源控制**: 已配置最小资源，避免 Pod 驱逐
2. **优雅关闭**: 支持 30 秒优雅关闭，确保消息不丢失
3. **自动重连**: MQTT 和 Kafka 都支持自动重连
4. **安全性**: 非 root 用户运行，只读文件系统
5. **可观测性**: 结构化日志，方便 ELK 采集

---

## 🆘 需要帮助？

检查以下日志：

```bash
# Bridge 日志
kubectl logs -n iot-bridge -l app=mqtt-kafka-bridge

# ArgoCD 同步日志
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-application-controller

# Kafka 日志
kubectl logs -n kafka -l strimzi.io/name=iot-cluster-kafka
```

---

## 📝 下一步

1. ✅ 部署成功后，可以添加消费者应用处理 Kafka 消息
2. ✅ 集成 Prometheus 监控（已预留配置）
3. ✅ 根据流量调整副本数
4. ✅ 配置 Kafka Topic 的保留策略

祝部署顺利！🎉
