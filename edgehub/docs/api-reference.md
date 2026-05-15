# EdgeHub API 参考文档

## 版本：V1.0
## 日期：2026年5月

---

## 1. 概述

EdgeHub API 提供了一套完整的 RESTful 接口，用于管理能源、算力、智能体和 IoT 设备等资源。

### 1.1 基础信息

| 项目 | 说明 |
|------|------|
| **Base URL** | `https://api.edgehub.io/api/v1` |
| **协议** | HTTPS |
| **数据格式** | JSON |
| **字符编码** | UTF-8 |
| **API版本** | v1 |

### 1.2 通用响应格式

所有API响应采用统一的JSON格式：

```json
{
  "code": 0,
  "message": "success",
  "data": { },
  "timestamp": 1715769600000
}
```

### 1.3 错误码定义

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 认证失败 |
| 1003 | 权限不足 |
| 1004 | 资源不存在 |
| 1005 | 资源已存在 |
| 2001 | 内部错误 |
| 2002 | 服务不可用 |
| 2003 | 请求超时 |

---

## 2. 认证说明

### 2.1 认证方式

EdgeHub API 支持 JWT (JSON Web Token) 认证。

### 2.2 获取Token

**请求**

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "your_password"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 86400
  },
  "timestamp": 1715769600000
}
```

### 2.3 使用Token

在请求头中添加 Authorization 字段：

```http
GET /api/v1/nodes
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### 2.4 刷新Token

**请求**

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 86400
  },
  "timestamp": 1715769600000
}
```

### 2.5 注销登录

**请求**

```http
POST /api/v1/auth/logout
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": null,
  "timestamp": 1715769600000
}
```

---

## 3. 认证接口 (Auth)

### 3.1 用户注册

**请求**

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "your_password",
  "name": "用户名",
  "phone": "13800138000"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "name": "用户名",
    "created_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

---

## 4. 节点接口 (Nodes)

### 4.1 获取节点列表

**请求**

```http
GET /api/v1/nodes?page=1&page_size=10&status=online
Authorization: Bearer {token}
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页数量，默认10 |
| status | string | 否 | 节点状态：online/offline/busy |
| region | string | 否 | 区域筛选 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 100,
    "page": 1,
    "page_size": 10,
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "node-001",
        "status": "online",
        "region": "ap-east-1",
        "capacity": {
          "cpu": "64",
          "memory": "256Gi",
          "gpu": "8",
          "gpu_type": "NVIDIA A100"
        },
        "allocatable": {
          "cpu": "60",
          "memory": "240Gi",
          "gpu": "7"
        },
        "labels": {
          "node-type": "gpu",
          "gpu-vendor": "nvidia"
        },
        "created_at": "2026-05-15T10:00:00Z",
        "updated_at": "2026-05-15T10:00:00Z"
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 4.2 注册节点

**请求**

```http
POST /api/v1/nodes
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "node-001",
  "cluster_id": "550e8400-e29b-41d4-a716-446655440001",
  "region": "ap-east-1",
  "capacity": {
    "cpu": "64",
    "memory": "256Gi",
    "gpu": "8",
    "gpu_type": "NVIDIA A100"
  },
  "labels": {
    "node-type": "gpu",
    "gpu-vendor": "nvidia"
  }
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "node-001",
    "status": "pending",
    "created_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 4.3 获取节点详情

**请求**

```http
GET /api/v1/nodes/{node_id}
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "node-001",
    "cluster_id": "550e8400-e29b-41d4-a716-446655440001",
    "status": "online",
    "region": "ap-east-1",
    "capacity": {
      "cpu": "64",
      "memory": "256Gi",
      "gpu": "8",
      "gpu_type": "NVIDIA A100"
    },
    "allocatable": {
      "cpu": "60",
      "memory": "240Gi",
      "gpu": "7"
    },
    "labels": {
      "node-type": "gpu",
      "gpu-vendor": "nvidia"
    },
    "annotations": {},
    "created_at": "2026-05-15T10:00:00Z",
    "updated_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 4.4 更新节点

**请求**

