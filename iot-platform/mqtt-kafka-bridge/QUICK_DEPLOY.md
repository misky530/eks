# ⚡ 超快速部署 (3 分钟)

## 前提条件

✅ 你已经配置好：
- Docker Desktop
- AWS CLI (已登录)
- kubectl (已连接 EKS)
- Git

## 🚀 部署步骤

### 1. 进入项目目录
```bash
cd /d/code2025/eks/iot-platform/mqtt-kafka-bridge
```

### 2. 运行一键部署脚本
```bash
chmod +x deploy-simple.sh
./deploy-simple.sh
```

**等待 2-3 分钟，脚本会自动：**
- ✓ 获取你的 AWS Account ID
- ✓ 创建 ECR 仓库
- ✓ 构建 Docker 镜像
- ✓ 推送到 ECR
- ✓ 更新配置文件

### 3. 设置 Git 仓库地址

编辑 `deployments/kubernetes/argocd-app.yaml`：

```bash
vim deployments/kubernetes/argocd-app.yaml
```

修改这一行：
```yaml
spec:
  source:
    repoURL: https://github.com/你的用户名/iot-platform  # 改成你的仓库
```

### 4. 提交到 Git

```bash
git add .
git commit -m "Add MQTT-Kafka Bridge"
git push origin main
```

### 5. 部署到 EKS

```bash
kubectl apply -f deployments/kubernetes/argocd-app.yaml
```

### 6. 验证部署

```bash
# 查看 Pod 状态
kubectl get pods -n iot-bridge

# 查看日志 (应该看到 "Connected to MQTT broker")
kubectl logs -f -n iot-bridge -l app=mqtt-kafka-bridge
```

## ✅ 完成！

如果看到类似这样的日志，说明成功了：
```json
{"level":"info","msg":"Starting MQTT-Kafka Bridge"}
{"level":"info","msg":"Connected to MQTT broker"}
{"level":"info","msg":"Successfully subscribed to MQTT topic"}
```

---

## 🧪 测试消息流

```bash
# 运行 Kafka 消费者，等待消息
./scripts/test-consumer.sh tenant123 project001
```

当外部 MQTT 设备发送消息到 `mtic/msg/client/realtime/tenant123/project001` 时，你会在消费者中看到消息。

---

## 🛠️ 常用命令

```bash
# 查看所有状态
kubectl get all -n iot-bridge

# 查看日志
kubectl logs -f -n iot-bridge -l app=mqtt-kafka-bridge

# 查看资源使用
kubectl top pod -n iot-bridge

# 重启 Pod
kubectl rollout restart deployment/mqtt-kafka-bridge -n iot-bridge

# 删除部署
kubectl delete -f deployments/kubernetes/argocd-app.yaml
```

---

## ❓ 遇到问题？

### Pod 一直 Pending
```bash
# 检查节点资源
kubectl describe nodes

# 可能是 CPU/Memory 不足
```

### ImagePullBackOff
```bash
# 检查镜像是否推送成功
aws ecr describe-images --repository-name mqtt-kafka-bridge
```

### CrashLoopBackOff
```bash
# 查看详细日志
kubectl logs -n iot-bridge -l app=mqtt-kafka-bridge --tail=100
kubectl describe pod -n iot-bridge -l app=mqtt-kafka-bridge
```

---

## 📚 更多文档

- 详细部署: `QUICKSTART.md`
- Windows 指南: `DEPLOY_WINDOWS.md`
- 架构设计: `ARCHITECTURE.md`
- 配置说明: `ENV_CONFIG.md`
- 完整文档: `README.md`

---

**就这么简单！🎉**
