# EKS Troubleshooting Guide

> 运维知识库 - AWS EKS 集群创建与管理问题排查指南

## 目录

- [环境信息](#环境信息)
- [问题速查表](#问题速查表)
- [详细问题与解决方案](#详细问题与解决方案)
- [AWS账号限制查询](#aws账号限制查询)
- [常用诊断命令](#常用诊断命令)
- [最佳实践](#最佳实践)
- [参考配置](#参考配置)

---

## 环境信息

| 项目 | 值 |
|------|-----|
| EKS版本 | 1.29 |
| 区域 | us-east-1 |
| 工具版本 | eksctl 0.218.0 |
| 账号类型 | Free Tier ($300额度) |

---

## 问题速查表

| 错误关键词 | 可能原因 | 快速解决 |
|------------|----------|----------|
| `not eligible for Free Tier` | 实例类型不在Free Tier列表 | 改用t3.small/t3.micro |
| `TerminationProtection is enabled` | CloudFormation栈有删除保护 | 先禁用保护再删除 |
| `exceeded max wait time` | eksctl等待超时 | 检查实际状态，可能仍在创建中 |
| `dial tcp [::1]:8080` | kubectl未配置 | 运行aws eks update-kubeconfig |
| `No node group found` | 节点组创建失败被删除 | 查看CloudFormation事件找原因 |
| `AsgInstanceLaunchFailures` | EC2实例启动失败 | 检查实例类型、配额、Spot容量 |

---

## 详细问题与解决方案

### 问题1：Free Tier实例类型限制

#### 现象

```
InvalidParameterCombination - The specified instance type is not eligible for Free Tier
```

#### 原因

AWS Free Tier账号或新账号只能使用特定的实例类型，不是所有实例都可用。

#### 解决方案

1. 查询可用的Free Tier实例类型：

```bash
aws ec2 describe-instance-types \
  --filters "Name=free-tier-eligible,Values=true" \
  --query 'InstanceTypes[*].[InstanceType,VCpuInfo.DefaultVCpus,MemoryInfo.SizeInMiB]' \
  --output table
```

2. 常见Free Tier实例类型：

| 实例类型 | vCPU | 内存 | 适用场景 |
|----------|------|------|----------|
| t3.micro | 2 | 1GB | 最小测试 |
| t3.small | 2 | 2GB | 开发环境（推荐） |
| t4g.micro | 2 | 1GB | ARM架构测试 |
| t4g.small | 2 | 2GB | ARM架构开发 |

3. 修改nodegroup配置：

```yaml
managedNodeGroups:
  - name: workers
    instanceType: t3.small    # 使用Free Tier支持的类型
    spot: false               # Free Tier限制下避免使用Spot
```

---

### 问题2：Spot实例容量不足

#### 现象

```
Could not launch Spot Instances. InsufficientInstanceCapacity
```

或eksctl长时间等待后超时。

#### 原因

- 所选区域/可用区的Spot实例容量不足
- 特定实例类型的Spot价格波动导致无法获取

#### 解决方案

**方案A：改用按需实例**

```yaml
managedNodeGroups:
  - name: workers
    spot: false    # 改为按需实例
```

**方案B：指定多个实例类型（增加Spot获取概率）**

```yaml
managedNodeGroups:
  - name: workers
    instanceTypes: ["t3.medium", "t3.large", "t3a.medium", "t3a.large"]
    spot: true
```

**方案C：混合使用按需和Spot**

```yaml
managedNodeGroups:
  - name: on-demand
    instanceType: t3.small
    desiredCapacity: 1
    spot: false
  - name: spot
    instanceTypes: ["t3.small", "t3.medium"]
    desiredCapacity: 2
    spot: true
```

---

### 问题3：CloudFormation栈删除保护

#### 现象

```
Stack [xxx] cannot be deleted while TerminationProtection is enabled
```

#### 解决方案

```bash
# 1. 禁用终止保护
aws cloudformation update-termination-protection \
  --no-enable-termination-protection \
  --stack-name <stack-name>

# 2. 删除栈
aws cloudformation delete-stack --stack-name <stack-name>

# 3. 等待删除完成
aws cloudformation wait stack-delete-complete --stack-name <stack-name>
```

#### 批量清理eksctl相关栈

```bash
# 列出所有相关栈
aws cloudformation list-stacks \
  --query 'StackSummaries[?contains(StackName,`iot-platform`)].[StackName,StackStatus]' \
  --output table

# 删除指定栈（需要先禁用保护）
for stack in $(aws cloudformation list-stacks \
  --query 'StackSummaries[?contains(StackName,`iot-platform`) && StackStatus!=`DELETE_COMPLETE`].StackName' \
  --output text); do
  echo "Deleting $stack..."
  aws cloudformation update-termination-protection \
    --no-enable-termination-protection \
    --stack-name $stack 2>/dev/null
  aws cloudformation delete-stack --stack-name $stack
done
```

---

### 问题4：kubectl连接失败

#### 现象

```
dial tcp [::1]:8080: connectex: No connection could be made because the target machine actively refused it
```

#### 原因

kubectl没有配置正确的kubeconfig，默认连接localhost:8080。

#### 解决方案

```bash
# 更新kubeconfig
aws eks update-kubeconfig --region us-east-1 --name iot-platform

# 验证配置
kubectl config current-context

# 测试连接
kubectl get nodes
```

#### 多集群管理

```bash
# 查看所有context
kubectl config get-contexts

# 切换context
kubectl config use-context <context-name>

# 指定kubeconfig文件
export KUBECONFIG=~/.kube/config-iot-platform
```

---

### 问题5：eksctl超时但资源仍在创建

#### 现象

```
exceeded max wait time for StackCreateComplete waiter
Error: failed to create nodegroups for cluster "iot-platform"
```

#### 诊断步骤

```bash
# 1. 检查节点组实际状态
aws eks describe-nodegroup \
  --cluster-name iot-platform \
  --nodegroup-name workers \
  --query 'nodegroup.status' \
  --output text

# 状态说明：
# CREATING - 仍在创建，等待即可
# ACTIVE - 创建成功
# CREATE_FAILED - 创建失败
# DELETING - 正在删除

# 2. 检查健康状态
aws eks describe-nodegroup \
  --cluster-name iot-platform \
  --nodegroup-name workers \
  --query 'nodegroup.health' \
  --output json

# 3. 检查EC2实例状态
aws ec2 describe-instances \
  --filters "Name=tag:eks:cluster-name,Values=iot-platform" \
  --query 'Reservations[].Instances[].[InstanceId,State.Name,InstanceType,LaunchTime]' \
  --output table
```

#### 如果状态是CREATING，持续监控

```bash
# 每30秒检查一次
while true; do
  status=$(aws eks describe-nodegroup \
    --cluster-name iot-platform \
    --nodegroup-name workers \
    --query 'nodegroup.status' \
    --output text 2>/dev/null)
  echo "$(date): $status"
  [ "$status" = "ACTIVE" ] && echo "成功!" && break
  [ "$status" = "CREATE_FAILED" ] && echo "失败!" && break
  sleep 30
done
```

---

## AWS账号限制查询

### 查询Free Tier支持的实例类型

```bash
aws ec2 describe-instance-types \
  --filters "Name=free-tier-eligible,Values=true" \
  --query 'InstanceTypes[*].[InstanceType,VCpuInfo.DefaultVCpus,MemoryInfo.SizeInMiB]' \
  --output table
```

### 查询EC2实例配额

```bash
# 按需实例vCPU配额
aws service-quotas get-service-quota \
  --service-code ec2 \
  --quota-code L-1216C47A \
  --query 'Quota.{Name:QuotaName,Value:Value}' \
  --output table

# 所有运行实例相关配额
aws service-quotas list-service-quotas \
  --service-code ec2 \
  --query 'Quotas[?contains(QuotaName,`Running`)][QuotaName,Value]' \
  --output table
```

### 查询当前使用量

```bash
# 当月费用
aws ce get-cost-and-usage \
  --time-period Start=$(date -d "$(date +%Y-%m-01)" +%Y-%m-%d),End=$(date +%Y-%m-%d) \
  --granularity MONTHLY \
  --metrics "UnblendedCost" \
  --query 'ResultsByTime[0].Total.UnblendedCost' \
  --output json
```

### AWS控制台链接

| 用途 | 链接 |
|------|------|
| Free Tier使用情况 | https://console.aws.amazon.com/billing/home#/freetier |
| Service Quotas | https://console.aws.amazon.com/servicequotas/home |
| EC2限制 | https://console.aws.amazon.com/ec2/v2/home#Limits |
| CloudFormation栈 | https://console.aws.amazon.com/cloudformation/home |
| EKS集群 | https://console.aws.amazon.com/eks/home |

### 新账号常见限制

| 限制类型 | 典型默认值 | 说明 |
|----------|------------|------|
| 按需实例vCPU | 5-32个 | 新账号通常较低 |
| Spot实例vCPU | 5个 | 需申请提升 |
| 弹性IP | 5个/region | 超出需申请 |
| EBS存储 | 30GB Free | 超出收费 |
| NAT Gateway | 5个/AZ | 按小时收费 |

### 申请提升配额

```bash
# 命令行申请
aws service-quotas request-service-quota-increase \
  --service-code ec2 \
  --quota-code L-1216C47A \
  --desired-value 64

# 或通过控制台
# Service Quotas → EC2 → 选择配额 → Request quota increase
```

---

## 常用诊断命令

### 集群级别

```bash
# 列出所有EKS集群
eksctl get cluster --region us-east-1

# 集群详情
aws eks describe-cluster --name iot-platform --query 'cluster.status'

# 集群端点
aws eks describe-cluster --name iot-platform \
  --query 'cluster.endpoint' --output text
```

### 节点组级别

```bash
# 列出节点组
aws eks list-nodegroups --cluster-name iot-platform

# 节点组详情
aws eks describe-nodegroup \
  --cluster-name iot-platform \
  --nodegroup-name workers

# 节点组健康检查
aws eks describe-nodegroup \
  --cluster-name iot-platform \
  --nodegroup-name workers \
  --query 'nodegroup.health'
```

### CloudFormation级别

```bash
# 列出相关栈
aws cloudformation list-stacks \
  --query 'StackSummaries[?contains(StackName,`iot-platform`)]'

# 查看栈事件（找错误原因）
aws cloudformation describe-stack-events \
  --stack-name <stack-name> \
  --output json | grep -i "reason"

# 查看CREATE_FAILED事件
aws cloudformation describe-stack-events \
  --stack-name <stack-name> \
  --query 'StackEvents[?ResourceStatus==`CREATE_FAILED`]'
```

### Kubernetes级别

```bash
# 节点状态
kubectl get nodes -o wide

# 节点详情
kubectl describe node <node-name>

# 系统Pod状态
kubectl get pods -n kube-system

# 查看事件
kubectl get events --sort-by='.lastTimestamp'
```

---

## 最佳实践

### 1. 集群创建前检查清单

- [ ] 确认AWS区域和可用区
- [ ] 查询账号Free Tier实例类型限制
- [ ] 检查EC2 vCPU配额
- [ ] 确认VPC CIDR不冲突
- [ ] 准备好IAM权限

### 2. 使用配置文件而非命令行

```bash
# ❌ 不推荐
eksctl create cluster --name xxx --region xxx --nodegroup-name xxx ...

# ✅ 推荐
eksctl create cluster -f cluster.yaml
```

### 3. 分离集群和节点组配置

```
eksctl/
├── cluster.yaml      # 集群配置（不含节点组）
└── nodegroup.yaml    # 节点组配置（可独立管理）
```

### 4. 先dry-run验证

```bash
eksctl create cluster -f cluster.yaml --dry-run
eksctl create nodegroup -f nodegroup.yaml --dry-run
```

### 5. 节点组创建失败时的清理步骤

```bash
# 1. 查看失败原因
aws cloudformation describe-stack-events \
  --stack-name eksctl-<cluster>-nodegroup-<name> \
  --output json | grep -i "reason"

# 2. 禁用删除保护
aws cloudformation update-termination-protection \
  --no-enable-termination-protection \
  --stack-name eksctl-<cluster>-nodegroup-<name>

# 3. 删除失败的栈
aws cloudformation delete-stack \
  --stack-name eksctl-<cluster>-nodegroup-<name>

# 4. 等待删除完成
aws cloudformation wait stack-delete-complete \
  --stack-name eksctl-<cluster>-nodegroup-<name>

# 5. 修改配置后重新创建
eksctl create nodegroup -f nodegroup.yaml
```

---

## 参考配置

### cluster.yaml（集群配置）

```yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig

metadata:
  name: iot-platform
  region: us-east-1
  version: "1.29"
  tags:
    Project: iot-platform
    Environment: dev
    ManagedBy: eksctl

iam:
  withOIDC: true

vpc:
  cidr: 10.0.0.0/16
  nat:
    gateway: Single
  clusterEndpoints:
    publicAccess: true
    privateAccess: true

cloudWatch:
  clusterLogging:
    enableTypes: []

addons:
  - name: vpc-cni
    version: latest
  - name: coredns
    version: latest
  - name: kube-proxy
    version: latest
  - name: aws-ebs-csi-driver
    version: latest
    attachPolicyARNs:
      - arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy
```

### nodegroup.yaml（Free Tier兼容）

```yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig

metadata:
  name: iot-platform
  region: us-east-1

managedNodeGroups:
  - name: workers
    instanceType: t3.small        # Free Tier支持
    desiredCapacity: 2
    minSize: 1
    maxSize: 3
    volumeSize: 30
    volumeType: gp3
    spot: false                   # 按需实例更稳定
    
    labels:
      role: worker
      environment: dev
    
    iam:
      withAddonPolicies:
        ebs: true
        albIngress: true
        cloudWatch: true
```

---

## 更新日志

| 日期 | 更新内容 |
|------|----------|
| 2025-11-27 | 初始版本，记录EKS集群创建问题 |

---

## 贡献

遇到新问题？请按以下格式添加：

```markdown
### 问题N：标题

#### 现象
错误信息或表现

#### 原因
根本原因分析

#### 解决方案
具体步骤
```