```http
PUT /api/v1/nodes/{node_id}
Authorization: Bearer {token}
Content-Type: application/json

{
  "labels": {
    "node-type": "gpu",
    "gpu-vendor": "nvidia",
    "environment": "production"
  }
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "updated_at": "2026-05-15T10:30:00Z"
  },
  "timestamp": 1715769600000
}
```

### 4.5 删除节点

**请求**

```http
DELETE /api/v1/nodes/{node_id}
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": null,
  "timestamp": 1715769600000
}
```

### 4.6 节点心跳

**请求**

```http
POST /api/v1/nodes/{node_id}/heartbeat
Authorization: Bearer {token}
Content-Type: application/json

{
  "status": "online",
  "metrics": {
    "cpu_usage": 45.5,
    "memory_usage": 60.2,
    "gpu_usage": 80.0
  }
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "acknowledged": true,
    "timestamp": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 4.7 获取节点指标

**请求**

```http
GET /api/v1/nodes/{node_id}/metrics?period=1h
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "node_id": "550e8400-e29b-41d4-a716-446655440000",
    "period": "1h",
    "metrics": {
      "cpu_usage_avg": 45.5,
      "cpu_usage_max": 80.2,
      "memory_usage_avg": 60.2,
      "memory_usage_max": 75.0,
      "gpu_usage_avg": 80.0,
      "gpu_usage_max": 95.0,
      "network_in_bytes": 1073741824,
      "network_out_bytes": 536870912
    }
  },
  "timestamp": 1715769600000
}
```

---

## 5. 任务接口 (Jobs)

### 5.1 获取任务列表

**请求**

```http
GET /api/v1/jobs?page=1&page_size=10&status=running
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 50,
    "page": 1,
    "page_size": 10,
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440002",
        "name": "training-job-001",
        "type": "training",
        "status": "running",
        "cluster_id": "550e8400-e29b-41d4-a716-446655440001",
        "node_id": "550e8400-e29b-41d4-a716-446655440000",
        "priority": "high",
        "resources": {
          "cpu": "16",
          "memory": "64Gi",
          "gpu": "4"
        },
        "created_at": "2026-05-15T08:00:00Z",
        "started_at": "2026-05-15T08:05:00Z"
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 5.2 提交任务

**请求**

```http
POST /api/v1/jobs
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "training-job-001",
  "type": "training",
  "cluster_id": "550e8400-e29b-41d4-a716-446655440001",
  "priority": "high",
  "resources": {
    "cpu": "16",
    "memory": "64Gi",
    "gpu": "4"
  },
  "spec": {
    "image": "pytorch/pytorch:2.0",
    "command": ["python", "train.py"],
    "args": ["--epochs", "100"],
    "env": {
      "CUDA_VISIBLE_DEVICES": "0,1,2,3"
    },
    "volumes": [
      {
        "name": "data",
        "mount_path": "/data",
        "size": "100Gi"
      }
    ]
  },
  "time_constraint": {
    "deadline": "2026-05-16T08:00:00Z",
    "interruptible": true
  },
  "energy_preference": {
    "max_price_per_kwh": 0.5,
    "min_green_ratio": 0.6
  }
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440002",
    "name": "training-job-001",
    "status": "pending",
    "created_at": "2026-05-15T08:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 5.3 获取任务详情

**请求**

```http
GET /api/v1/jobs/{job_id}
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440002",
    "name": "training-job-001",
    "type": "training",
    "status": "running",
    "cluster_id": "550e8400-e29b-41d4-a716-446655440001",
    "node_id": "550e8400-e29b-41d4-a716-446655440000",
    "priority": "high",
    "resources": {
      "cpu": "16",
      "memory": "64Gi",
      "gpu": "4"
    },
    "spec": {
      "image": "pytorch/pytorch:2.0",
      "command": ["python", "train.py"]
    },
    "energy_cost": 125.50,
    "carbon_emission": 45.2,
    "green_ratio": 0.65,
    "created_at": "2026-05-15T08:00:00Z",
    "started_at": "2026-05-15T08:05:00Z",
    "updated_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 5.4 停止任务

