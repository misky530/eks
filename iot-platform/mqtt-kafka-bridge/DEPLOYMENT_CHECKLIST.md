# 部署检查清单

## ✅ 部署前检查

### 环境准备
- [ ] EKS 集群运行正常: `kubectl get nodes`
- [ ] kubectl 上下文正确: `kubectl config current-context`
- [ ] AWS CLI 已配置: `aws sts get-caller-identity`
- [ ] Docker 已安装: `docker --version`
- [ ] Git 仓库已准备: 将代码推送到你的 Git 仓库

### 资源检查
- [ ] 检查节点资源: `kubectl top nodes`
- [ ] 检查 Pod 数量: `kubectl get pods --all-namespaces --no-headers | wc -l`
- [ ] 确认 vCPU 配额: AWS 控制台 > Service Quotas

### 依赖服务
- [ ] Kafka 集群运行: `kubectl get kafka -n kafka iot-cluster`
- [ ] Kafka Bootstrap 服务可访问: `kubectl get svc -n kafka iot-cluster-kafka-bootstrap`
- [ ] ArgoCD 已安装: `kubectl get pods -n argocd`

---

## 📝 配置修改清单

### 1. deployment.yaml
```yaml
# 需要修改的地方:
image: <YOUR_AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/mqtt-kafka-bridge:latest

env:
- name: MQTT_BROKER
  value: "tcp://hats.hcs.cn:1883"  # ✅ 已确认
- name: MQTT_TOPIC
  value: "mtic/msg/client/realtime/tenant123/#"  # ✅ 已确认
- name: KAFKA_BROKERS
  value: "iot-cluster-kafka-bootstrap.kafka:9092"  # ✅ 已确认
```

### 2. argocd-app.yaml
```yaml
# 需要修改的地方:
spec:
  source:
    repoURL: <YOUR_GIT_REPO>  # ⚠️ 必须修改
    path: mqtt-kafka-bridge/deployments/kubernetes  # 根据实际路径调整
```

---

## 🚀 部署步骤

### 第 1 步: 解压项目
```bash
tar -xzf mqtt-kafka-bridge.tar.gz
cd mqtt-kafka-bridge
```
- [ ] 完成

### 第 2 步: 运行部署脚本
```bash
./deploy.sh
```
输入信息:
- AWS Account ID: _____________
- AWS Region: us-east-1

- [ ] ECR 仓库创建成功
- [ ] Docker 镜像构建成功
- [ ] 镜像推送到 ECR 成功
- [ ] deployment.yaml 已更新

### 第 3 步: 修改 ArgoCD 配置
```bash
vim deployments/kubernetes/argocd-app.yaml
# 修改 spec.source.repoURL
```
- [ ] 完成

### 第 4 步: 提交到 Git
```bash
git init  # 如果是新项目
git add .
git commit -m "Add MQTT-Kafka Bridge"
git remote add origin <YOUR_GIT_REPO>
git push -u origin main
```
- [ ] 完成

### 第 5 步: 部署到 EKS
```bash
kubectl apply -f deployments/kubernetes/argocd-app.yaml
```
- [ ] ArgoCD Application 创建成功

---

## 🔍 验证检查

### ArgoCD 同步
```bash
kubectl get application -n argocd mqtt-kafka-bridge
```
期望状态:
```
NAME                 SYNC STATUS   HEALTH STATUS
mqtt-kafka-bridge    Synced        Healthy
```
- [ ] Sync Status: Synced
- [ ] Health Status: Healthy

### Namespace 创建
```bash
kubectl get namespace iot-bridge
```
- [ ] Namespace 存在

### Deployment 状态
```bash
kubectl get deployment -n iot-bridge
```
期望输出:
```
NAME                 READY   UP-TO-DATE   AVAILABLE
mqtt-kafka-bridge    1/1     1            1
```
- [ ] READY: 1/1
- [ ] AVAILABLE: 1

### Pod 状态
```bash
kubectl get pods -n iot-bridge
```
期望输出:
```
NAME                                 READY   STATUS    RESTARTS
mqtt-kafka-bridge-xxx                1/1     Running   0
```
- [ ] STATUS: Running
- [ ] READY: 1/1

### 日志检查
```bash
kubectl logs -n iot-bridge -l app=mqtt-kafka-bridge --tail=50
```
期望日志包含:
- [ ] "Starting MQTT-Kafka Bridge"
- [ ] "Connected to MQTT broker"
- [ ] "Successfully subscribed to MQTT topic"

---

## 🧪 功能测试

