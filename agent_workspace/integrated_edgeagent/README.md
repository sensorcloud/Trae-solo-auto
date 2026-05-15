# EdgeAgent-Hub

一个分布式 AI Agent 平台，用于计算资源和电力资源的智能调度。

## 功能特性

- **AI Agent 运行时**: 安全的沙箱式 Agent 执行，支持多运行时
- **计算资源市场**: 资源发布、订单管理、竞价系统
- **能源管理**: 电力监控、存储调度、VPP 管理
- **算力协同调度**: 负载预测、多目标优化
- **IoT 服务**: 设备管理、遥测采集、协议适配
- **用户管理**: 身份认证、授权、账单管理
- **监控与告警**: 指标采集、告警规则

## 快速开始

### 前置要求

- Go 1.21+
- PostgreSQL 或 SQLite
- Docker (可选)

### 本地开发

```bash
# 克隆仓库
git clone https://gitcode.com/ywtech/EdgeAgent-Hub.git
cd EdgeAgent-Hub

# 安装依赖
go mod download

# 构建
go build -o edgeagent ./cmd/edgeagent

# 运行
./edgeagent --config config/config-dev.yaml
```

### Docker 部署

```bash
docker-compose -f deploy/docker/docker-compose.yml up
```

### Kubernetes 部署

```bash
helm install edgeagent-hub deploy/helm/edgeagent-hub
```

## API 端点

| 模块 | 基础路径 | 描述 |
|------|----------|------|
| 认证 | `/api/v1/auth` | 用户注册、登录 |
| Agents | `/api/v1/agents` | Agent 管理 |
| 资源 | `/api/v1/assets` | 计算资源列表 |
| 订单 | `/api/v1/orders` | 订单管理 |
| 电力 | `/api/v1/power` | 电力源管理 |
| 存储 | `/api/v1/storage` | 能源存储管理 |
| 调度 | `/api/v1/schedule` | 协同调度 |
| 设备 | `/api/v1/devices` | IoT 设备管理 |
| 账单 | `/api/v1/bills` | 账单管理 |
| 指标 | `/api/v1/metrics` | 系统指标 |

## 架构

平台采用分层架构：

- **用户层**: Web 控制台、CLI、SDK、REST/gRPC API
- **编排层**: API 网关、身份认证、工作流引擎
- **能力层**: Agent 运行时、市场、能源、协同、IoT
- **基础设施层**: Kubernetes、PostgreSQL、Kafka、Redis

## 许可证

MIT