**请求**

```http
POST /api/v1/jobs/{job_id}/stop
Authorization: Bearer {token}
Content-Type: application/json

{
  "reason": "用户主动停止"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440002",
    "status": "stopping",
    "updated_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 5.5 获取任务日志

**请求**

```http
GET /api/v1/jobs/{job_id}/logs?tail=100
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "job_id": "550e8400-e29b-41d4-a716-446655440002",
    "logs": [
      {
        "timestamp": "2026-05-15T08:05:00Z",
        "level": "INFO",
        "message": "Starting training..."
      },
      {
        "timestamp": "2026-05-15T08:10:00Z",
        "level": "INFO",
        "message": "Epoch 1/100 completed, loss: 0.5"
      }
    ]
  },
  "timestamp": 1715769600000
}
```

---

## 6. 能源接口 (Energy)

### 6.1 获取电源列表

**请求**

```http
GET /api/v1/energy/power-sources
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440003",
        "name": "solar-farm-001",
        "type": "solar",
        "status": "online",
        "capacity": 1000.0,
        "current_output": 750.5,
        "region": "ap-east-1",
        "green_ratio": 1.0,
        "carbon_intensity": 0.0,
        "created_at": "2026-05-15T10:00:00Z"
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 6.2 创建电源

**请求**

```http
POST /api/v1/energy/power-sources
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "solar-farm-001",
  "type": "solar",
  "capacity": 1000.0,
  "region": "ap-east-1",
  "location": {
    "latitude": 22.5,
    "longitude": 114.0
  },
  "config": {
    "panel_type": "monocrystalline",
    "efficiency": 0.22
  }
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440003",
    "name": "solar-farm-001",
    "status": "pending",
    "created_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 6.3 获取储能设备列表

**请求**

```http
GET /api/v1/storage/devices
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440004",
        "name": "battery-storage-001",
        "type": "lithium_ion",
        "status": "idle",
        "capacity": 500.0,
        "soc": 75.0,
        "current_power": 0.0,
        "max_charge_rate": 100.0,
        "max_discharge_rate": 100.0,
        "strategy": "peak_valley_arbitrage",
        "created_at": "2026-05-15T10:00:00Z"
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 6.4 储能充电

**请求**

```http
POST /api/v1/storage/devices/{id}/charge
Authorization: Bearer {token}
Content-Type: application/json

{
  "power": 50.0,
  "duration": 60
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "device_id": "550e8400-e29b-41d4-a716-446655440004",
    "status": "charging",
    "power": 50.0,
    "started_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 6.5 储能放电

**请求**

```http
POST /api/v1/storage/devices/{id}/discharge
Authorization: Bearer {token}
Content-Type: application/json

