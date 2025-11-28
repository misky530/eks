package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Config struct {
	MQTTBroker   string
	MQTTClientID string
	MQTTTopic    string
	KafkaBrokers []string
	KafkaTopic   string // 新增：统一的 Kafka Topic
}

// 消息结构（用于 JSON 序列化）
type Message struct {
	TenantID  string    `json:"tenant_id"`
	ProjectID string    `json:"project_id"`
	Topic     string    `json:"mqtt_topic"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

type Bridge struct {
	config       Config
	mqttClient   mqtt.Client
	kafkaWriter  *kafka.Writer
	logger       *zap.Logger
	messageQueue chan Message // 新增：消息队列
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

func main() {
	// 初始化日志
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 加载配置
	config := Config{
		MQTTBroker:   getEnv("MQTT_BROKER", "tcp://hats.hcs.cn:1883"),
		MQTTClientID: getEnv("MQTT_CLIENT_ID", "mqtt-kafka-bridge"),
		MQTTTopic:    getEnv("MQTT_TOPIC", "mtic/msg/client/realtime/#"), // 改为通配符
		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "iot-cluster-kafka-bootstrap.kafka:9092"), ","),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "iot-messages"), // 统一 topic
	}

	logger.Info("Starting MQTT-Kafka Bridge",
		zap.String("mqtt_broker", config.MQTTBroker),
		zap.String("mqtt_topic", config.MQTTTopic),
		zap.Strings("kafka_brokers", config.KafkaBrokers),
		zap.String("kafka_topic", config.KafkaTopic),
	)

	// 创建 Bridge
	bridge := NewBridge(config, logger)

	// 启动
	if err := bridge.Start(); err != nil {
		logger.Fatal("Failed to start bridge", zap.Error(err))
	}

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down gracefully...")
	bridge.Stop()
}

func NewBridge(config Config, logger *zap.Logger) *Bridge {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建 Kafka Writer - 配置为不丢数据
	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP(config.KafkaBrokers...),
		Topic:        config.KafkaTopic, // 使用统一 topic
		Balancer:     &kafka.Hash{},     // Hash 分区，相同 tenantId 到同一分区
		BatchTimeout: 100 * time.Millisecond,
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireAll, // 🔥 改为等待所有副本确认
		Compression:  kafka.Snappy,
		MaxAttempts:  10, // 🔥 增加重试次数
		Async:        false, // 🔥 同步写入
	}

	return &Bridge{
		config:       config,
		kafkaWriter:  kafkaWriter,
		logger:       logger,
		messageQueue: make(chan Message, 10000), // 🔥 新增消息队列
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (b *Bridge) Start() error {
	// 🔥 启动消息处理 goroutine
	b.wg.Add(1)
	go b.messageProcessor()

	// 配置 MQTT 客户端
	opts := mqtt.NewClientOptions().
		AddBroker(b.config.MQTTBroker).
		SetClientID(b.config.MQTTClientID).
		SetKeepAlive(60 * time.Second).
		SetPingTimeout(10 * time.Second).
		SetCleanSession(false). // 🔥 持久会话
		SetAutoReconnect(true).
		SetMaxReconnectInterval(10 * time.Second).
		SetConnectionLostHandler(b.onConnectionLost).
		SetOnConnectHandler(b.onConnect)

	// 创建客户端
	b.mqttClient = mqtt.NewClient(opts)

	// 连接
	if token := b.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	b.logger.Info("Connected to MQTT broker")
	return nil
}

func (b *Bridge) Stop() {
	b.logger.Info("Stopping bridge...")

	// 🔥 取消 context，停止消息处理
	b.cancel()

	// 断开 MQTT
	if b.mqttClient != nil && b.mqttClient.IsConnected() {
		b.mqttClient.Disconnect(1000) // 等待 1 秒
	}

	// 🔥 等待消息处理完成
	b.wg.Wait()

	// 关闭 Kafka Writer
	if b.kafkaWriter != nil {
		b.kafkaWriter.Close()
	}

	b.logger.Info("Bridge stopped gracefully")
}

func (b *Bridge) onConnect(client mqtt.Client) {
	b.logger.Info("MQTT connected, subscribing to topic", zap.String("topic", b.config.MQTTTopic))

	// 🔥 QoS 2 订阅（确保消息不丢失）
	token := client.Subscribe(b.config.MQTTTopic, 2, b.onMessage)
	if token.Wait() && token.Error() != nil {
		b.logger.Error("Failed to subscribe", zap.Error(token.Error()))
		return
	}

	b.logger.Info("Successfully subscribed to MQTT topic")
}

func (b *Bridge) onConnectionLost(client mqtt.Client, err error) {
	b.logger.Warn("MQTT connection lost", zap.Error(err))
}

func (b *Bridge) onMessage(client mqtt.Client, msg mqtt.Message) {
	// 提取 tenantId 和 projectId
	// Topic 格式: mtic/msg/client/realtime/tenant123/project456
	parts := strings.Split(msg.Topic(), "/")
	if len(parts) < 6 {
		b.logger.Warn("Invalid topic format", zap.String("topic", msg.Topic()))
		return
	}

	tenantID := parts[4]
	projectID := parts[5]

	// 🔥 构造消息对象
	message := Message{
		TenantID:  tenantID,
		ProjectID: projectID,
		Topic:     msg.Topic(),
		Payload:   string(msg.Payload()),
		Timestamp: time.Now(),
	}

	// 🔥 非阻塞发送到队列
	select {
	case b.messageQueue <- message:
		// 成功入队
	default:
		// 队列满了，记录严重错误
		b.logger.Error("Message queue full, dropping message",
			zap.String("tenant_id", tenantID),
			zap.String("project_id", projectID))
	}
}

// 🔥 新增：消息处理器（批量写入 Kafka）
func (b *Bridge) messageProcessor() {
	defer b.wg.Done()

	batch := make([]kafka.Message, 0, 100)
	ticker := time.NewTicker(1 * time.Second) // 每秒刷新一次
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		err := b.kafkaWriter.WriteMessages(b.ctx, batch...)
		if err != nil {
			b.logger.Error("Failed to write batch to Kafka",
				zap.Error(err),
				zap.Int("batch_size", len(batch)))
		} else {
			b.logger.Info("Batch written to Kafka",
				zap.Int("count", len(batch)))
		}

		batch = batch[:0] // 清空但保留容量
	}

	for {
		select {
		case msg := <-b.messageQueue:
			// 序列化消息为 JSON
			payload, err := json.Marshal(msg)
			if err != nil {
				b.logger.Error("Failed to marshal message", zap.Error(err))
				continue
			}

			// 添加到批次
			batch = append(batch, kafka.Message{
				Key:   []byte(msg.TenantID), // 使用 tenantId 作为 key
				Value: payload,
				Time:  msg.Timestamp,
			})

			// 批次满了立即刷新
			if len(batch) >= 100 {
				flush()
			}

		case <-ticker.C:
			// 定时刷新
			flush()

		case <-b.ctx.Done():
			// 优雅关闭：刷新剩余消息
			b.logger.Info("Flushing remaining messages", zap.Int("count", len(batch)))
			flush()
			return
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
