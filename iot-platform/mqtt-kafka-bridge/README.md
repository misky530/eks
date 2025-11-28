# MQTT-Kafka Bridge

轻量级 MQTT 到 Kafka 消息桥接服务，用于 IoT 数据处理平台。

## 功能特性

- ✅ 订阅外部 MQTT Broker (通配符支持)
- ✅ 动态 Kafka Topic 映射 (基于 tenantId/projectId)
- ✅ 原样转发 JSON 消息
- ✅ 自动重连和容错
- ✅ 优雅关闭 (Kubernetes 友好)
- ✅ 极低资源占用 (<20MB 镜像, 64Mi 内存)

## 架构

```
MQTT Broker (hats.hcs.cn:1883)
    |
    | Subscribe: mtic/msg/client/realtime/tenant123/#
    |
    v
mqtt-kafka-bridge (Pod)
    |
    | Dynamic Topic: tenant123.project456
    |
    v
Kafka Cluster (iot-cluster)
```

## 部署步骤

### 1. 构建镜像

```bash
# 登录 ECR
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com

# 创建 ECR 仓库
aws ecr create-repository --repository-name mqtt-kafka-bridge --region us-east-1

# 构建镜像
docker build -t mqtt-kafka-bridge:latest .

# 标记镜像
docker tag mqtt-kafka-bridge:latest \
  <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/mqtt-kafka-bridge:latest

# 推送镜像
docker push <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/mqtt-kafka-bridge:latest
```

### 2. 更新 Kubernetes 配置

编辑 `deployments/kubernetes/deployment.yaml`：

```yaml
image: <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/mqtt-kafka-bridge:latest
```

### 3. 提交到 Git

```bash
git add .
git commit -m "Add MQTT-Kafka Bridge application"
git push origin main
```

### 4. 部署 ArgoCD Application

```bash
# 更新 argocd-app.yaml 中的 repoURL
kubectl apply -f deployments/kubernetes/argocd-app.yaml

# 查看同步状态
kubectl get application -n argocd mqtt-kafka-bridge
```

### 5. 验证部署

```bash
# 检查 Pod 状态
kubectl get pods -n iot-bridge

# 查看日志
kubectl logs -f -n iot-bridge -l app=mqtt-kafka-bridge

# 检查资源使用
kubectl top pod -n iot-bridge
```

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `MQTT_BROKER` | `tcp://hats.hcs.cn:1883` | MQTT Broker 地址 |
| `MQTT_CLIENT_ID` | `mqtt-kafka-bridge` | MQTT 客户端 ID |
| `MQTT_TOPIC` | `mtic/msg/client/realtime/tenant123/#` | 订阅主题 (支持通配符) |
| `KAFKA_BROKERS` | `iot-cluster-kafka-bootstrap.kafka:9092` | Kafka 集群地址 |
| `TENANT_ID` | `tenant123` | 租户 ID |

### Topic 映射规则

MQTT Topic: `mtic/msg/client/realtime/tenant123/project456`  
→ Kafka Topic: `tenant123.project456`

## 资源占用

- **镜像大小**: ~18MB
- **内存**: 
  - Request: 64Mi
  - Limit: 128Mi
- **CPU**:
  - Request: 50m (0.05 核)
  - Limit: 200m (0.2 核)

## 监控

查看实时日志：
```bash
kubectl logs -f -n iot-bridge deployment/mqtt-kafka-bridge
```

预期日志输出：
```
{"level":"info","msg":"Starting MQTT-Kafka Bridge","mqtt_broker":"tcp://hats.hcs.cn:1883"}
{"level":"info","msg":"Connected to MQTT broker"}
{"level":"info","msg":"MQTT connected, subscribing to topic","topic":"mtic/msg/client/realtime/tenant123/#"}
{"level":"info","msg":"Successfully subscribed to MQTT topic"}
{"level":"debug","msg":"Message forwarded","mqtt_topic":"mtic/msg/client/realtime/tenant123/project456","kafka_topic":"tenant123.project456","size":1234}
```

## 故障排查

### Pod 无法启动

```bash
# 查看 Pod 事件
kubectl describe pod -n iot-bridge -l app=mqtt-kafka-bridge

# 检查镜像拉取权限
kubectl get secret -n iot-bridge
```

### 连接 MQTT 失败

```bash
# 从 Pod 内测试连接
kubectl exec -it -n iot-bridge deployment/mqtt-kafka-bridge -- sh
# (如果 shell 不可用，检查日志中的连接错误)
```

### Kafka 写入失败

```bash
# 检查 Kafka 集群状态
kubectl get kafka -n kafka iot-cluster

# 测试 Kafka 连通性
kubectl run kafka-test --rm -it --restart=Never --image=confluentinc/cp-kafka:latest \
  -- kafka-topics --list --bootstrap-server iot-cluster-kafka-bootstrap.kafka:9092
```

## 扩展

### 多租户订阅

修改 `MQTT_TOPIC` 为 `mtic/msg/client/realtime/+/#` 订阅所有租户。

### 增加副本

```yaml
spec:
  replicas: 2  # 注意 MQTT 客户端 ID 冲突问题
```

### 启用监控

在 deployment.yaml 中启用 Prometheus 抓取：
```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
  prometheus.io/path: "/metrics"
```

## 本地开发

```bash
# 安装依赖
go mod download

# 运行
export MQTT_BROKER=tcp://hats.hcs.cn:1883
export KAFKA_BROKERS=localhost:9092
go run cmd/bridge/main.go
```

## License

MIT