{
  "power": 80.0,
  "duration": 30
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "device_id": "550e8400-e29b-41d4-a716-446655440004",
    "status": "discharging",
    "power": 80.0,
    "started_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 6.6 获取能源市场概览

**请求**

```http
GET /api/v1/energy/market/overview?region=ap-east-1
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "region": "ap-east-1",
    "current_price": 0.45,
    "price_change": 0.05,
    "trading_volume": 10000.0,
    "green_ratio": 0.65,
    "peak_price": 0.85,
    "valley_price": 0.25,
    "active_orders": 150,
    "available_power": 5000.0,
    "updated_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

---

## 7. 交易接口 (Trading)

### 7.1 创建能源订单

**请求**

```http
POST /api/v1/trading/orders
Authorization: Bearer {token}
Content-Type: application/json

{
  "type": "buy",
  "energy_type": "green",
  "quantity": 1000.0,
  "price": 0.45,
  "region": "ap-east-1",
  "valid_until": "2026-05-15T12:00:00Z"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440005",
    "order_no": "ORD20260515001",
    "type": "buy",
    "energy_type": "green",
    "quantity": 1000.0,
    "price": 0.45,
    "status": "pending",
    "created_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 7.2 获取价格行情

**请求**

```http
GET /api/v1/trading/prices?region=ap-east-1&energy_type=green
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "region": "ap-east-1",
    "energy_type": "green",
    "current_price": 0.45,
    "prices": [
      {
        "timestamp": "2026-05-15T10:00:00Z",
        "price": 0.45
      },
      {
        "timestamp": "2026-05-15T09:00:00Z",
        "price": 0.42
      }
    ],
    "forecast": [
      {
        "timestamp": "2026-05-15T11:00:00Z",
        "price": 0.48,
        "confidence": 0.85
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 7.3 获取绿证列表

**请求**

```http
GET /api/v1/trading/green-certificates
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440006",
        "source_type": "solar",
        "owner_id": "550e8400-e29b-41d4-a716-446655440000",
        "status": "active",
        "energy_amount": 1000.0,
        "valid_from": "2026-01-01T00:00:00Z",
        "valid_until": "2026-12-31T23:59:59Z",
        "created_at": "2026-01-01T00:00:00Z"
      }
    ]
  },
  "timestamp": 1715769600000
}
```

---

## 8. 虚拟电厂接口 (VPP)

### 8.1 获取VPP列表

**请求**

```http
GET /api/v1/vpp
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440007",
        "name": "vpp-001",
        "type": "commercial",
        "status": "online",
        "total_capacity": 5000.0,
        "available_capacity": 3500.0,
        "power_sources_count": 10,
        "storage_devices_count": 5,
        "control_strategy": "price_optimized",
        "created_at": "2026-05-15T10:00:00Z"
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 8.2 创建VPP

**请求**

```http
POST /api/v1/vpp
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "vpp-001",
  "type": "commercial",
  "region": "ap-east-1",
  "control_strategy": "price_optimized"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440007",
    "name": "vpp-001",
    "status": "created",
    "created_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 8.3 VPP调度

**请求**

```http
POST /api/v1/vpp/{id}/dispatch
Authorization: Bearer {token}
Content-Type: application/json

{
  "power": 500.0,
  "duration": 60,
  "priority": 1,
  "response_type": "peak_shaving",
  "reason": "电网调峰需求"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "request_id": "550e8400-e29b-41d4-a716-446655440008",
    "vpp_id": "550e8400-e29b-41d4-a716-446655440007",
    "dispatched_power": 480.0,
    "actual_power": 475.0,
    "start_time": "2026-05-15T10:00:00Z",
    "end_time": "2026-05-15T11:00:00Z",
    "status": "dispatched"
  },
  "timestamp": 1715769600000
}
```

---

## 9. IoT接口

### 9.1 获取设备列表

**请求**

```http
GET /api/v1/iot/devices?page=1&page_size=10&status=online
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 500,
    "page": 1,
    "page_size": 10,
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440009",
        "name": "temperature-sensor-001",
        "description": "温度传感器",
        "profile_id": "550e8400-e29b-41d4-a716-446655440010",
        "protocol": "mqtt",
        "status": "online",
        "labels": {
          "location": "room-101",
          "type": "sensor"
        },
        "last_online_at": "2026-05-15T10:00:00Z",
        "created_at": "2026-05-01T00:00:00Z"
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 9.2 创建设备

**请求**

```http
POST /api/v1/iot/devices
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "temperature-sensor-001",
  "description": "温度传感器",
  "profile_id": "550e8400-e29b-41d4-a716-446655440010",
  "protocol": "mqtt",
  "connection_info": {
    "topic_prefix": "devices/temp-sensor-001",
    "qos": 1
  },
  "labels": {
    "location": "room-101",
    "type": "sensor"
  }
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440009",
    "name": "temperature-sensor-001",
    "status": "offline",
    "created_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 9.3 获取设备影子

**请求**

```http
GET /api/v1/iot/devices/{device_id}/shadow
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "device_id": "550e8400-e29b-41d4-a716-446655440009",
    "reported": {
      "temperature": 25.5,
      "humidity": 60.0,
      "battery_level": 85
    },
    "desired": {
      "report_interval": 60
    },
    "delta": {},
    "metadata": {
      "reported": {
        "temperature": {
          "timestamp": "2026-05-15T10:00:00Z"
        }
      }
    },
    "updated_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 9.4 更新设备影子

