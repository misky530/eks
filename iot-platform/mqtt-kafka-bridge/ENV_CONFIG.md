# 环境变量配置示例

## 基础配置 (必填)

### MQTT 配置
```bash
MQTT_BROKER=tcp://hats.hcs.cn:1883
MQTT_CLIENT_ID=mqtt-kafka-bridge
MQTT_TOPIC=mtic/msg/client/realtime/tenant123/#
```

### Kafka 配置
```bash
KAFKA_BROKERS=iot-cluster-kafka-bootstrap.kafka:9092
```

### 租户配置
```bash
TENANT_ID=tenant123
```

---

## 高级配置 (可选)

### 多租户模式
订阅所有租户的消息：
```bash
MQTT_TOPIC=mtic/msg/client/realtime/+/#
```

### 多 Kafka Broker
使用逗号分隔多个 Broker：
```bash
KAFKA_BROKERS=kafka-1.kafka:9092,kafka-2.kafka:9092,kafka-3.kafka:9092
```

### 日志级别
```bash
LOG_LEVEL=debug  # debug, info, warn, error
```

---

## Kubernetes ConfigMap 方式

创建 ConfigMap：
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mqtt-kafka-bridge-config
  namespace: iot-bridge
data:
  mqtt.broker: "tcp://hats.hcs.cn:1883"
  mqtt.topic: "mtic/msg/client/realtime/tenant123/#"
  kafka.brokers: "iot-cluster-kafka-bootstrap.kafka:9092"
  tenant.id: "tenant123"
```

在 Deployment 中引用：
```yaml
spec:
  containers:
  - name: bridge
    envFrom:
    - configMapRef:
        name: mqtt-kafka-bridge-config
```

---

## Kubernetes Secret 方式 (敏感信息)

如果 MQTT 需要认证：
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mqtt-credentials
  namespace: iot-bridge
type: Opaque
stringData:
  username: "your-username"
  password: "your-password"
```

在 Deployment 中引用：
```yaml
env:
- name: MQTT_USERNAME
  valueFrom:
    secretKeyRef:
      name: mqtt-credentials
      key: username
- name: MQTT_PASSWORD
  valueFrom:
    secretKeyRef:
      name: mqtt-credentials
      key: password
```

然后修改 Go 代码支持认证：
```go
opts := mqtt.NewClientOptions().
    AddBroker(config.MQTTBroker).
    SetUsername(getEnv("MQTT_USERNAME", "")).
    SetPassword(getEnv("MQTT_PASSWORD", ""))
```

---

## 本地开发配置

### .env 文件
```bash
# 创建 .env 文件 (不要提交到 Git)
cat > .env << EOF
MQTT_BROKER=tcp://hats.hcs.cn:1883
MQTT_CLIENT_ID=local-dev-bridge
MQTT_TOPIC=mtic/msg/client/realtime/tenant123/#
KAFKA_BROKERS=localhost:9092
TENANT_ID=tenant123
LOG_LEVEL=debug
EOF
```

### 使用 direnv (推荐)
```bash
# 安装 direnv
brew install direnv  # macOS
apt install direnv   # Ubuntu

# 创建 .envrc
cat > .envrc << EOF
export MQTT_BROKER=tcp://hats.hcs.cn:1883
export MQTT_CLIENT_ID=local-dev-bridge
export MQTT_TOPIC=mtic/msg/client/realtime/tenant123/#
export KAFKA_BROKERS=localhost:9092
export TENANT_ID=tenant123
export LOG_LEVEL=debug
EOF

# 允许加载
direnv allow
```

---

## Docker Compose 配置

```yaml
version: '3.8'

services:
  mqtt-kafka-bridge:
    build: .
    environment:
      MQTT_BROKER: tcp://hats.hcs.cn:1883
      MQTT_CLIENT_ID: docker-bridge
      MQTT_TOPIC: mtic/msg/client/realtime/tenant123/#
      KAFKA_BROKERS: kafka:9092
      TENANT_ID: tenant123
    depends_on:
      - kafka
    restart: unless-stopped

  kafka:
    image: confluentinc/cp-kafka:latest
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    depends_on:
      - zookeeper

  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
```

---

## 多环境配置

