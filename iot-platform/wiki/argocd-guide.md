# ArgoCD 安装与使用指南

> 运维知识库 - ArgoCD GitOps 持续部署平台

## 目录

- [概述](#概述)
- [安装步骤](#安装步骤)
- [访问配置](#访问配置)
- [核心概念](#核心概念)
- [常用操作](#常用操作)
- [Application配置详解](#application配置详解)
- [最佳实践](#最佳实践)
- [故障排查](#故障排查)

---

## 概述

### 什么是ArgoCD

ArgoCD是一个声明式的GitOps持续部署工具，它：

- 监控Git仓库中的Kubernetes配置
- 自动将集群状态同步到Git定义的期望状态
- 提供可视化UI管理应用部署

### 核心理念

```
Git Repository (期望状态)
        │
        ▼
    ArgoCD (检测差异)
        │
        ▼
Kubernetes Cluster (实际状态)
```

**Git是唯一真相来源**，所有变更通过Git提交，ArgoCD自动同步。

---

## 安装步骤

### 1. 创建Namespace

```bash
kubectl create namespace argocd
```

### 2. 安装ArgoCD

```bash
# 使用官方manifests安装
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

### 3. 等待Pod就绪

```bash
# 等待所有Pod就绪（约2-3分钟）
kubectl wait --for=condition=Ready pods --all -n argocd --timeout=300s

# 验证安装
kubectl get pods -n argocd
```

### 安装的组件

| 组件 | 作用 | 资源类型 |
|------|------|----------|
| argocd-server | API服务器和Web UI | Deployment |
| argocd-repo-server | Git仓库克隆和管理 | Deployment |
| argocd-application-controller | 应用同步控制器 | StatefulSet |
| argocd-applicationset-controller | 批量应用管理 | Deployment |
| argocd-dex-server | SSO/OIDC认证 | Deployment |
| argocd-redis | 缓存层 | Deployment |
| argocd-notifications-controller | 通知服务 | Deployment |

---

## 访问配置

### 方式1：LoadBalancer（公网访问）

```bash
# 将服务改为LoadBalancer类型
kubectl patch svc argocd-server -n argocd -p '{"spec": {"type": "LoadBalancer"}}'

# 等待获取外部地址（约1-2分钟）
kubectl get svc argocd-server -n argocd -w

# 输出示例：
# NAME            TYPE           CLUSTER-IP     EXTERNAL-IP                                      PORT(S)
# argocd-server   LoadBalancer   172.20.7.186   xxx.us-east-1.elb.amazonaws.com                  80:31605/TCP,443:30604/TCP
```

访问：`https://<EXTERNAL-IP>`

### 方式2：Port Forward（本地访问）

```bash
# 在本地终端运行（保持运行）
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

访问：`https://localhost:8080`

### 方式3：Ingress（生产推荐）

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: argocd-server
  namespace: argocd
  annotations:
    nginx.ingress.kubernetes.io/ssl-passthrough: "true"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
spec:
  ingressClassName: nginx
  rules:
    - host: argocd.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: argocd-server
                port:
                  number: 443
  tls:
    - hosts:
        - argocd.example.com
      secretName: argocd-tls
```

### 获取登录凭据

```bash
# 获取admin初始密码
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
echo

# 登录信息
# 用户名: admin
# 密码: 上面命令的输出
```

### 修改密码（可选）

```bash
# 安装argocd CLI
# MacOS
brew install argocd

# Linux
curl -sSL -o /usr/local/bin/argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
chmod +x /usr/local/bin/argocd

# Windows (PowerShell)
# 从 https://github.com/argoproj/argo-cd/releases 下载

# 登录并修改密码
argocd login <ARGOCD_SERVER>
argocd account update-password
```

---

## 核心概念

### Application

Application是ArgoCD的核心资源，定义了：

- **源（Source）**：Git仓库地址、分支、路径
- **目标（Destination）**：部署到哪个集群和namespace
- **同步策略（Sync Policy）**：自动/手动同步

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/org/repo.git
    targetRevision: main
    path: k8s/
  destination:
    server: https://kubernetes.default.svc
    namespace: my-namespace
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### Project

Project用于多租户隔离，限制：

- 可访问的Git仓库
- 可部署的集群和namespace
- 可部署的资源类型

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: production
  namespace: argocd
spec:
  description: Production applications
  sourceRepos:
    - 'https://github.com/myorg/*'
  destinations:
    - namespace: 'prod-*'
      server: https://kubernetes.default.svc
  clusterResourceWhitelist:
    - group: ''
      kind: Namespace
```

### 同步状态

| 状态 | 含义 |
|------|------|
| Synced | 集群状态与Git一致 |
| OutOfSync | 集群状态与Git不一致 |
| Unknown | 无法确定状态 |

### 健康状态

| 状态 | 含义 |
|------|------|
| Healthy | 所有资源健康 |
| Progressing | 正在部署中 |
| Degraded | 部分资源异常 |
| Suspended | 已暂停 |
| Missing | 资源不存在 |

---

## 常用操作

### 查看应用

```bash
# 列出所有应用
kubectl get applications -n argocd

# 查看应用详情
kubectl describe application <app-name> -n argocd

# 使用argocd CLI
argocd app list
argocd app get <app-name>
```

### 同步应用

```bash
# 手动同步
argocd app sync <app-name>

# 强制同步（忽略差异）
argocd app sync <app-name> --force

# 同步并等待完成
argocd app sync <app-name> --timeout 300
```

### 回滚应用

```bash
# 查看历史
argocd app history <app-name>

# 回滚到指定版本
argocd app rollback <app-name> <history-id>
```

### 删除应用

```bash
# 删除应用（保留K8s资源）
argocd app delete <app-name>

# 删除应用及其K8s资源
argocd app delete <app-name> --cascade
```

---

## Application配置详解

### 使用Helm Chart

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: prometheus
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://prometheus-community.github.io/helm-charts
    chart: prometheus
    targetRevision: 25.0.0
    helm:
      values: |
        server:
          resources:
            limits:
              memory: 512Mi
            requests:
              memory: 256Mi
  destination:
    server: https://kubernetes.default.svc
    namespace: monitoring
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### 使用Kustomize

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/org/repo.git
    targetRevision: main
    path: kustomize/overlays/production
  destination:
    server: https://kubernetes.default.svc
    namespace: production
```

### 使用纯YAML目录

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/org/repo.git
    targetRevision: main
    path: manifests/
    directory:
      recurse: true
  destination:
    server: https://kubernetes.default.svc
    namespace: default
```

### 同步策略选项

```yaml
syncPolicy:
  automated:
    prune: true           # 删除Git中已移除的资源
    selfHeal: true        # 自动修复集群漂移
    allowEmpty: false     # 不允许删除所有资源
  syncOptions:
    - CreateNamespace=true          # 自动创建namespace
    - PrunePropagationPolicy=foreground  # 删除顺序
    - PruneLast=true                # 最后再删除旧资源
  retry:
    limit: 5              # 重试次数
    backoff:
      duration: 5s
      factor: 2
      maxDuration: 3m
```

---

## 最佳实践

### 1. 项目结构

```
iot-platform/
├── argocd-apps/
│   ├── platform/              # 基础平台组件
│   │   ├── kafka.yaml
│   │   ├── prometheus.yaml
│   │   └── argocd.yaml
│   └── applications/          # 业务应用
│       ├── mqtt-broker.yaml
│       └── processors.yaml
├── helm-charts/               # 自定义Charts
└── kustomize/                 # 环境配置
```

### 2. App of Apps模式

用一个Application管理其他所有Applications：

```yaml
# argocd-apps/root.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: root
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/org/iot-platform.git
    targetRevision: main
    path: argocd-apps
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### 3. 环境分离

```yaml
# argocd-apps/platform/kafka-dev.yaml
spec:
  source:
    helm:
      valueFiles:
        - values-dev.yaml

# argocd-apps/platform/kafka-prod.yaml
spec:
  source:
    helm:
      valueFiles:
        - values-prod.yaml
```

### 4. 资源限制

为ArgoCD组件设置资源限制，特别是在小集群上：

```bash
# 查看当前资源使用
kubectl top pods -n argocd

# 编辑deployment调整资源
kubectl edit deployment argocd-server -n argocd
```

### 5. 安全建议

- 使用SSO而非admin账号
- 启用RBAC限制权限
- 使用私有Git仓库
- 定期轮换密码和Token

---

## 故障排查

### 应用同步失败

```bash
# 查看应用状态
argocd app get <app-name>

# 查看同步详情
kubectl describe application <app-name> -n argocd

# 查看Controller日志
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-application-controller
```

### UI无法访问

```bash
# 检查Pod状态
kubectl get pods -n argocd

# 检查Service
kubectl get svc -n argocd

# 检查argocd-server日志
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-server
```

### 仓库连接失败

```bash
# 测试仓库连接
argocd repo add <repo-url> --username <user> --password <pass>

# 查看repo-server日志
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-repo-server
```

### 常见错误

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| ComparisonError | 无法比较状态 | 检查repo-server日志 |
| InvalidSpecError | Application配置错误 | 验证YAML语法 |
| PermissionDenied | RBAC限制 | 检查Project权限 |
| ConnectionRefused | 集群连接失败 | 检查网络和凭据 |

---

## 参考资源

- [ArgoCD官方文档](https://argo-cd.readthedocs.io/)
- [ArgoCD GitHub](https://github.com/argoproj/argo-cd)
- [Helm Charts仓库](https://artifacthub.io/)

---

## 更新日志

| 日期 | 更新内容 |
|------|----------|
| 2025-11-27 | 初始版本，EKS上安装ArgoCD |