**请求**

```http
PUT /api/v1/iot/devices/{device_id}/shadow
Authorization: Bearer {token}
Content-Type: application/json

{
  "desired": {
    "report_interval": 30,
    "alert_threshold": 30.0
  }
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "device_id": "550e8400-e29b-41d4-a716-446655440009",
    "desired": {
      "report_interval": 30,
      "alert_threshold": 30.0
    },
    "updated_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 9.5 获取遥测数据

**请求**

```http
GET /api/v1/iot/telemetry/{device_id}/history?property=temperature&start=2026-05-14T00:00:00Z&end=2026-05-15T00:00:00Z
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "device_id": "550e8400-e29b-41d4-a716-446655440009",
    "property": "temperature",
    "data_points": [
      {
        "timestamp": "2026-05-14T00:00:00Z",
        "value": 24.5,
        "quality": "good"
      },
      {
        "timestamp": "2026-05-14T01:00:00Z",
        "value": 24.8,
        "quality": "good"
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 9.6 执行设备命令

**请求**

```http
POST /api/v1/iot/commands
Authorization: Bearer {token}
Content-Type: application/json

{
  "device_id": "550e8400-e29b-41d4-a716-446655440009",
  "command_name": "set_report_interval",
  "parameters": {
    "interval": 30
  },
  "timeout": 30,
  "async": false
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "correlation_id": "550e8400-e29b-41d4-a716-446655440011",
    "device_id": "550e8400-e29b-41d4-a716-446655440009",
    "command_name": "set_report_interval",
    "status": "success",
    "result": {
      "previous_interval": 60,
      "new_interval": 30
    },
    "timestamp": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

---

## 10. 智能体接口 (Agents)

### 10.1 获取沙箱列表

**请求**

```http
GET /api/v1/agents/sandboxes
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440012",
        "name": "sandbox-001",
        "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
        "runtime": "python",
        "status": "running",
        "resources": {
          "cpu_limit": "4",
          "memory_limit": "8Gi"
        },
        "agents_count": 2,
        "created_at": "2026-05-15T10:00:00Z"
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 10.2 创建沙箱

**请求**

```http
POST /api/v1/agents/sandboxes
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "sandbox-001",
  "runtime": "python",
  "resources": {
    "cpu_limit": "4",
    "memory_limit": "8Gi"
  },
  "network": {
    "enabled": true,
    "allowed_hosts": ["api.openai.com"]
  },
  "security": {
    "allow_file_write": true,
    "allow_network": true
  }
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440012",
    "name": "sandbox-001",
    "status": "creating",
    "created_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 10.3 执行代码

**请求**

```http
POST /api/v1/agents/{agent_id}/execute/code
Authorization: Bearer {token}
Content-Type: application/json

{
  "code": "print('Hello, World!')\nfor i in range(5):\n    print(i)",
  "language": "python",
  "timeout": 60
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "execution_id": "550e8400-e29b-41d4-a716-446655440013",
    "status": "completed",
    "output": "Hello, World!\n0\n1\n2\n3\n4\n",
    "error": null,
    "exit_code": 0,
    "duration": 150,
    "metrics": {
      "cpu_usage": 5.2,
      "memory_usage": 25.5
    }
  },
  "timestamp": 1715769600000
}
```

### 10.4 执行Shell命令

**请求**

```http
POST /api/v1/agents/{agent_id}/execute/shell
Authorization: Bearer {token}
Content-Type: application/json

{
  "command": "ls",
  "args": ["-la"],
  "timeout": 30
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "execution_id": "550e8400-e29b-41d4-a716-446655440014",
    "status": "completed",
    "output": "total 8\ndrwxr-xr-x 2 root root 4096 May 15 10:00 .\ndrwxr-xr-x 3 root root 4096 May 15 10:00 ..\n",
    "error": null,
    "exit_code": 0,
    "duration": 50
  },
  "timestamp": 1715769600000
}
```

### 10.5 获取沙箱统计

**请求**

```http
GET /api/v1/agents/statistics
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_sandboxes": 10,
    "running_sandboxes": 8,
    "paused_sandboxes": 1,
    "stopped_sandboxes": 1,
    "total_agents": 25,
    "running_agents": 20,
    "total_executions": 1000,
    "running_executions": 5,
    "completed_executions": 950,
    "failed_executions": 45,
    "runtime_status": {
      "python": "available",
      "javascript": "available",
      "wasm": "available"
    }
  },
  "timestamp": 1715769600000
}
```

---

## 11. 算电协同接口 (Coordination)

### 11.1 提交协同调度请求

**请求**

```http
POST /api/v1/coordination/schedule
Authorization: Bearer {token}
Content-Type: application/json

