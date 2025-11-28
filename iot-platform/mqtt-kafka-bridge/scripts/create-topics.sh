#!/bin/bash
set -e

# Kafka Topic 预创建脚本
# 用于创建动态 Topic，避免 Bridge 首次写入时延迟

KAFKA_BROKER="iot-cluster-kafka-bootstrap.kafka:9092"
TENANT_ID="tenant123"

echo "Creating Kafka topics for tenant: ${TENANT_ID}"

# 常见项目 ID 列表（根据实际情况调整）
PROJECTS=(
  "project001"
  "project002"
  "project003"
)

for PROJECT in "${PROJECTS[@]}"; do
  TOPIC="${TENANT_ID}.${PROJECT}"
  
  echo "Creating topic: ${TOPIC}"
  
  kubectl run kafka-admin-$RANDOM --rm -it --restart=Never \
    --image=confluentinc/cp-kafka:latest \
    -- kafka-topics --create \
      --if-not-exists \
      --bootstrap-server ${KAFKA_BROKER} \
      --topic ${TOPIC} \
      --partitions 3 \
      --replication-factor 1 \
      --config retention.ms=604800000 \
      --config compression.type=snappy
done

echo ""
echo "Listing all topics:"
kubectl run kafka-admin-$RANDOM --rm -it --restart=Never \
  --image=confluentinc/cp-kafka:latest \
  -- kafka-topics --list --bootstrap-server ${KAFKA_BROKER}
