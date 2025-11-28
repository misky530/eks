```mermaid
graph TB
    subgraph "外部环境"
        MQTT[MQTT Broker<br/>hats.hcs.cn:1883]
        Device1[IoT 设备 1<br/>tenant123/project001]
        Device2[IoT 设备 2<br/>tenant123/project002]
        Device3[IoT 设备 3<br/>tenant123/project003]
    end

    subgraph "EKS Cluster"
        subgraph "Namespace: iot-bridge"
            Bridge[mqtt-kafka-bridge Pod<br/>CPU: 50m<br/>Memory: 64Mi]
        end
        
        subgraph "Namespace: kafka"
            Kafka[Kafka Cluster<br/>iot-cluster]
            Topic1[Topic: tenant123.project001]
            Topic2[Topic: tenant123.project002]
            Topic3[Topic: tenant123.project003]
        end
        
        subgraph "Namespace: argocd"
            ArgoCD[ArgoCD<br/>GitOps 控制器]
        end
    end
    
    subgraph "Git Repository"
        Git[deployment.yaml<br/>argocd-app.yaml]
    end

    Device1 -->|Publish| MQTT
    Device2 -->|Publish| MQTT
    Device3 -->|Publish| MQTT
    
    MQTT -->|Subscribe<br/>tenant123/#| Bridge
    
    Bridge -->|Produce| Topic1
    Bridge -->|Produce| Topic2
    Bridge -->|Produce| Topic3
    
    Topic1 --> Kafka
    Topic2 --> Kafka
    Topic3 --> Kafka
    
    Git -->|Sync| ArgoCD
    ArgoCD -->|Deploy| Bridge

    style Bridge fill:#4CAF50,stroke:#333,stroke-width:3px,color:#fff
    style Kafka fill:#FF6B6B,stroke:#333,stroke-width:2px,color:#fff
    style MQTT fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style ArgoCD fill:#FFC107,stroke:#333,stroke-width:2px,color:#000
```

## 消息流程详解

### 1. 发布阶段 (IoT 设备 → MQTT Broker)
```
IoT Device → MQTT Broker
Topic: mtic/msg/client/realtime/tenant123/project001
Payload: {"deviceId": "sensor-001", "temp": 25.3, ...}
```

### 2. 订阅阶段 (Bridge → MQTT Broker)
```
Bridge 订阅: mtic/msg/client/realtime/tenant123/#
↓
接收所有 tenant123 下的消息:
  - tenant123/project001
  - tenant123/project002
  - tenant123/project003
```

### 3. 转换阶段 (Bridge 内部处理)
```go
// 解析 MQTT Topic
input:  "mtic/msg/client/realtime/tenant123/project001"
output: {
    tenantId: "tenant123",
    projectId: "project001",
    kafkaTopic: "tenant123.project001"
}
```

### 4. 写入阶段 (Bridge → Kafka)
```
Kafka Message {
    Topic: "tenant123.project001"
    Key: "project001"
    Value: {"deviceId": "sensor-001", "temp": 25.3, ...}
}
```

### 5. GitOps 部署流程
```
开发者 Push → Git Repository
    ↓
ArgoCD 检测变更
    ↓
自动同步到集群
    ↓
Bridge Pod 更新
```

## 时序图

```mermaid
sequenceDiagram
    participant Device as IoT 设备
    participant MQTT as MQTT Broker
    participant Bridge as mqtt-kafka-bridge
    participant Kafka as Kafka Cluster
    
    Device->>MQTT: Publish 消息<br/>Topic: tenant123/project001
    MQTT->>Bridge: Forward 消息 (订阅)
    Bridge->>Bridge: 解析 tenantId/projectId
    Bridge->>Bridge: 构造 Kafka Topic<br/>tenant123.project001
    Bridge->>Kafka: Produce 消息
    Kafka-->>Bridge: ACK
    Bridge->>Bridge: 记录日志<br/>Message forwarded
    
    Note over Device,Kafka: 端到端延迟: < 50ms
```

## 部署流程图

```mermaid
flowchart LR
    A[开始] --> B{环境检查}
    B -->|✅ 通过| C[运行 deploy.sh]
    B -->|❌ 失败| B1[修复环境]
    B1 --> B
    
    C --> D[构建 Docker 镜像]
    D --> E[推送到 ECR]
    E --> F[更新 K8s 配置]
    F --> G[提交到 Git]
    G --> H[应用 ArgoCD App]
    H --> I[ArgoCD 同步]
    I --> J{Pod 健康检查}
    J -->|✅ Running| K[部署成功]
    J -->|❌ 失败| L[查看日志]
    L --> M[故障排查]
    M --> H
    
    K --> N[监控消息流]
    
    style K fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style L fill:#FF6B6B,stroke:#333,stroke-width:2px,color:#fff
    style N fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
```

## 资源占用对比

```mermaid
pie title "单节点 t3.small 资源分配 (2 vCPU / 2GB RAM)"
    "System Reserved" : 15
    "Existing Pods" : 70
    "mqtt-kafka-bridge" : 3
    "Available" : 12
```

## 消息路由逻辑

```mermaid
graph LR
    A[MQTT 消息] --> B{解析 Topic}
    B --> C{parts.length >= 6?}
    C -->|No| D[丢弃 - 无效格式]
    C -->|Yes| E[提取 tenantId]
    E --> F[提取 projectId]
    F --> G[构造 Kafka Topic<br/>tenantId.projectId]
    G --> H{Kafka 写入}
    H -->|Success| I[记录 Debug 日志]
    H -->|Failure| J[记录 Error 日志]
    
    style I fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style J fill:#FF6B6B,stroke:#333,stroke-width:2px,color:#fff
    style D fill:#FFC107,stroke:#333,stroke-width:2px,color:#000
```

## 容错机制

```mermaid
stateDiagram-v2
    [*] --> Connecting: 启动
    Connecting --> Connected: MQTT 连接成功
    Connected --> Subscribed: 订阅成功
    Subscribed --> Forwarding: 接收消息
    Forwarding --> Forwarding: 持续转发
    
    Connected --> Disconnected: 连接丢失
    Subscribed --> Disconnected: 连接丢失
    Forwarding --> Disconnected: 连接丢失
    
    Disconnected --> Reconnecting: 自动重连
    Reconnecting --> Connected: 重连成功
    Reconnecting --> Reconnecting: 重试中 (最多 10s)
    
    Forwarding --> KafkaError: Kafka 写入失败
    KafkaError --> Forwarding: 重试成功
    KafkaError --> Forwarding: 记录错误继续
    
    Subscribed --> [*]: SIGTERM 优雅关闭
    Forwarding --> [*]: SIGTERM 优雅关闭
```