{
  "compute_job_id": "550e8400-e29b-41d4-a716-446655440002",
  "estimated_power": 100.0,
  "duration": 120,
  "preferred_start": "2026-05-15T12:00:00Z",
  "preferred_end": "2026-05-15T18:00:00Z",
  "max_energy_cost": 500.0,
  "min_green_ratio": 0.6,
  "region": "ap-east-1"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440015",
    "decision": "scheduled",
    "scheduled_start": "2026-05-15T14:00:00Z",
    "scheduled_end": "2026-05-15T16:00:00Z",
    "energy_source": "green",
    "estimated_cost": 350.0,
    "estimated_carbon": 25.5,
    "green_ratio": 0.75,
    "priority": 80,
    "reason": "最优调度时段：绿电充足，电价较低"
  },
  "timestamp": 1715769600000
}
```

### 11.2 获取最优时段

**请求**

```http
GET /api/v1/coordination/optimal-time?estimated_power=100&duration=120&region=ap-east-1
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "start_time": "2026-05-15T14:00:00Z",
    "end_time": "2026-05-15T16:00:00Z",
    "expected_cost": 350.0,
    "expected_green_ratio": 0.75,
    "confidence": 0.85,
    "reason": "预测该时段绿电充足，电价处于低谷",
    "alternatives": [
      {
        "start_time": "2026-05-15T02:00:00Z",
        "end_time": "2026-05-15T04:00:00Z",
        "expected_cost": 300.0,
        "expected_green_ratio": 0.50,
        "confidence": 0.80
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 11.3 获取能源预测

**请求**

```http
GET /api/v1/coordination/forecast?region=ap-east-1&horizon=24
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "region": "ap-east-1",
    "generated_at": "2026-05-15T10:00:00Z",
    "horizon": 24,
    "points": [
      {
        "timestamp": "2026-05-15T11:00:00Z",
        "expected_power": 5000.0,
        "expected_price": 0.45,
        "green_ratio": 0.65,
        "carbon_intensity": 150.0,
        "confidence": 0.85
      },
      {
        "timestamp": "2026-05-15T12:00:00Z",
        "expected_power": 5500.0,
        "expected_price": 0.50,
        "green_ratio": 0.70,
        "carbon_intensity": 130.0,
        "confidence": 0.82
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 11.4 获取碳排放强度

**请求**

```http
GET /api/v1/coordination/carbon-intensity?region=ap-east-1
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "region": "ap-east-1",
    "current_intensity": 150.0,
    "unit": "gCO2/kWh",
    "forecast": [
      {
        "timestamp": "2026-05-15T11:00:00Z",
        "intensity": 145.0
      },
      {
        "timestamp": "2026-05-15T12:00:00Z",
        "intensity": 130.0
      }
    ],
    "updated_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 11.5 获取绿电比例

**请求**

```http
GET /api/v1/coordination/green-ratio?region=ap-east-1
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "region": "ap-east-1",
    "current_ratio": 0.65,
    "sources": {
      "solar": 0.35,
      "wind": 0.20,
      "hydro": 0.10,
      "other_green": 0.00
    },
    "forecast": [
      {
        "timestamp": "2026-05-15T11:00:00Z",
        "ratio": 0.70
      },
      {
        "timestamp": "2026-05-15T12:00:00Z",
        "ratio": 0.75
      }
    ],
    "updated_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 11.6 获取实时状态

**请求**

```http
GET /api/v1/coordination/status
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "timestamp": "2026-05-15T10:00:00Z",
    "task_queue": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440015",
        "name": "training-job-001",
        "status": "scheduled",
        "priority": "high"
      }
    ],
    "available_energy": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440003",
        "name": "solar-farm-001",
        "type": "solar",
        "current_output": 750.5,
        "available_capacity": 249.5
      }
    ],
    "storage_status": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440004",
        "name": "battery-storage-001",
        "soc": 75.0,
        "status": "idle"
      }
    ],
    "current_price": 0.45,
    "carbon_intensity": 150.0,
    "green_ratio": 0.65
  },
  "timestamp": 1715769600000
}
```

---

## 12. 算力市场接口 (Market)

### 12.1 获取算力报价列表

**请求**

```http
GET /api/v1/market/offers?type=gpu&region=ap-east-1
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440016",
        "provider_id": "550e8400-e29b-41d4-a716-446655440000",
        "resource_spec": {
          "gpu_type": "NVIDIA A100",
          "gpu_count": 8,
          "memory": "80GB"
        },
        "price_per_unit": 10.0,
        "unit": "GPU时",
        "min_duration": 1,
        "max_duration": 720,
        "available": true,
        "valid_from": "2026-05-15T00:00:00Z",
        "valid_until": "2026-05-31T23:59:59Z",
        "region": "ap-east-1",
        "green_ratio": 0.65
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 12.2 创建算力报价

