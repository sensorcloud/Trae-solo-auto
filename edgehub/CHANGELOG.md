# 变更日志

所有重要的项目变更都将记录在此文件中。

## [1.1.0] - 2026-05-12

### 架构升级版本

本版本根据架构分析报告（7.2 架构差距）进行了全面的功能增强，填补了多项架构空白。

#### 新增功能

##### 服务网格集成
- **Linkerd/mTLS 支持**
  - 服务网格配置管理
  - 双向 TLS 证书管理
  - 安全的微服务通信
  - 支持 Linkerd 和 Istio 两种网格类型

##### 多集群联邦管理
- **Karmada 集成**
  - 集群注册与发现
  - 跨集群工作负载分发
  - 多集群服务同步
  - 联邦集群健康监控
  - 统一的集群视图

##### 批处理调度增强
- **Kueue/Volcano 集成**
  - 多队列优先级管理
  - Gang 调度（All-or-Nothing）
  - 公平调度支持
  - 作业重试机制
  - 批量任务优化调度

##### GPU 虚拟化支持
- **HAMi 集成**
  - GPU 资源抽象层
  - vGPU 虚拟化支持
  - GPU 内存管理
  - 多实例 GPU 共享
  - GPU 调度优化

##### Web UI 控制台
- **React 前端应用**
  - 仪表盘（集群/节点/Pod 概览）
  - 集群管理界面
  - 工作负载监控
  - 批处理任务管理
  - GPU 资源监控
  - 系统设置面板
  - JWT 认证登录

#### 架构改进

| 组件 | 改进项 | 说明 |
|------|--------|------|
| 服务网格 | Linkerd mTLS | 微服务安全通信 |
| 多集群 | Karmada | 跨集群联邦管理 |
| 调度器 | Kueue/Volcano | 企业级批处理 |
| GPU | HAMi | 虚拟化 GPU 资源 |
| 前端 | React Console | Web 控制台 |

#### 代码质量

- 修复类型不匹配错误
- 移除未使用的导入
- 修复 TLS 配置字段错误
- 所有模块编译通过
- 单元测试通过

#### 升级指南

##### API Server
```yaml
# config.yaml
service_mesh:
  enabled: true
  type: linkerd
  mTLS_enabled: true
```

##### Web Console
```bash
cd web
npm install
npm run dev
```

##### GPU 节点配置
```yaml
# 添加 HAMi 调度器
spec:
  schedulerName: hamischeduler
```

---

## [1.0.0] - 2026-05-12

### 首次发布 (MVP)

这是 EdgeHub 边缘算力集群聚合平台的首个正式版本。

#### 新增功能

##### 核心服务
- **API Server**: 基于 Gin 框架的高性能 REST API 服务
  - 健康检查端点 `/health`
  - Prometheus 指标端点 `/metrics`
  - JWT 认证与授权
  - CORS 跨域支持
  - 请求追踪与日志

- **Scheduler**: Kubernetes 任务调度引擎
  - 多队列任务管理
  - 拓扑感知节点选择
  - 资源匹配与评分算法
  - 孤儿 Pod 自动清理

- **Node Agent**: 边缘节点代理
  - Kubernetes 客户端集成
  - 节点信息收集（硬件/网络/标签）
  - 心跳上报机制
  - Pod 监控与事件处理

- **CLI**: 命令行工具
  - 节点管理命令 (`edge node`)
  - 任务管理命令 (`edge job`)
  - 算力市场命令 (`edge market`)
  - 集群管理命令 (`edge cluster`)

##### API 端点

| 模块 | 端点 | 方法 | 描述 |
|------|------|------|------|
| 认证 | `/api/v1/auth/login` | POST | 用户登录 |
| 认证 | `/api/v1/auth/register` | POST | 用户注册 |
| 节点 | `/api/v1/nodes` | GET/POST | 节点列表/注册 |
| 节点 | `/api/v1/nodes/:id` | GET/PUT/DELETE | 节点操作 |
| 任务 | `/api/v1/jobs` | GET/POST | 任务列表/提交 |
| 任务 | `/api/v1/jobs/:id` | GET/PUT/DELETE | 任务操作 |
| 市场 | `/api/v1/market/offers` | GET/POST | 算力挂单 |
| 市场 | `/api/v1/market/orders` | GET/POST | 订单管理 |
| 市场 | `/api/v1/market/prices` | GET | 价格查询 |
| 计费 | `/api/v1/billing/bills` | GET | 账单管理 |

##### 数据模型
- 用户与租户管理
- 集群与节点管理
- 任务与工作负载
- 算力市场（挂单/订单）
- 计费与账单
- 监控指标与告警
- 性能评测基准

##### 部署配置
- Kubernetes 部署清单
- Helm Chart 包
- Docker 多阶段构建
- GitHub Actions CI/CD

#### 安全改进
- 密码 bcrypt 加密存储
- JWT Token 认证
- RBAC 权限控制中间件
- CORS 安全配置

#### 技术栈
- Go 1.21+
- Kubernetes 1.28+
- PostgreSQL 16
- Redis 7
- Prometheus + Grafana
- Gin Web 框架
- GORM 数据库 ORM

#### 文档
- 架构设计文档
- API 参考文档
- 部署指南
- CLI 使用手册

---

*格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)*
