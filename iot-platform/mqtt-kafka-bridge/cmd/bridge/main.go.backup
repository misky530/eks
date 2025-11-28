package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Config struct {
	MQTTBroker    string
	MQTTClientID  string
	MQTTTopic     string
	KafkaBrokers  []string
	TenantID      string
}

type Bridge struct {
	config       Config
	mqttClient   mqtt.Client
	kafkaWriter  *kafka.Writer
	logger       *zap.Logger
}

func main() {
	// 初始化日志
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 加载配置
	config := Config{
		MQTTBroker:   getEnv("MQTT_BROKER", "tcp://hats.hcs.cn:1883"),
		MQTTClientID: getEnv("MQTT_CLIENT_ID", "mqtt-kafka-bridge"),
		MQTTTopic:    getEnv("MQTT_TOPIC", "mtic/msg/client/realtime/tenant123/#"),
		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "iot-cluster-kafka-bootstrap.kafka:9092"), ","),
		TenantID:     getEnv("TENANT_ID", "tenant123"),
	}

	logger.Info("Starting MQTT-Kafka Bridge",
		zap.String("mqtt_broker", config.MQTTBroker),
		zap.String("mqtt_topic", config.MQTTTopic),
		zap.Strings("kafka_brokers", config.KafkaBrokers),
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
	// 创建 Kafka Writer
	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP(config.KafkaBrokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireOne,
		Compression:  kafka.Snappy,
	}

	return &Bridge{
		config:      config,
		kafkaWriter: kafkaWriter,
		logger:      logger,
	}
}

func (b *Bridge) Start() error {
	// 配置 MQTT 客户端
	opts := mqtt.NewClientOptions().
		AddBroker(b.config.MQTTBroker).
		SetClientID(b.config.MQTTClientID).
		SetKeepAlive(60 * time.Second).
		SetPingTimeout(10 * time.Second).
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

	// 断开 MQTT
	if b.mqttClient != nil && b.mqttClient.IsConnected() {
		b.mqttClient.Disconnect(250)
	}

	// 关闭 Kafka Writer
	if b.kafkaWriter != nil {
		b.kafkaWriter.Close()
	}

	b.logger.Info("Bridge stopped")
}

func (b *Bridge) onConnect(client mqtt.Client) {
	b.logger.Info("MQTT connected, subscribing to topic", zap.String("topic", b.config.MQTTTopic))

	// 订阅主题
	token := client.Subscribe(b.config.MQTTTopic, 1, b.onMessage)
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
	// 提取 projectId
	// Topic 格式: mtic/msg/client/realtime/tenant123/project456
	parts := strings.Split(msg.Topic(), "/")
	if len(parts) < 6 {
		b.logger.Warn("Invalid topic format", zap.String("topic", msg.Topic()))
		return
	}

	tenantID := parts[4]
	projectID := parts[5]

	// 构造 Kafka Topic: tenant123.project456
	kafkaTopic := fmt.Sprintf("%s.%s", tenantID, projectID)

	// 转发到 Kafka
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := b.kafkaWriter.WriteMessages(ctx, kafka.Message{
		Topic: kafkaTopic,
		Key:   []byte(projectID),
		Value: msg.Payload(),
	})

	if err != nil {
		b.logger.Error("Failed to write to Kafka",
			zap.Error(err),
			zap.String("kafka_topic", kafkaTopic),
		)
		return
	}

	b.logger.Debug("Message forwarded",
		zap.String("mqtt_topic", msg.Topic()),
		zap.String("kafka_topic", kafkaTopic),
		zap.Int("size", len(msg.Payload())),
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
