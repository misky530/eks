# Windows 部署指南

## 环境要求

- ✅ Git Bash (推荐) 或 PowerShell
- ✅ Docker Desktop for Windows
- ✅ AWS CLI
- ✅ kubectl

---

## 🚀 快速部署（Git Bash）

### ⚡ 方式 1：一键部署脚本（推荐）

```bash
# 使用简化版脚本（自动获取 AWS 信息）
chmod +x deploy-simple.sh
./deploy-simple.sh
```

脚本会自动：
1. 从 AWS CLI 获取你的 Account ID 和 Region
2. 创建 ECR 仓库
3. 登录 ECR
4. 构建 Docker 镜像
5. 推送到 ECR
6. 更新 Kubernetes 配置

**执行后只需 3 步**：
1. 编辑 `argocd-app.yaml` 设置 Git 仓库地址
2. 提交到 Git
3. `kubectl apply -f deployments/kubernetes/argocd-app.yaml`

---

### 📋 方式 2：手动部署（逐步执行）

如果你想了解每一步的细节：

### 第 1 步：解压项目

```bash
# 打开 Git Bash
cd ~/Downloads

# 解压
tar -xzf mqtt-kafka-bridge.tar.gz

# 进入目录
cd mqtt-kafka-bridge
```

### 第 2 步：验证 AWS 配置

```bash
# 检查 AWS 配置（你已经配置好了）
aws sts get-caller-identity

# 应该看到你的账号信息
# {
#     "UserId": "...",
#     "Account": "你的账号ID",
#     "Arn": "..."
# }

# 自动获取 AWS 信息
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
export AWS_REGION=$(aws configure get region)
echo "AWS Account: $AWS_ACCOUNT_ID"
echo "AWS Region: $AWS_REGION"
```

### 第 3 步：创建 ECR 仓库

```bash
aws ecr create-repository \
  --repository-name mqtt-kafka-bridge \
  --region $AWS_REGION
```

### 第 4 步：登录 ECR

```bash
aws ecr get-login-password --region $AWS_REGION | \
  docker login --username AWS --password-stdin \
  $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com
```

### 第 5 步：构建镜像

```bash
docker build -t mqtt-kafka-bridge:latest .
```

### 第 6 步：推送镜像

```bash
# 标记镜像
docker tag mqtt-kafka-bridge:latest \
  $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/mqtt-kafka-bridge:latest

# 推送
docker push \
  $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/mqtt-kafka-bridge:latest
```

### 第 7 步：更新配置文件

编辑 `deployments/kubernetes/deployment.yaml`：

```bash
# 使用你喜欢的编辑器（VS Code、Notepad++、vim 等）
code deployments/kubernetes/deployment.yaml

# 或者用 sed 自动替换
sed -i "s|<YOUR_ECR_REPO>|$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/mqtt-kafka-bridge|g" \
  deployments/kubernetes/deployment.yaml
```

编辑 `deployments/kubernetes/argocd-app.yaml`：

```yaml
spec:
  source:
    repoURL: https://github.com/你的用户名/你的仓库  # 修改这里
```

### 第 8 步：提交到 Git

```bash
# 初始化 Git（如果是新仓库）
git init
git add .
git commit -m "Add MQTT-Kafka Bridge"

# 关联远程仓库
git remote add origin https://github.com/你的用户名/你的仓库.git

# 推送
git push -u origin main
```

### 第 9 步：部署到 EKS

```bash
# 应用 ArgoCD Application
kubectl apply -f deployments/kubernetes/argocd-app.yaml

# 查看状态
kubectl get application -n argocd mqtt-kafka-bridge
```

### 第 10 步：验证部署

```bash
# 查看 Pod
kubectl get pods -n iot-bridge

# 查看日志
kubectl logs -f -n iot-bridge -l app=mqtt-kafka-bridge
```

---

## 💻 PowerShell 部署（替代方案）

如果不想用 Git Bash：

