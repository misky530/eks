# 📚 MQTT-Kafka Bridge 文档索引

快速找到你需要的文档！

---

## 🚀 快速开始

**我想立即部署** → [`QUICKSTART.md`](QUICKSTART.md)
- 5 分钟快速部署指南
- 一键部署脚本使用
- 验证和测试步骤

---

## 📖 详细文档

### 核心文档

| 文档 | 用途 | 适合人群 |
|------|------|---------|
| [`README.md`](README.md) | 完整项目文档 | 所有人 |
| [`PROJECT_SUMMARY.md`](PROJECT_SUMMARY.md) | 项目交付总结 | 第一次了解项目 |
| [`ARCHITECTURE.md`](ARCHITECTURE.md) | 架构设计详解 | 深入理解实现 |

### 操作指南

| 文档 | 用途 | 什么时候用 |
|------|------|----------|
| [`QUICKSTART.md`](QUICKSTART.md) | 快速部署 | 立即部署时 |
| [`DEPLOYMENT_CHECKLIST.md`](DEPLOYMENT_CHECKLIST.md) | 部署检查清单 | 部署前后验证 |
| [`ENV_CONFIG.md`](ENV_CONFIG.md) | 环境变量配置 | 修改配置时 |

### 参考资料

| 文档 | 用途 | 什么时候用 |
|------|------|----------|
| [`DIAGRAMS.md`](DIAGRAMS.md) | 流程图和架构图 | 理解系统流程 |

---

## 🗂️ 按场景查找

### 场景 1: 我是第一次接触这个项目
1. 阅读 [`PROJECT_SUMMARY.md`](PROJECT_SUMMARY.md) - 了解项目全貌
2. 浏览 [`DIAGRAMS.md`](DIAGRAMS.md) - 查看架构图
3. 运行 [`QUICKSTART.md`](QUICKSTART.md) - 5 分钟部署

### 场景 2: 我要部署到生产环境
1. 检查 [`DEPLOYMENT_CHECKLIST.md`](DEPLOYMENT_CHECKLIST.md) - 逐项确认
2. 参考 [`ENV_CONFIG.md`](ENV_CONFIG.md) - 配置环境变量
3. 使用 [`QUICKSTART.md`](QUICKSTART.md) - 部署流程

### 场景 3: 我遇到了问题
1. 查看 [`DEPLOYMENT_CHECKLIST.md`](DEPLOYMENT_CHECKLIST.md) 故障排查章节
2. 参考 [`README.md`](README.md) 故障排查部分
3. 检查日志: `make logs`

### 场景 4: 我想理解代码实现
1. 阅读 [`ARCHITECTURE.md`](ARCHITECTURE.md) - 架构设计
2. 查看 [`DIAGRAMS.md`](DIAGRAMS.md) - 消息流程图
3. 阅读 `cmd/bridge/main.go` - 源代码

### 场景 5: 我想修改配置
1. 参考 [`ENV_CONFIG.md`](ENV_CONFIG.md) - 所有配置选项
2. 修改 `deployments/kubernetes/deployment.yaml`
3. 提交到 Git 并同步

### 场景 6: 我想扩展功能
1. 理解 [`ARCHITECTURE.md`](ARCHITECTURE.md) - 当前架构
2. 查看 [`PROJECT_SUMMARY.md`](PROJECT_SUMMARY.md) 扩展路径章节
3. 修改 `cmd/bridge/main.go`

---

## 📂 文件结构

```
mqtt-kafka-bridge/
│
├── 📄 文档 (7 个)
│   ├── README.md                    # 完整项目文档
│   ├── PROJECT_SUMMARY.md           # 项目交付总结 ⭐
│   ├── QUICKSTART.md                # 5 分钟快速部署 ⭐
│   ├── ARCHITECTURE.md              # 架构设计详解
│   ├── DEPLOYMENT_CHECKLIST.md      # 部署检查清单 ⭐
│   ├── ENV_CONFIG.md                # 环境变量配置
│   ├── DIAGRAMS.md                  # 流程图和架构图
│   └── INDEX.md                     # 本文档
│
├── 💻 源代码 (4 个)
│   ├── cmd/bridge/main.go           # Go 主程序 (191 行)
│   ├── go.mod                       # Go 模块定义
│   ├── go.sum                       # 依赖校验
│   └── Dockerfile                   # 容器镜像构建
│
├── 🚀 部署配置 (3 个)
│   ├── deployments/kubernetes/
│   │   ├── deployment.yaml          # Kubernetes 部署配置
│   │   └── argocd-app.yaml         # ArgoCD GitOps 配置
│   └── deploy.sh                    # 一键部署脚本
│
├── 🛠️ 工具脚本 (2 个)
│   ├── scripts/create-topics.sh     # 预创建 Kafka Topics
│   └── scripts/test-consumer.sh     # Kafka 消费测试
│
└── ⚙️ 辅助文件 (2 个)
    ├── Makefile                     # 常用命令
    └── .gitignore                   # Git 忽略规则
```