**请求**

```http
POST /api/v1/market/offers
Authorization: Bearer {token}
Content-Type: application/json

{
  "resource_spec": {
    "gpu_type": "NVIDIA A100",
    "gpu_count": 8,
    "memory": "80GB"
  },
  "price_per_unit": 10.0,
  "unit": "GPU时",
  "min_duration": 1,
  "max_duration": 720,
  "valid_from": "2026-05-15T00:00:00Z",
  "valid_until": "2026-05-31T23:59:59Z",
  "region": "ap-east-1"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440016",
    "status": "active",
    "created_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 12.3 创建算力订单

**请求**

```http
POST /api/v1/market/orders
Authorization: Bearer {token}
Content-Type: application/json

{
  "offer_id": "550e8400-e29b-41d4-a716-446655440016",
  "duration": 24,
  "quantity": 4
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440017",
    "offer_id": "550e8400-e29b-41d4-a716-446655440016",
    "type": "buy",
    "status": "pending",
    "price": 960.0,
    "created_at": "2026-05-15T10:00:00Z"
  },
  "timestamp": 1715769600000
}
```

### 12.4 获取价格推荐

**请求**

```http
GET /api/v1/market/prices/recommend?resource_type=gpu&region=ap-east-1
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "resource_type": "gpu",
    "region": "ap-east-1",
    "recommendations": [
      {
        "provider": "provider-001",
        "gpu_type": "NVIDIA A100",
        "price": 8.5,
        "green_ratio": 0.75,
        "score": 95,
        "reason": "绿电比例高，价格优惠"
      },
      {
        "provider": "provider-002",
        "gpu_type": "NVIDIA A100",
        "price": 9.0,
        "green_ratio": 0.60,
        "score": 85,
        "reason": "价格适中，可用性好"
      }
    ]
  },
  "timestamp": 1715769600000
}
```

---

## 13. 计费接口 (Billing)

### 13.1 获取账单列表

**请求**

```http
GET /api/v1/billing/bills?start_date=2026-05-01&end_date=2026-05-31
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440018",
        "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
        "period": "2026-05",
        "resource_type": "compute",
        "quantity": 1000.0,
        "unit": "GPU时",
        "unit_price": 10.0,
        "total_amount": 10000.0,
        "currency": "CNY",
        "status": "unpaid",
        "due_date": "2026-06-15T00:00:00Z",
        "created_at": "2026-05-31T23:59:59Z"
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 13.2 获取账单汇总

**请求**

