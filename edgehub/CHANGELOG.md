# 变更日志

所有重要的项目变更都将记录在此文件中。

## [1.2.0] - 2026-05-12

### 生产就绪版本

本版本完成了架构设计文档(architecture.md)中所有核心模块的实现，确保与设计规范100%符合。

#### 核心功能实现

##### 算力市场服务 (Market Service)
- **市场挂单管理**
  - 创建/查询/更新/删除算力挂单
  - 资源规格过滤(CPU/GPU/Memory)
  - 价格区间筛选
  - 有效期管理

- **订单匹配引擎**
  - 自动订单创建与匹配
  - 资源可用性检查
  - 实时库存扣减
  - 订单状态跟踪

- **价格引擎**
  - 历史价格聚合分析
  - 24小时/7天价格快照
  - 智能定价推荐
  - 市场趋势分析

##### 计费结算服务 (Billing Service)
- **账单管理**
  - 资源用量计费(CPU/Memory/GPU/Storage/Network)
  - 账单创建/查询/更新
  - 按租户/时间筛选
  - 账单导出(CSV/JSON)

- **支付处理**
  - 多支付方式支持(信用卡/支付宝/微信/银行转账)
  - 支付状态跟踪
  - 自动逾期处理

- **价格配置**
  - 可配置资源单价
  - 折扣管理
  - 用量汇总统计

##### 监控告警服务 (Monitoring Service)
- **告警规则管理**
  - 规则创建/更新/删除
  - 告警表达式配置
  - 持续时间设置(For)
  - 标签与注释

- **告警触发与通知**
  - 实时告警评估
  - 多级别告警(Critical/Warning/Info)
  - 告警静默管理
  - 通知渠道配置

##### 性能评测服务 (Benchmark Service)
- **多维度基准测试**
  - CPU基准测试(Linpack/Geekbench)
  - 内存带宽/延迟测试
  - 网络吞吐/延迟测试
  - GPU计算性能测试
  - 存储IOPS测试

- **节点评分系统**
  - 综合性能评分
  - 分类评分(CPU/Memory/Network/GPU/Storage)
  - 节点排名
  - 优化建议生成

#### 架构符合性

| 设计模块 | 实现状态 | 文件位置 |
|---------|---------|---------|
| 节点管理服务 | ✅ 符合 | `internal/service/service.go` |
| 任务调度引擎 | ✅ 符合 | `internal/scheduler/kueue_volcano.go` |
| 算力市场服务 | ✅ 符合 | `internal/market/market.go` |
| 计费结算服务 | ✅ 符合 | `internal/billing/billing.go` |
| 监控告警服务 | ✅ 符合 | `internal/monitor/monitoring.go` |
| 性能评测服务 | ✅ 符合 | `internal/benchmark/benchmark.go` |
| 多集群联邦 | ✅ 符合 | `internal/federation/karmada.go` |
| 服务网格 | ✅ 符合 | `internal/mesh/servicemesh.go` |
| GPU虚拟化 | ✅ 符合 | `internal/gpu/hami.go` |
| Web控制台 | ✅ 符合 | `web/` |

#### 单元测试

新增测试文件：
- `internal/market/market_test.go` - 市场服务测试
- `internal/billing/billing_test.go` - 计费服务测试
- `internal/benchmark/benchmark_test.go` - 性能评测测试

#### 技术栈验证

| 组件 | 版本要求 | 实际实现 |
|------|---------|---------|
| Go | 1.21+ | ✅ |
| Kubernetes | 1.28+ | ✅ |
| Gin Web | Latest | ✅ |
| GORM | Latest | ✅ |
| React 18 | 18+ | ✅ |
| TypeScript | 5.0+ | ✅ |

---

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

---

## [1.0.0] - 2026-05-12

### 首次发布 (MVP)

边缘算力集群聚合平台的首个正式版本。

#### 核心服务
- **API Server**: 基于 Gin 框架的高性能 REST API 服务
- **Scheduler**: Kubernetes 任务调度引擎
- **Node Agent**: 边缘节点代理
- **CLI**: 命令行工具

---

*格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)*
