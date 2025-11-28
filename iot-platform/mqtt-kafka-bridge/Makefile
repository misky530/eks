.PHONY: help build test run clean docker-build docker-push deploy

help: ## 显示帮助信息
	@echo "MQTT-Kafka Bridge Makefile"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## 构建 Go 二进制文件
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/bridge ./cmd/bridge

test: ## 运行测试
	go test -v ./...

run: ## 本地运行
	go run cmd/bridge/main.go

clean: ## 清理构建文件
	rm -rf bin/

docker-build: ## 构建 Docker 镜像
	docker build -t mqtt-kafka-bridge:latest .

docker-run: ## 本地运行 Docker 容器
	docker run --rm \
		-e MQTT_BROKER=tcp://hats.hcs.cn:1883 \
		-e KAFKA_BROKERS=host.docker.internal:9092 \
		mqtt-kafka-bridge:latest

deploy: ## 一键部署到 EKS
	./deploy.sh

logs: ## 查看 Pod 日志
	kubectl logs -f -n iot-bridge -l app=mqtt-kafka-bridge

status: ## 查看部署状态
	@echo "=== ArgoCD Application ==="
	kubectl get application -n argocd mqtt-kafka-bridge
	@echo ""
	@echo "=== Pods ==="
	kubectl get pods -n iot-bridge
	@echo ""
	@echo "=== Resource Usage ==="
	kubectl top pod -n iot-bridge 2>/dev/null || echo "Metrics server not available"

delete: ## 删除部署
	kubectl delete application -n argocd mqtt-kafka-bridge
	kubectl delete namespace iot-bridge
