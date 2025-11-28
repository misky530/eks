#!/bin/bash
set -e

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}MQTT-Kafka Bridge Deployment Script${NC}"
echo -e "${GREEN}========================================${NC}"

# 检查必要的命令
command -v docker >/dev/null 2>&1 || { echo -e "${RED}Error: docker not found${NC}" >&2; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo -e "${RED}Error: kubectl not found${NC}" >&2; exit 1; }
command -v aws >/dev/null 2>&1 || { echo -e "${RED}Error: aws cli not found${NC}" >&2; exit 1; }

# 检查 AWS 配置
echo -e "\n${YELLOW}Checking AWS configuration...${NC}"
if ! aws sts get-caller-identity >/dev/null 2>&1; then
    echo -e "${RED}Error: AWS credentials not configured${NC}"
    echo -e "${YELLOW}Please run: aws configure${NC}"
    exit 1
fi

# 自动获取 AWS Account ID
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
echo -e "${GREEN}✓ AWS Account ID: ${AWS_ACCOUNT_ID}${NC}"

# 读取配置
echo -e "\n${YELLOW}Please provide the following information:${NC}"
read -p "AWS Region [us-east-1]: " AWS_REGION
AWS_REGION=${AWS_REGION:-us-east-1}

ECR_REPO="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/mqtt-kafka-bridge"

# 1. 创建 ECR 仓库
echo -e "\n${GREEN}[1/5] Creating ECR repository...${NC}"
aws ecr describe-repositories --repository-names mqtt-kafka-bridge --region $AWS_REGION >/dev/null 2>&1 || \
  aws ecr create-repository --repository-name mqtt-kafka-bridge --region $AWS_REGION

# 2. 登录 ECR
echo -e "\n${GREEN}[2/5] Logging into ECR...${NC}"
aws ecr get-login-password --region $AWS_REGION | \
  docker login --username AWS --password-stdin $ECR_REPO

# 3. 构建镜像
echo -e "\n${GREEN}[3/5] Building Docker image...${NC}"
docker build -t mqtt-kafka-bridge:latest .

# 4. 推送镜像
echo -e "\n${GREEN}[4/5] Pushing image to ECR...${NC}"
docker tag mqtt-kafka-bridge:latest ${ECR_REPO}:latest
docker push ${ECR_REPO}:latest

# 5. 更新 Kubernetes 配置
echo -e "\n${GREEN}[5/5] Updating Kubernetes configuration...${NC}"
sed -i.bak "s|<YOUR_ECR_REPO>|${ECR_REPO}|g" deployments/kubernetes/deployment.yaml
rm -f deployments/kubernetes/deployment.yaml.bak

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}Deployment preparation completed!${NC}"
echo -e "${GREEN}========================================${NC}"

echo -e "\n${YELLOW}Next steps:${NC}"
echo -e "1. Update ArgoCD application configuration:"
echo -e "   ${YELLOW}deployments/kubernetes/argocd-app.yaml${NC}"
echo -e "   Set 'repoURL' to your Git repository"
echo -e ""
echo -e "2. Commit and push to Git:"
echo -e "   ${YELLOW}git add .${NC}"
echo -e "   ${YELLOW}git commit -m 'Add MQTT-Kafka Bridge'${NC}"
echo -e "   ${YELLOW}git push origin main${NC}"
echo -e ""
echo -e "3. Deploy via ArgoCD:"
echo -e "   ${YELLOW}kubectl apply -f deployments/kubernetes/argocd-app.yaml${NC}"
echo -e ""
echo -e "4. Monitor deployment:"
echo -e "   ${YELLOW}kubectl get pods -n iot-bridge -w${NC}"
echo -e ""