### 测试 1: MQTT 连接
```bash
kubectl logs -n iot-bridge -l app=mqtt-kafka-bridge | grep "Connected to MQTT"
```
- [ ] 看到连接成功日志

### 测试 2: 订阅确认
```bash
kubectl logs -n iot-bridge -l app=mqtt-kafka-bridge | grep "Successfully subscribed"
```
- [ ] 看到订阅成功日志

### 测试 3: Kafka 消费
```bash
./scripts/test-consumer.sh tenant123 project001
# 等待外部 MQTT 发送消息到对应的 topic
```
- [ ] 能够看到转发的消息

### 测试 4: 资源使用
```bash
kubectl top pod -n iot-bridge
```
期望:
- CPU: < 100m
- Memory: < 100Mi
- [ ] 资源使用在预期范围内

---

## 📊 性能验证

### 消息转发延迟
观察日志中的 "Message forwarded" 条目:
- [ ] 延迟 < 100ms (正常网络条件)

### Pod 稳定性
```bash
kubectl get pods -n iot-bridge -w
# 观察 5 分钟
```
- [ ] 无重启 (RESTARTS=0)
- [ ] 状态始终 Running

---

## 🐛 故障排查清单

### Pod 无法启动

#### 检查项 1: 描述 Pod
```bash
kubectl describe pod -n iot-bridge -l app=mqtt-kafka-bridge
```
常见问题:
- [ ] ImagePullBackOff → 检查 ECR 权限
- [ ] CrashLoopBackOff → 查看日志
- [ ] Pending → 检查资源配额

#### 检查项 2: 节点资源
```bash
kubectl describe node | grep -A 5 "Allocated resources"
```
- [ ] CPU 可用
- [ ] Memory 可用

### MQTT 连接失败

#### 检查项 1: 网络连通性
```bash
kubectl run test-mqtt --rm -it --restart=Never --image=busybox -- \
  ping -c 3 hats.hcs.cn
```
- [ ] 能 ping 通

#### 检查项 2: 端口访问
```bash
kubectl run test-mqtt --rm -it --restart=Never --image=nicolaka/netshoot -- \
  nc -zv hats.hcs.cn 1883
```
- [ ] 端口可达

### Kafka 写入失败

#### 检查项 1: Kafka 集群状态
```bash
kubectl get kafka -n kafka iot-cluster -o yaml
```
- [ ] status.conditions[?(@.type=='Ready')].status == "True"

#### 检查项 2: Kafka 服务
```bash
kubectl get svc -n kafka iot-cluster-kafka-bootstrap
```
- [ ] Service 存在且 ClusterIP 正常

#### 检查项 3: Kafka 连通性
```bash
kubectl run kafka-test --rm -it --restart=Never --image=confluentinc/cp-kafka:latest -- \
  kafka-broker-api-versions --bootstrap-server iot-cluster-kafka-bootstrap.kafka:9092
```
- [ ] 能连接到 Kafka

---

## 📈 监控设置 (可选)

### Prometheus 抓取
```bash
# 启用监控 (未来)
kubectl edit deployment -n iot-bridge mqtt-kafka-bridge

# 修改 annotations:
prometheus.io/scrape: "true"
prometheus.io/port: "8080"
prometheus.io/path: "/metrics"
```
- [ ] 监控已启用 (可选)

### Grafana Dashboard
- [ ] 导入自定义 Dashboard (可选)

---

## ✅ 最终验证

在所有检查完成后:

```bash
# 综合状态检查
make status
```

期望输出:
```
=== ArgoCD Application ===
NAME                 SYNC STATUS   HEALTH STATUS
mqtt-kafka-bridge    Synced        Healthy

=== Pods ===
NAME                                 READY   STATUS    RESTARTS   AGE
mqtt-kafka-bridge-xxx                1/1     Running   0          5m

=== Resource Usage ===
NAME                                 CPU    MEMORY
mqtt-kafka-bridge-xxx                45m    58Mi
```

- [ ] 所有组件健康
- [ ] 消息正常转发
- [ ] 资源使用正常
- [ ] 无错误日志

---

## 🎉 部署完成

恭喜！MQTT-Kafka Bridge 已成功部署。

### 下一步建议:
1. 监控运行 24 小时，确保稳定性
2. 根据实际流量调整资源配置
3. 配置告警规则
4. 定期检查日志

### 文档参考:
- 快速开始: `QUICKSTART.md`
- 架构设计: `ARCHITECTURE.md`
- 完整文档: `README.md`

---

**日期**: ___________  
**部署人**: ___________  
**环境**: EKS Cluster (iot-platform)  
**版本**: v1.0.0