```powershell
# 1. 解压项目（使用 Windows 资源管理器或 7-Zip）

# 2. 打开 PowerShell，进入目录
cd D:\code2025\eks\iot-platform\mqtt-kafka-bridge

# 3. 自动获取 AWS 信息
$AWS_ACCOUNT_ID = (aws sts get-caller-identity --query Account --output text)
$AWS_REGION = (aws configure get region)
if ([string]::IsNullOrEmpty($AWS_REGION)) { $AWS_REGION = "us-east-1" }
$ECR_REPO = "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/mqtt-kafka-bridge"

Write-Host "AWS Account: $AWS_ACCOUNT_ID"
Write-Host "AWS Region: $AWS_REGION"

# 4. 创建 ECR 仓库
aws ecr create-repository --repository-name mqtt-kafka-bridge --region $AWS_REGION

# 5. 登录 ECR
aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"

# 6. 构建镜像
docker build -t mqtt-kafka-bridge:latest .

# 7. 推送镜像
docker tag mqtt-kafka-bridge:latest "${ECR_REPO}:latest"
docker push "${ECR_REPO}:latest"

# 8. 更新配置（手动编辑）
notepad deployments\kubernetes\deployment.yaml

# 9. 提交到 Git
git init
git add .
git commit -m "Add MQTT-Kafka Bridge"
git remote add origin https://github.com/你的用户名/你的仓库.git
git push -u origin main

# 10. 部署
kubectl apply -f deployments\kubernetes\argocd-app.yaml
```

---

## 🔧 常见问题

### Q: tar 命令不可用？
**A:** 安装 Git for Windows 会自带 Git Bash，里面有 tar 命令。
或者用 Windows 自带解压：右键 → 全部提取

### Q: deploy.sh 无法执行？
**A:** Windows 下运行：
```bash
# Git Bash
chmod +x deploy.sh
./deploy.sh

# 或者手动执行脚本中的命令
```

### Q: sed 命令不可用（PowerShell）？
**A:** 手动编辑文件，或者使用 Git Bash。

### Q: Docker 镜像构建慢？
**A:** 第一次构建需要下载 Go 镜像，后续会快很多。

---

## 📂 Windows 目录结构示例

解压后的目录（在你的 Windows）：

```
C:\Users\Anthony\Downloads\mqtt-kafka-bridge\
├── cmd\
│   └── bridge\
│       └── main.go
├── deployments\
│   └── kubernetes\
│       ├── deployment.yaml
│       └── argocd-app.yaml
├── scripts\
│   ├── create-topics.sh
│   └── test-consumer.sh
├── START_HERE.md
├── Dockerfile
├── deploy.sh
└── ...
```

**注意**：Windows 使用反斜杠 `\`，Git Bash 使用正斜杠 `/`。

---

## ⚡ 一键部署脚本（Git Bash）

已为你准备好！创建并运行：

```bash
# 使用简化版脚本
chmod +x deploy-simple.sh
./deploy-simple.sh
```

**脚本功能**：
- ✅ 自动获取 AWS Account ID 和 Region
- ✅ 创建 ECR 仓库（如果不存在）
- ✅ 登录 ECR
- ✅ 构建 Docker 镜像
- ✅ 推送镜像到 ECR
- ✅ 更新 Kubernetes 配置文件

**输出示例**：
```
🚀 MQTT-Kafka Bridge - 快速部署

📋 获取 AWS 信息...
✓ AWS Account ID: 123456789012
✓ AWS Region: us-east-1

📦 [1/5] 创建 ECR 仓库...
✓ ECR 仓库已就绪
🔐 [2/5] 登录 ECR...
✓ ECR 登录成功
🏗️  [3/5] 构建 Docker 镜像...
✓ 镜像构建完成
📤 [4/5] 推送镜像到 ECR...
✓ 镜像推送完成
📝 [5/5] 更新 Kubernetes 配置...
✓ 配置更新完成

✅ 部署准备完成！
```

---

## 🎯 推荐工作流程（Windows）

1. **解压项目** → Git Bash
2. **构建镜像** → Docker Desktop
3. **编辑配置** → VS Code / Notepad++
4. **Git 操作** → Git Bash
5. **部署应用** → kubectl (Git Bash 或 PowerShell)

---

## 📞 需要帮助？

- Git Bash 问题 → 确认安装了 Git for Windows
- Docker 问题 → 确认 Docker Desktop 正在运行
- kubectl 问题 → 确认配置了正确的 kubeconfig

参考完整文档：`README.md` 和 `DEPLOYMENT_CHECKLIST.md`
