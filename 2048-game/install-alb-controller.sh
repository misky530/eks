#!/bin/bash
set -e

echo "=== 安装 AWS Load Balancer Controller ==="

# 1. 关联 OIDC provider
echo "Step 1: 关联 OIDC provider..."
eksctl utils associate-iam-oidc-provider \
  --region us-east-1 \
  --cluster web-quickstart \
  --approve

# 2. 下载并创建 IAM policy
echo "Step 2: 创建 IAM policy..."
curl -so iam_policy.json https://raw.githubusercontent.com/kubernetes-sigs/aws-load-balancer-controller/v2.8.0/docs/install/iam_policy.json

# 检查 policy 是否已存在
if aws iam get-policy --policy-arn arn:aws:iam::645890933537:policy/AWSLoadBalancerControllerIAMPolicy 2>/dev/null; then
  echo "IAM policy already exists, skipping creation..."
else
  aws iam create-policy \
    --policy-name AWSLoadBalancerControllerIAMPolicy \
    --policy-document file://iam_policy.json
fi

# 3. 创建 Service Account
echo "Step 3: 创建 Service Account..."
eksctl create iamserviceaccount \
  --cluster=web-quickstart \
  --namespace=kube-system \
  --name=aws-load-balancer-controller \
  --role-name AmazonEKSLoadBalancerControllerRole \
  --attach-policy-arn=arn:aws:iam::645890933537:policy/AWSLoadBalancerControllerIAMPolicy \
  --approve \
  --override-existing-serviceaccounts \
  --region=us-east-1

# 4. 安装 Helm（如果没有）
if ! command -v helm &> /dev/null; then
  echo "Installing Helm..."
  curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi

# 5. 安装 controller
echo "Step 4: 安装 Load Balancer Controller..."
helm repo add eks https://aws.github.io/eks-charts
helm repo update

VPC_ID=$(aws eks describe-cluster --name web-quickstart --region us-east-1 --query "cluster.resourcesVpcConfig.vpcId" --output text)

helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=web-quickstart \
  --set serviceAccount.create=false \
  --set serviceAccount.name=aws-load-balancer-controller \
  --set region=us-east-1 \
  --set vpcId=$VPC_ID

# 6. 等待 controller 就绪
echo "Step 5: 等待 controller 就绪..."
kubectl wait --for=condition=available --timeout=300s deployment/aws-load-balancer-controller -n kube-system

echo "=== 安装完成 ==="
echo "现在查看 Ingress 状态："
kubectl get ingress -n game-2048