⭐ = 推荐优先阅读

---

## 🔍 关键字搜索

### 部署相关
- **快速部署**: [`QUICKSTART.md`](QUICKSTART.md)
- **检查清单**: [`DEPLOYMENT_CHECKLIST.md`](DEPLOYMENT_CHECKLIST.md)
- **一键部署**: `deploy.sh`
- **ArgoCD**: [`argocd-app.yaml`](deployments/kubernetes/argocd-app.yaml)

### 配置相关
- **环境变量**: [`ENV_CONFIG.md`](ENV_CONFIG.md)
- **MQTT 配置**: [`deployment.yaml`](deployments/kubernetes/deployment.yaml)
- **Kafka 配置**: [`deployment.yaml`](deployments/kubernetes/deployment.yaml)
- **资源限制**: [`deployment.yaml`](deployments/kubernetes/deployment.yaml)

### 架构相关
- **系统架构**: [`ARCHITECTURE.md`](ARCHITECTURE.md)
- **消息流程**: [`DIAGRAMS.md`](DIAGRAMS.md)
- **Topic 映射**: [`ARCHITECTURE.md`](ARCHITECTURE.md)
- **容错设计**: [`ARCHITECTURE.md`](ARCHITECTURE.md)

### 开发相关
- **源代码**: `cmd/bridge/main.go`
- **依赖管理**: `go.mod`
- **Docker 构建**: `Dockerfile`
- **本地开发**: [`ENV_CONFIG.md`](ENV_CONFIG.md)

### 故障排查
- **日志查看**: `make logs` 或 [`README.md`](README.md)
- **健康检查**: [`DEPLOYMENT_CHECKLIST.md`](DEPLOYMENT_CHECKLIST.md)
- **常见问题**: [`README.md`](README.md) 和 [`ENV_CONFIG.md`](ENV_CONFIG.md)
- **资源问题**: [`ARCHITECTURE.md`](ARCHITECTURE.md)

---

## 💡 推荐阅读顺序

### 快速上手 (15 分钟)
1. [`PROJECT_SUMMARY.md`](PROJECT_SUMMARY.md) (5 分钟)
2. [`QUICKSTART.md`](QUICKSTART.md) (10 分钟)

### 深入理解 (1 小时)
1. [`ARCHITECTURE.md`](ARCHITECTURE.md) (30 分钟)
2. [`DIAGRAMS.md`](DIAGRAMS.md) (15 分钟)
3. `cmd/bridge/main.go` (15 分钟)

### 生产部署 (30 分钟)
1. [`DEPLOYMENT_CHECKLIST.md`](DEPLOYMENT_CHECKLIST.md) (10 分钟)
2. [`ENV_CONFIG.md`](ENV_CONFIG.md) (10 分钟)
3. [`QUICKSTART.md`](QUICKSTART.md) (10 分钟)

---

## 📞 快速命令

```bash
# 查看所有文档
ls -la *.md

# 搜索关键字
grep -r "MQTT" *.md

# 查看项目统计
make help

# 部署应用
./deploy.sh

# 查看日志
make logs

# 查看状态
make status
```

---

## 🆘 需要帮助？

根据你的问题，查找对应文档：

| 问题类型 | 查看文档 |
|---------|---------|
| 不知道如何开始 | [`PROJECT_SUMMARY.md`](PROJECT_SUMMARY.md) |
| 想快速部署 | [`QUICKSTART.md`](QUICKSTART.md) |
| 部署遇到问题 | [`DEPLOYMENT_CHECKLIST.md`](DEPLOYMENT_CHECKLIST.md) |
| 想修改配置 | [`ENV_CONFIG.md`](ENV_CONFIG.md) |
| 想理解实现 | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| 想查看流程图 | [`DIAGRAMS.md`](DIAGRAMS.md) |
| 其他问题 | [`README.md`](README.md) |

---

## 📊 文档大小

| 文档 | 大小 | 内容 |
|------|------|------|
| `PROJECT_SUMMARY.md` | 8.1K | 项目总结 |
| `ARCHITECTURE.md` | 14K | 架构详解 |
| `DEPLOYMENT_CHECKLIST.md` | 6.9K | 部署清单 |
| `ENV_CONFIG.md` | 6.8K | 配置说明 |
| `DIAGRAMS.md` | 5.2K | 流程图 |
| `QUICKSTART.md` | 5.0K | 快速开始 |
| `README.md` | 4.5K | 项目文档 |

**总计**: ~50K 文档内容

---

**更新时间**: 2025-11-27  
**项目版本**: v1.0.0  
**维护者**: Claude