### 开发环境 (dev)
```yaml
env:
- name: MQTT_BROKER
  value: "tcp://dev-mqtt.example.com:1883"
- name: MQTT_TOPIC
  value: "mtic/msg/client/realtime/tenant-dev/#"
- name: LOG_LEVEL
  value: "debug"
```

### 测试环境 (staging)
```yaml
env:
- name: MQTT_BROKER
  value: "tcp://staging-mqtt.example.com:1883"
- name: MQTT_TOPIC
  value: "mtic/msg/client/realtime/tenant-staging/#"
- name: LOG_LEVEL
  value: "info"
```

### 生产环境 (prod)
```yaml
env:
- name: MQTT_BROKER
  value: "tcp://hats.hcs.cn:1883"
- name: MQTT_TOPIC
  value: "mtic/msg/client/realtime/tenant123/#"
- name: LOG_LEVEL
  value: "warn"
```

---

## 性能调优参数

### Kafka Writer 优化
```go
// 在代码中调整这些参数

BatchTimeout: 10 * time.Millisecond  // 批量延迟 (降低可提高吞吐)
BatchSize: 100                        // 批量大小
Compression: kafka.Snappy             // 压缩算法 (Snappy, Gzip, Lz4)
RequiredAcks: kafka.RequireOne        // ACK 级别 (All, One, None)
WriteTimeout: 10 * time.Second        // 写入超时
```

### MQTT 客户端优化
```go
// 在代码中调整这些参数

SetKeepAlive(60 * time.Second)              // 心跳间隔
SetPingTimeout(10 * time.Second)            // Ping 超时
SetMaxReconnectInterval(10 * time.Second)   // 最大重连间隔
SetAutoReconnect(true)                      // 自动重连
```

---

## 监控配置

### Prometheus 抓取配置
```yaml
metadata:
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
```

### 自定义指标 (未来扩展)
```go
// 添加到代码中
import "github.com/prometheus/client_golang/prometheus"

var (
    mqttMessagesReceived = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "mqtt_messages_received_total",
            Help: "Total number of MQTT messages received",
        },
    )
)
```

---

## 安全配置

### TLS 加密 (MQTT)
如果 MQTT Broker 支持 TLS：
```bash
MQTT_BROKER=ssl://hats.hcs.cn:8883
```

代码修改：
```go
opts := mqtt.NewClientOptions().
    AddBroker("ssl://hats.hcs.cn:8883").
    SetTLSConfig(&tls.Config{
        InsecureSkipVerify: false,  // 生产环境设为 false
    })
```

### Kafka SSL (未来支持)
```bash
KAFKA_BROKERS=kafka.example.com:9093
KAFKA_SSL_ENABLED=true
KAFKA_SSL_CA_CERT=/path/to/ca-cert
```

---

## 故障排查配置

### 启用详细日志
```yaml
env:
- name: LOG_LEVEL
  value: "debug"
```

### 增加超时时间 (网络不稳定)
```go
// 代码中调整
SetPingTimeout(30 * time.Second)    // MQTT Ping 超时
WriteTimeout: 30 * time.Second       // Kafka 写入超时
```

### 禁用自动重连 (调试)
```go
// 临时调试用
SetAutoReconnect(false)
```

---

## 配置验证清单

部署前检查：

- [ ] MQTT_BROKER 地址正确且可访问
- [ ] MQTT_TOPIC 格式符合预期
- [ ] KAFKA_BROKERS 集群内地址正确
- [ ] TENANT_ID 与实际租户匹配
- [ ] 资源 Limits 适合当前负载
- [ ] 健康检查参数合理
- [ ] 日志级别设置正确

---

## 常见问题

### Q: 如何修改订阅多个租户？
A: 修改 `MQTT_TOPIC=mtic/msg/client/realtime/+/#`

### Q: 如何增加 Kafka 写入吞吐？
A: 增加 `BatchSize` 和降低 `BatchTimeout`

### Q: 如何减少内存占用？
A: 减少 `BatchSize` 和 Kafka Writer 缓冲

### Q: 如何启用 MQTT 认证？
A: 添加 `MQTT_USERNAME` 和 `MQTT_PASSWORD` 环境变量，并修改代码

---

更多配置说明请参考 `README.md` 和 `ARCHITECTURE.md`。