```http
GET /api/v1/billing/bills/summary?period=2026-05
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "period": "2026-05",
    "total_amount": 15000.0,
    "currency": "CNY",
    "breakdown": {
      "compute": 10000.0,
      "storage": 3000.0,
      "network": 1000.0,
      "energy": 1000.0
    },
    "savings": {
      "green_energy_discount": 500.0,
      "off_peak_discount": 300.0
    },
    "carbon_saved": 150.0
  },
  "timestamp": 1715769600000
}
```

### 13.3 导出账单

**请求**

```http
GET /api/v1/billing/bills/export?period=2026-05&format=csv
Authorization: Bearer {token}
```

**响应**

返回 CSV 文件下载。

---

## 14. 监控接口 (Monitoring)

### 14.1 获取指标

**请求**

```http
GET /api/v1/monitoring/metrics?names=cpu_usage,memory_usage&period=1h
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "metrics": [
      {
        "name": "cpu_usage",
        "values": [
          {
            "timestamp": "2026-05-15T09:00:00Z",
            "value": 45.5
          },
          {
            "timestamp": "2026-05-15T09:30:00Z",
            "value": 52.3
          }
        ]
      },
      {
        "name": "memory_usage",
        "values": [
          {
            "timestamp": "2026-05-15T09:00:00Z",
            "value": 60.2
          },
          {
            "timestamp": "2026-05-15T09:30:00Z",
            "value": 62.5
          }
        ]
      }
    ]
  },
  "timestamp": 1715769600000
}
```

### 14.2 获取告警列表

**请求**

```http
GET /api/v1/monitoring/alerts?status=active
Authorization: Bearer {token}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440019",
        "name": "HighCPUUsage",
        "severity": "warning",
        "status": "active",
        "message": "CPU使用率超过80%",
        "resource_type": "node",
        "resource_id": "550e8400-e29b-41d4-a716-446655440000",
        "triggered_at": "2026-05-15T09:30:00Z",
        "labels": {
          "node": "node-001"
        }
      }
    ]
  },
  "timestamp": 1715769600000
}
```

---

## 15. 速率限制

### 15.1 限制策略

| API类型 | 限制 | 时间窗口 |
|---------|------|----------|
| 认证接口 | 10次/分钟 | 1分钟 |
| 查询接口 | 100次/分钟 | 1分钟 |
| 写入接口 | 50次/分钟 | 1分钟 |
| 批量接口 | 10次/分钟 | 1分钟 |

### 15.2 响应头

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1715769660
```

### 15.3 超限响应

```json
{
  "code": 2002,
  "message": "Rate limit exceeded",
  "data": {
    "retry_after": 60
  },
  "timestamp": 1715769600000
}
```

---

## 16. SDK 示例

### 16.1 Python SDK

```python
from edgehub import EdgeHubClient

client = EdgeHubClient(
    endpoint="https://api.edgehub.io",
    api_key="your_api_key"
)

nodes = client.nodes.list(status="online")
print(f"Found {len(nodes.items)} online nodes")

job = client.jobs.submit(
    name="training-job",
    type="training",
    resources={"cpu": "16", "memory": "64Gi", "gpu": "4"},
    spec={
        "image": "pytorch/pytorch:2.0",
        "command": ["python", "train.py"]
    }
)
print(f"Job submitted: {job.id}")
```

### 16.2 Go SDK

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/edgehub/edgehub-go"
)

func main() {
    client := edgehub.NewClient("https://api.edgehub.io", "your_api_key")
    
    nodes, err := client.Nodes.List(context.Background(), &edgehub.NodeListOptions{
        Status: "online",
    })
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found %d online nodes\n", len(nodes.Items))
    
    job, err := client.Jobs.Submit(context.Background(), &edgehub.JobSubmitRequest{
        Name: "training-job",
        Type: "training",
        Resources: edgehub.Resources{
            CPU:    "16",
            Memory: "64Gi",
            GPU:    "4",
        },
    })
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Job submitted: %s\n", job.ID)
}
```

---

## 17. 变更日志

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| v1.0 | 2026-05-15 | 初始版本发布 |

---

*文档版本：V1.0*
*最后更新：2026年5月*
*维护者：EdgeHub Team*
