#!/bin/bash
set -e

echo "🚀 MQTT-Kafka Bridge - 快速部署"
echo ""

# 自动获取 AWS 信息
echo "📋 获取 AWS 信息..."
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
AWS_REGION=$(aws configure get region)
AWS_REGION=${AWS_REGION:-us-east-1}

echo "✓ AWS Account ID: ${AWS_ACCOUNT_ID}"
echo "✓ AWS Region: ${AWS_REGION}"
echo ""

ECR_REPO="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/mqtt-kafka-bridge"

# 1. 创建 ECR 仓库
echo "📦 [1/5] 创建 ECR 仓库..."
aws ecr describe-repositories --repository-names mqtt-kafka-bridge --region $AWS_REGION >/dev/null 2>&1 || \
  aws ecr create-repository --repository-name mqtt-kafka-bridge --region $AWS_REGION >/dev/null
echo "✓ ECR 仓库已就绪"

# 2. 登录 ECR
echo "🔐 [2/5] 登录 ECR..."
aws ecr get-login-password --region $AWS_REGION | \
  docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com
echo "✓ ECR 登录成功"

# 3. 构建镜像
echo "🏗️  [3/5] 构建 Docker 镜像..."
docker build -t mqtt-kafka-bridge:latest .
echo "✓ 镜像构建完成"

# 4. 推送镜像
echo "📤 [4/5] 推送镜像到 ECR..."
docker tag mqtt-kafka-bridge:latest ${ECR_REPO}:latest
docker push ${ECR_REPO}:latest
echo "✓ 镜像推送完成"

# 5. 更新 Kubernetes 配置
echo "📝 [5/5] 更新 Kubernetes 配置..."
sed -i.bak "s|<YOUR_ECR_REPO>|${ECR_REPO}|g" deployments/kubernetes/deployment.yaml
rm -f deployments/kubernetes/deployment.yaml.bak
echo "✓ 配置更新完成"

echo ""
echo "========================================="
echo "✅ 部署准备完成！"
echo "========================================="
echo ""
echo "📋 镜像信息:"
echo "   Repository: ${ECR_REPO}"
echo "   Tag: latest"
echo ""
echo "📝 下一步操作:"
echo ""
echo "1. 编辑 ArgoCD 配置文件:"
echo "   vim deployments/kubernetes/argocd-app.yaml"
echo "   修改 spec.source.repoURL 为你的 Git 仓库地址"
echo ""
echo "2. 提交到 Git:"
echo "   git add ."
echo "   git commit -m 'Add MQTT-Kafka Bridge'"
echo "   git push origin main"
echo ""
echo "3. 部署到 EKS:"
echo "   kubectl apply -f deployments/kubernetes/argocd-app.yaml"
echo ""
echo "4. 查看状态:"
echo "   kubectl get pods -n iot-bridge"
echo "   kubectl logs -f -n iot-bridge -l app=mqtt-kafka-bridge"
echo ""
