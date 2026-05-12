# EdgeHub 边缘算力集群聚合平台

## 部署指南 v1.2.0

本文档提供 EdgeHub 平台在各环境下的详细部署方案。

---

## 目录

1. [环境要求](#1-环境要求)
2. [Kubernetes 原生部署](#2-kubernetes-原生部署)
3. [阿里云部署](#3-阿里云部署)
4. [华为云部署](#4-华为云部署)
5. [腾讯云部署](#5-腾讯云部署)
6. [Docker Compose 部署](#6-docker-compose-部署)
7. [Kind 本地部署](#7-kind-本地部署)
8. [虚拟机部署](#8-虚拟机部署)
9. [生产环境最佳实践](#9-生产环境最佳实践)

---

## 1. 环境要求

### 1.1 硬件要求

| 组件 | 最低配置 | 推荐配置 |
|------|----------|----------|
| API Server | 2核CPU, 4GB内存 | 4核CPU, 8GB内存 |
| Scheduler | 2核CPU, 2GB内存 | 4核CPU, 4GB内存 |
| PostgreSQL | 2核CPU, 4GB内存, 50GB存储 | 4核CPU, 8GB内存, 100GB SSD |
| Redis | 1核CPU, 2GB内存 | 2核CPU, 4GB内存 |

### 1.2 软件要求

- Kubernetes >= 1.28
- Helm >= 3.12
- kubectl >= 1.28
- PostgreSQL >= 16
- Redis >= 7

---

## 2. Kubernetes 原生部署

### 2.1 使用 Helm 快速部署

```bash
helm repo add edgehub https://charts.edgehub.io
helm repo update

helm install edgehub edgehub/edgehub \
  --namespace edgehub-system \
  --create-namespace \
  --set apiServer.replicas=2
```

### 2.2 使用 YAML 清单部署

```bash
kubectl apply -f config/manifests/00-namespace.yaml
kubectl apply -f config/manifests/01-config.yaml
kubectl apply -f config/manifests/02-deployment.yaml
kubectl apply -f config/manifests/03-storage.yaml
kubectl apply -f config/manifests/04-ingress.yaml
```

---

## 3. 阿里云部署

### 3.1 使用 ACK (容器服务)

```bash
# 创建集群
aliyun cs POST /clusters --header "Content-Type:application/json" \
  --body '{"cluster_type":"ManagedKubernetes","name":"edgehub-cluster","region_id":"cn-beijing"}'

# 配置 kubectl
aliyun cs GET /k8s/cluster-id/user_config > kubeconfig
export KUBECONFIG=kubeconfig

# 部署 EdgeHub
kubectl apply -f aliyun/edgehub-aliyun.yaml
```

### 3.2 使用阿里云数据库

```bash
# RDS PostgreSQL
aliyun rds CreateDBInstance \
  --Engine PostgreSQL \
  --EngineVersion 16.0 \
  --DBInstanceClass postgres.rds.g6.large

# Redis
aliyun kvstore CreateInstance \
  --Engine Redis \
  --EngineVersion 7.0
```

---

## 4. 华为云部署

### 4.1 使用 CCE (云容器引擎)

```bash
# 创建集群
curl -X POST "https://cce.cn-north-4.myhuaweicloud.com/api/v3/projects/${PROJECT_ID}/clusters" \
  -H "X-Auth-Token: ${TOKEN}" \
  -d '{"kind":"Cluster","spec":{"type":"ClusterOperation","flavor":"turing","version":"v1.28"}}'

# 部署 EdgeHub
kubectl apply -f huawei/edgehub-huawei.yaml
```

---

## 5. 腾讯云部署

### 5.1 使用 TKE (容器服务)

```bash
# 创建集群
tccli tke CreateCluster \
  --ClusterCIDR "172.16.0.0/16" \
  --ClusterName "edgehub-cluster"

# 部署 EdgeHub
kubectl apply -f tencent/edgehub-tencent.yaml
```

---

## 6. Docker Compose 部署

### 6.1 docker-compose.yaml

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: edgehub
      POSTGRES_USER: edgehub
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"

  api-server:
    image: edgehub/api-server:v1.2.0
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DB_HOST: postgres
      DB_PASSWORD: ${POSTGRES_PASSWORD}
      REDIS_HOST: redis
      REDIS_PASSWORD: ${REDIS_PASSWORD}
    ports:
      - "8080:8080"

volumes:
  postgres_data:
  redis_data:
```

### 6.2 启动命令

```bash
docker compose up -d
```

---

## 7. Kind 本地部署

### 7.1 创建集群

```bash
kind create cluster --name edgehub

# 安装 ingress
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
```

### 7.2 部署 EdgeHub

```bash
helm install edgehub edgehub/edgehub \
  --namespace edgehub-system \
  --create-namespace \
  --set apiServer.service.type=NodePort
```

---

## 8. 虚拟机部署

### 8.1 系统要求

| 组件 | CPU | 内存 | 磁盘 |
|------|-----|------|------|
| API Server | 4核 | 8GB | 50GB SSD |
| PostgreSQL | 4核 | 8GB | 100GB SSD |

### 8.2 安装脚本

```bash
# 安装 PostgreSQL
apt install -y postgresql-16

# 安装 Redis
apt install -y redis-server

# 下载并启动 API Server
curl -LO https://github.com/sensorcloud/edgehub/releases/download/v1.2.0/edgehub-api
chmod +x edgehub-api
./edgehub-api -config /etc/edgehub/config.yaml
```

---

## 9. 生产环境最佳实践

### 9.1 高可用配置

- API Server 至少 3 副本
- PostgreSQL 主从复制
- Redis 集群模式

### 9.2 安全配置

```bash
# 启用 TLS
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -out tls.crt -keyout tls.key

kubectl create secret tls edgehub-tls --cert=tls.crt --key=tls.key
```

### 9.3 备份策略

```bash
# PostgreSQL 备份
pg_dump -h $DB_HOST -U edgehub edgehub | gzip > backup_$(date +%Y%m%d).sql.gz
```

---

## 附录

- **GitHub**: https://github.com/sensorcloud/edgehub
- **文档**: https://docs.edgehub.io

*最后更新: 2026-05-12*
