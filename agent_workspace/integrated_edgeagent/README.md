# EdgeAgent-Hub

一个分布式 AI Agent 平台，用于计算资源和电力资源的智能调度。

## 功能特性

- **AI Agent 运行时**: 安全的沙箱式 Agent 执行，支持多运行时
- **计算资源市场**: 资源发布、订单管理、竞价系统
- **能源管理**: 电力监控、储能调度、虚拟电厂(VPP)
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

## 前端界面

项目包含现代化的 Web 管理界面，位于 `frontend/` 目录：

- **index.html** - 主仪表盘，展示系统概览
- **power-management.html** - 电源和能源管理
- **market.html** - 算力市场资源浏览
- **agents.html** - AI Agent 管理

### 启动前端

前端为纯静态 HTML 文件，可直接用浏览器打开，或使用任意 Web 服务器：

```bash
# 使用 Python 启动简单服务器
cd frontend
python3 -m http.server 8080

# 或使用 Node.js
npx serve .
```

然后访问 http://localhost:8080

## API 端点

| 模块 | 基础路径 | 描述 |
|------|----------|------|
| 认证 | `/api/v1/auth` | 用户注册、登录 |
| Agents | `/api/v1/agents` | Agent 管理 |
| 资源 | `/api/v1/assets` | 计算资源列表 |
| 订单 | `/api/v1/orders` | 订单管理 |
| 电力 | `/api/v1/energy/power` | 电源管理 |
| 储能 | `/api/v1/energy/storage` | 储能设备管理 |
| VPP | `/api/v1/energy/vpp` | 虚拟电厂管理 |
| 调度 | `/api/v1/schedule` | 协同调度 |
| 设备 | `/api/v1/devices` | IoT 设备管理 |
| 账单 | `/api/v1/bills` | 账单管理 |
| 指标 | `/api/v1/metrics` | 系统指标 |

## 架构

平台采用分层架构：

- **用户层**: Web 控制台、CLI工具、SDK、REST/gRPC API
- **编排层**: API 网关、身份认证、工作流引擎
- **能力层**: Agent 运行时、市场、能源、协同、IoT
- **基础设施层**: Kubernetes、PostgreSQL、Kafka、Redis

## 技术栈

**后端**
- Go 1.21+
- Gin Web Framework
- GORM ORM
- JWT Authentication

**前端**
- HTML5 + CSS3
- Vanilla JavaScript
- 响应式设计

**部署**
- Docker & Docker Compose
- Kubernetes & Helm
- PostgreSQL/SQLite

## 项目结构

```
.
├── cmd/
│   └── edgeagent/          # 主程序入口
├── internal/
│   ├── agent/              # AI Agent 模块
│   ├── market/             # 算力市场模块
│   ├── energy/             # 能源管理模块
│   ├── coordination/       # 算电协同模块
│   ├── iot/                # IoT 连接模块
│   ├── billing/            # 计费结算模块
│   ├── monitor/            # 监控告警模块
│   └── user/               # 用户认证模块
├── pkg/
│   └── database/           # 数据库配置
├── frontend/               # Web 管理界面
│   ├── index.html         # 主仪表盘
│   ├── power-management.html # 电源管理
│   ├── market.html        # 算力市场
│   └── agents.html        # Agent 管理
├── deploy/
│   ├── docker/            # Docker 部署
│   ├── k8s/               # Kubernetes 部署
│   └── helm/              # Helm Chart
├── config/                # 配置文件
└── README.md
```

## 配置说明

配置文件位于 `config/` 目录：

- `config.yaml` - 生产环境配置
- `config-dev.yaml` - 开发环境配置

主要配置项：

```yaml
server:
  host: 0.0.0.0
  port: 8080

database:
  type: postgres  # 或 sqlite
  dsn: "host=localhost user=root password=123456 dbname=edgeagent port=5432 sslmode=disable"

jwt:
  secret: your-secret-key
  expiration: 24h
```

## 开发指南

### 添加新模块

1. 在 `internal/` 下创建模块目录
2. 实现 `models.go`、`handlers.go`、`routes.go`
3. 在 `pkg/database/migrate.go` 中注册模型
4. 在 `cmd/edgeagent/main.go` 中注册路由

### API 开发规范

- 使用 Gin 框架
- 统一响应格式：`{"code": 0, "message": "success", "data": {...}}`
- 添加 JWT 认证中间件
- 使用 validator 进行参数校验

## 许可证

MIT
