#!/bin/bash
set -e

# Kafka 消费测试脚本
# 用于验证 Bridge 是否正常转发消息到 Kafka

KAFKA_BROKER="iot-cluster-kafka-bootstrap.kafka:9092"
TENANT_ID="${1:-tenant123}"
PROJECT_ID="${2:-project001}"
TOPIC="${TENANT_ID}.${PROJECT_ID}"

echo "========================================="
echo "Kafka Consumer Test"
echo "========================================="
echo "Broker: ${KAFKA_BROKER}"
echo "Topic:  ${TOPIC}"
echo "========================================="
echo ""
echo "Waiting for messages... (Press Ctrl+C to stop)"
echo ""

kubectl run kafka-consumer-$RANDOM --rm -it --restart=Never \
  --image=confluentinc/cp-kafka:latest \
  -- kafka-console-consumer \
    --bootstrap-server ${KAFKA_BROKER} \
    --topic ${TOPIC} \
    --from-beginning \
    --property print.timestamp=true \
    --property print.key=true \
    --property print.value=true
