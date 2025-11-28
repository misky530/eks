package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Config struct {
	KafkaBrokers []string
	KafkaTopic   string
	DBHost       string
	DBPort       string
	DBName       string
	DBUser       string
	DBPassword   string
	BatchSize    int
	FlushInterval time.Duration
}

type Message struct {
	TenantID  string    `json:"tenant_id"`
	ProjectID string    `json:"project_id"`
	Topic     string    `json:"mqtt_topic"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

type Consumer struct {
	config Config
	logger *zap.Logger
	db     *sql.DB
	reader *kafka.Reader
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	config := Config{
		KafkaBrokers:  strings.Split(getEnv("KAFKA_BROKERS", "iot-cluster-kafka-bootstrap.kafka:9092"), ","),
		KafkaTopic:    getEnv("KAFKA_TOPIC", "iot-messages"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBName:        getEnv("DB_NAME", "iot_data"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
	}

	logger.Info("Starting Kafka Consumer",
		zap.Strings("kafka_brokers", config.KafkaBrokers),
		zap.String("kafka_topic", config.KafkaTopic),
		zap.String("db_host", config.DBHost),
	)

	consumer, err := NewConsumer(config, logger)
	if err != nil {
		logger.Fatal("Failed to create consumer", zap.Error(err))
	}
	defer consumer.Close()

	if err := consumer.Start(); err != nil {
		logger.Fatal("Failed to start consumer", zap.Error(err))
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down gracefully...")
}

func NewConsumer(config Config, logger *zap.Logger) (*Consumer, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Connected to database")

	if err := initDatabase(db, logger); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     config.KafkaBrokers,
		Topic:       config.KafkaTopic,
		GroupID:     "iot-consumer-group",
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})

	return &Consumer{
		config: config,
		logger: logger,
		db:     db,
		reader: reader,
	}, nil
}

func initDatabase(db *sql.DB, logger *zap.Logger) error {
	logger.Info("Initializing database schema...")

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS iot_messages (
			time        TIMESTAMPTZ NOT NULL,
			tenant_id   TEXT NOT NULL,
			project_id  TEXT NOT NULL,
			mqtt_topic  TEXT,
			payload     TEXT,
			created_at  TIMESTAMPTZ DEFAULT NOW(),
			PRIMARY KEY (time, tenant_id, project_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_tenant_time ON iot_messages (tenant_id, time DESC);
		CREATE INDEX IF NOT EXISTS idx_project_time ON iot_messages (project_id, time DESC);
	`)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	logger.Info("Database schema initialized")
	return nil
}

func (c *Consumer) Start() error {
	c.logger.Info("Starting message consumption...")

	ctx := context.Background()
	batch := make([]Message, 0, c.config.BatchSize)
	ticker := time.NewTicker(c.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if len(batch) > 0 {
				c.flushBatch(ctx, batch)
				batch = batch[:0]
			}

		default:
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			msg, err := c.reader.ReadMessage(ctx)
			cancel()

			if err != nil {
				if err == context.DeadlineExceeded {
					if len(batch) > 0 {
						c.flushBatch(context.Background(), batch)
						batch = batch[:0]
					}
					continue
				}
				c.logger.Error("Failed to read message", zap.Error(err))
				continue
			}

			var message Message
			if err := json.Unmarshal(msg.Value, &message); err != nil {
				c.logger.Error("Failed to unmarshal message", zap.Error(err))
				continue
			}

			batch = append(batch, message)

			if len(batch) >= c.config.BatchSize {
				c.flushBatch(context.Background(), batch)
				batch = batch[:0]
			}
		}
	}
}

func (c *Consumer) flushBatch(ctx context.Context, batch []Message) {
	if len(batch) == 0 {
		return
	}

	start := time.Now()

	stmt := `
		INSERT INTO iot_messages (time, tenant_id, project_id, mqtt_topic, payload)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (time, tenant_id, project_id) DO NOTHING
	`

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		c.logger.Error("Failed to begin transaction", zap.Error(err))
		return
	}
	defer tx.Rollback()

	for _, msg := range batch {
		_, err := tx.Exec(stmt, msg.Timestamp, msg.TenantID, msg.ProjectID, msg.Topic, msg.Payload)
		if err != nil {
			c.logger.Error("Failed to insert message",
				zap.Error(err),
				zap.String("tenant_id", msg.TenantID),
				zap.String("project_id", msg.ProjectID))
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.logger.Error("Failed to commit transaction", zap.Error(err))
		return
	}

	duration := time.Since(start)
	c.logger.Info("Batch inserted to database",
		zap.Int("count", len(batch)),
		zap.Duration("duration", duration))
}

func (c *Consumer) Close() {
	c.logger.Info("Closing consumer...")

	if c.reader != nil {
		c.reader.Close()
	}

	if c.db != nil {
		c.db.Close()
	}

	c.logger.Info("Consumer closed")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
