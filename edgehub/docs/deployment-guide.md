# EdgeHub 边缘算力集群聚合平台

## 部署指南 v1.2.0

本文档提供 EdgeHub 平台在各环境下的详细部署方案，包括 Kubernetes 集群、公有云（阿里云、华为云、腾讯云）、Docker、Kind 本地环境以及虚拟机部署。

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
10. [验证与故障排除](#10-验证与故障排除)

---

## 1. 环境要求

### 1.1 硬件要求

| 组件 | 最低配置 | 推荐配置 |
|------|----------|----------|
| API Server | 2核CPU, 4GB内存 | 4核CPU, 8GB内存 |
| Scheduler | 2核CPU, 2GB内存 | 4核CPU, 4GB内存 |
| PostgreSQL | 2核CPU, 4GB内存, 50GB存储 | 4核CPU, 8GB内存, 100GB SSD |
| Redis | 1核CPU, 2GB内存 | 2核CPU, 4GB内存 |
| Web Console | 1核CPU, 1GB内存 | 2核CPU, 2GB内存 |

### 1.2 软件要求

```bash
# 基础工具
- Kubernetes >= 1.28
- Helm >= 3.12
- kubectl >= 1.28
- Docker >= 24.0 (可选)
- PostgreSQL >= 16
- Redis >= 7

# Go 环境 (开发)
- Go >= 1.21
```

### 1.3 网络要求

- **入站端口**: 80 (HTTP), 443 (HTTPS), 8080 (API)
- **出站端口**: 443 (Kubernetes API), 5432 (PostgreSQL), 6379 (Redis)
- **节点间端口**: 10250 (Kubelet), 2379 (etcd)

---

## 2. Kubernetes 原生部署

### 2.1 快速部署 (使用 Helm)

```bash
# 添加 Helm 仓库
helm repo add edgehub https://charts.edgehub.io
helm repo update

# 安装 EdgeHub
helm install edgehub edgehub/edgehub \
  --namespace edgehub-system \
  --create-namespace \
  --set apiServer.replicas=2 \
  --set apiServer.service.type=LoadBalancer \
  --set postgres.enabled=true \
  --set redis.enabled=true

# 检查部署状态
kubectl get pods -n edgehub-system
```

### 2.2 详细配置部署

#### 2.2.1 创建命名空间

```yaml
# config/manifests/00-namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: edgehub-system
  labels:
    name: edgehub-system
```

#### 2.2.2 配置 ConfigMap

```yaml
# config/manifests/01-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: edgehub-config
  namespace: edgehub-system
data:
  config.yaml: |
    server:
      host: "0.0.0.0"
      port: 8080
      mode: "release"
      read_timeout: 60s
      write_timeout: 60s

    database:
      host: "${POSTGRES_HOST}"
      port: 5432
      user: "edgehub"
      password: "${POSTGRES_PASSWORD}"
      name: "edgehub"
      max_open_conns: 100
      max_idle_conns: 10

    redis:
      host: "${REDIS_HOST}"
      port: 6379
      password: "${REDIS_PASSWORD}"
      db: 0
      pool_size: 100

    jwt:
      secret: "${JWT_SECRET}"
      expiration: 24h

    scheduler:
      enabled: true
      workers: 4
      queue_size: 1000

    monitoring:
      enabled: true
      port: 9090
```

#### 2.2.3 部署 API Server

```yaml
# config/manifests/02-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: edgehub-api-server
  namespace: edgehub-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: edgehub-api-server
  template:
    metadata:
      labels:
        app: edgehub-api-server
    spec:
      containers:
      - name: api-server
        image: edgehub/api-server:v1.2.0
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: POSTGRES_HOST
          valueFrom:
            secretKeyRef:
              name: edgehub-secrets
              key: postgres-host
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: edgehub-secrets
              key: postgres-password
        - name: REDIS_HOST
          valueFrom:
            secretKeyRef:
              name: edgehub-secrets
              key: redis-host
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: edgehub-secrets
              key: redis-password
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: edgehub-secrets
              key: jwt-secret
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 2000m
            memory: 4Gi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - edgehub-api-server
              topologyKey: kubernetes.io/hostname
```

#### 2.2.4 部署 Scheduler

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: edgehub-scheduler
  namespace: edgehub-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: edgehub-scheduler
  template:
    metadata:
      labels:
        app: edgehub-scheduler
    spec:
      containers:
      - name: scheduler
        image: edgehub/scheduler:v1.2.0
        ports:
        - containerPort: 9091
        env:
        - name: KUBERNETES_HOST
          value: "https://kubernetes.default.svc"
        - name: API_SERVER_URL
          value: "http://edgehub-api-server:8080"
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 2Gi
```

#### 2.2.5 部署 Ingress

```yaml
# config/manifests/04-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: edgehub-ingress
  namespace: edgehub-system
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
spec:
  tls:
  - hosts:
    - edgehub.example.com
    secretName: edgehub-tls
  rules:
  - host: edgehub.example.com
    http:
      paths:
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: edgehub-api-server
            port:
              number: 8080
      - path: /
        pathType: Prefix
        backend:
          service:
            name: edgehub-web-console
            port:
              number: 80
```

### 2.3 一键部署脚本

```bash
#!/bin/bash
# scripts/deploy-kubernetes.sh

set -e

NAMESPACE="edgehub-system"
CHART_VERSION="1.2.0"

echo "=== EdgeHub Kubernetes 部署脚本 ==="

# 创建命名空间
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# 创建密钥
kubectl create secret generic edgehub-secrets \
  --namespace $NAMESPACE \
  --from-literal=postgres-host="postgres.default.svc" \
  --from-literal=postgres-password="$(openssl rand -base64 32)" \
  --from-literal=redis-host="redis.default.svc" \
  --from-literal=redis-password="$(openssl rand -base64 32)" \
  --from-literal=jwt-secret="$(openssl rand -base64 64)" \
  --dry-run=client -o yaml | kubectl apply -f -

# 应用清单
kubectl apply -f config/manifests/00-namespace.yaml -n $NAMESPACE
kubectl apply -f config/manifests/01-config.yaml -n $NAMESPACE
kubectl apply -f config/manifests/02-deployment.yaml -n $NAMESPACE
kubectl apply -f config/manifests/03-storage.yaml -n $NAMESPACE
kubectl apply -f config/manifests/04-ingress.yaml -n $NAMESPACE

# 等待部署就绪
echo "等待 Pod 就绪..."
kubectl wait --for=condition=ready pod -l app=edgehub-api-server \
  --namespace $NAMESPACE --timeout=300s

echo "=== 部署完成 ==="
kubectl get pods -n $NAMESPACE
```

---

## 3. 阿里云部署

### 3.1 部署架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                         阿里云 VPC                                  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                   ACK (Kubernetes 集群)                       │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │  │
│  │  │ API Server  │  │ Scheduler   │  │ Node Agent  │          │  │
│  │  │  (3副本)    │  │  (2副本)    │  │ (每节点)    │          │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘          │  │
│  │                                                               │  │
│  │  ┌──────────────────────────────────────────────────────┐    │  │
│  │  │              阿里云托管服务                            │    │  │
│  │  │  • RDS PostgreSQL (高可用版)                          │    │  │
│  │  │  • Redis 云数据库 (集群版)                             │    │  │
│  │  │  • OSS 对象存储                                       │    │  │
│  │  └──────────────────────────────────────────────────────┘    │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    SLB 负载均衡                              │  │
│  │  • 公网SLB: edgehub.example.com:443                        │  │
│  │  • 内网SLB: 集群内部通信                                     │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 使用 ACK (容器服务)

```yaml
# aliyun/ack-values.yaml
apiVersion: v1
kind: Secret
metadata:
  name: aliyun-secrets
  namespace: edgehub-system
type: Opaque
stringData:
  # 阿里云 AccessKey (建议使用 RAM 子账号)
  ALIBABA_CLOUD_ACCESS_KEY_ID: "${ACCESS_KEY_ID}"
  ALIBABA_CLOUD_ACCESS_KEY_SECRET: "${ACCESS_KEY_SECRET}"

---
# aliyun/edgehub-aliyun.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: edgehub-api-server
  namespace: edgehub-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: edgehub-api-server
  template:
    metadata:
      annotations:
        # 阿里云日志采集
        aliyun.log_store: edgehub-logstore
    spec:
      containers:
      - name: api-server
        image: edgehub/api-server:v1.2.0
        env:
        - name: ALIBABA_CLOUD_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: aliyun-secrets
              key: ALIBABA_CLOUD_ACCESS_KEY_ID
        - name: DB_HOST
          value: "pgm-xxxxxxxx.pg.rds.aliyuncs.com"  # RDS 内网地址
        - name: REDIS_HOST
          value: "r-xxxxxxxx.redis.rds.aliyuncs.com"  # Redis 内网地址
        resources:
          requests:
            cpu: 1000m
            memory: 2Gi
          limits:
            cpu: 4000m
            memory: 8Gi
```

### 3.3 ACK 快速部署

```bash
#!/bin/bash
# aliyun/deploy-ack.sh

set -e

# 配置阿里云凭证
export ALIBABA_CLOUD_ACCESS_KEY_ID="your-access-key-id"
export ALIBABA_CLOUD_ACCESS_KEY_SECRET="your-access-key-secret"

# 创建集群 (如果不存在)
aliyun cs POST /clusters \
  --header "Content-Type=application/json" \
  --body '{
    "cluster_type": "ManagedKubernetes",
    "name": "edgehub-cluster",
    "region_id": "cn-beijing",
    "kubernetes_version": "1.28",
    "vpcid": "vpc-xxxxxxxx",
    "vswitch_ids": ["vsw-xxxxxxxx"],
    "container_cidr": "172.20.0.0/16",
    "service_cidr": "172.21.0.0/20",
    "master_instance_types": ["ecs.g7.large"],
    "master_system_disk_category": "cloud_essd",
    "master_system_disk_size": 120,
    "worker_instance_types": ["ecs.g7.2xlarge"],
    "worker_system_disk_category": "cloud_essd",
    "worker_system_disk_size": 120,
    "num_of_nodes": 3,
    "login_password": "YourPassword123!"
  }'

# 配置 kubectl
aliyun cs GET /k8s/cluster-id/user_config > kubeconfig
export KUBECONFIG=kubeconfig

# 添加阿里云日志服务
helm repo add aliyunlogs https://aliyun.github.io/Logstash-Helm-Chart
helm install aliyun-log-operator aliyunlogs/log-operator -n kube-system

# 部署 EdgeHub
kubectl create namespace edgehub-system
kubectl apply -f aliyun/edgehub-aliyun.yaml

# 配置 SLB Ingress
kubectl apply -f aliyun/ingress-alb.yaml
```

### 3.4 使用阿里云数据库

```bash
# 创建 RDS PostgreSQL
aliyun rds CreateDBInstance \
  --RegionId cn-beijing \
  --Engine PostgreSQL \
  --EngineVersion 16.0 \
  --DBInstanceClass postgres.rds.g6.large \
  --DBInstanceStorage 100 \
  --SecurityGroupId sg-xxxxxxxx \
  --VPCId vpc-xxxxxxxx \
  --VSwitchId vsw-xxxxxxxx

# 创建 Redis
aliyun kvstore CreateInstance \
  --RegionId cn-beijing \
  --Engine Redis \
  --EngineVersion 7.0 \
  --InstanceType redis.sharding.2g.2g.sharding.redis.sharding.pub.2g.2x \
  --Capacity 64 \
  --VpcId vpc-xxxxxxxx \
  --VSwitchId vsw-xxxxxxxx
```

---

## 4. 华为云部署

### 4.1 部署架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                         华为云 VPC                                  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                   CCE (云容器引擎)                            │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │  │
│  │  │ API Server  │  │ Scheduler   │  │ Node Agent  │          │  │
│  │  │  (3副本)    │  │  (2副本)    │  │ (每节点)    │          │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘          │  │
│  │                                                               │  │
│  │  ┌──────────────────────────────────────────────────────┐    │  │
│  │  │              华为云托管服务                            │    │  │
│  │  │  • RDS PostgreSQL (高可用)                             │    │  │
│  │  │  • DCS Redis (分布式缓存)                             │    │  │
│  │  │  • OBS 对象存储                                       │    │  │
│  │  └──────────────────────────────────────────────────────┘    │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    ELB 负载均衡                              │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.2 使用 CCE (云容器引擎)

```bash
#!/bin/bash
# huawei/deploy-cce.sh

set -e

# 配置华为云凭证
export HW_ACCESS_KEY="${ACCESS_KEY_ID}"
export HW_SECRET_KEY="${ACCESS_KEY_SECRET}"
export HW_REGION="cn-north-4"

# 获取 IAM Token
TOKEN=$(curl -s -X POST "https://iam.cn-north-4.myhuaweicloud.com/v3/auth/tokens" \
  -H "Content-Type: application/json" \
  -d '{
    "auth": {
      "identity": {
        "methods": ["password"],
        "password": {
          "user": {
            "name": "your-username",
            "password": "your-password",
            "domain": {
              "name": "your-domain-name"
            }
          }
        }
      },
      "scope": {
        "project": {
          "name": "cn-north-4"
        }
      }
    }
  }' | jq -r '.token.id')

# 创建 CCE 集群
curl -X POST "https://cce.cn-north-4.myhuaweicloud.com/api/v3/projects/${PROJECT_ID}/clusters" \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: ${TOKEN}" \
  -d '{
    "kind": "Cluster",
    "apiVersion": "v3",
    "metadata": {
      "name": "edgehub-cluster",
      "labels": {
        "clusterType": "ClusterOperation"
      }
    },
    "spec": {
      "type": "ClusterOperation",
      "flavor": "turing",
      "version": "v1.28",
      "containerNetworkCidr": "172.16.0.0/16",
      "containerNetworkMode": "overlay_l2",
      "serviceNetworkCidr": "172.17.0.0/16",
      "enableAlphaFeature": false,
      "ipv6enable": false,
      "clusterAz": "cn-north-4a,cn-north-4b,cn-north-4c",
      "authenticatingProxy": {
        "mode": "rbac"
      },
      "billingMode": "postPaid"
    }
  }'
```

### 4.3 CCE 部署配置

```yaml
# huawei/edgehub-huawei.yaml
apiVersion: v1
kind: Secret
metadata:
  name: huawei-secrets
  namespace: edgehub-system
type: Opaque
stringData:
  hw-access-key: "${ACCESS_KEY_ID}"
  hw-secret-key: "${ACCESS_KEY_SECRET}"
  obs-endpoint: "obs.cn-north-4.myhuaweicloud.com"

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: edgehub-api-server
  namespace: edgehub-system
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      containers:
      - name: api-server
        image: edgehub/api-server:v1.2.0
        env:
        - name: DB_HOST
          value: "gw-xxxxxxxx.pg.dbs.huaweicloud.com"  # RDS 内网地址
        - name: DB_PORT
          value: "5432"
        - name: REDIS_HOST
          value: "redis-xxxxxxxx.dcs.dbs.huaweicloud.com"  # Redis 地址
        - name: OBS_ENDPOINT
          valueFrom:
            secretKeyRef:
              name: huawei-secrets
              key: obs-endpoint
        resources:
          requests:
            cpu: "1000m"
            memory: "2Gi"
          limits:
            cpu: "4000m"
            memory: "8Gi"
        volumeMounts:
        - name: obs-cache
          mountPath: /cache
      volumes:
      - name: obs-cache
        emptyDir:
          sizeLimit: 10Gi
```

### 4.4 华为云数据库配置

```bash
# 创建 RDS PostgreSQL
hwcloud rds create \
  --name edgehub-db \
  --flavor-ref "rds.pg.s3.large.2" \
  --volume type SSD,size 100 \
  --vpc-id vpc-xxxxxxxx \
  --subnet-id subnet-xxxxxxxx \
  --security-group-id sg-xxxxxxxx \
  --db-version "16"

# 创建 DCS Redis
hwcloud dcs create \
  --name edgehub-redis \
  --engine Redis \
  --engine-version "7.0" \
  --capacity 64 \
  --vpc-id vpc-xxxxxxxx \
  --subnet-id subnet-xxxxxxxx \
  --bandwidth 256
```

---

## 5. 腾讯云部署

### 5.1 部署架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                         腾讯云 VPC                                  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                   TKE (容器服务)                              │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │  │
│  │  │ API Server  │  │ Scheduler   │  │ Node Agent  │          │  │
│  │  │  (3副本)    │  │  (2副本)    │  │ (每节点)    │          │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘          │  │
│  │                                                               │  │
│  │  ┌──────────────────────────────────────────────────────┐    │  │
│  │  │              腾讯云托管服务                            │    │  │
│  │  │  • PostgreSQL (Managed)                               │    │  │
│  │  │  • Redis (Cluster)                                   │    │  │
│  │  │  • COS 对象存储                                       │    │  │
│  │  └──────────────────────────────────────────────────────┘    │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    CLB 负载均衡                              │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.2 使用 TKE (容器服务)

```bash
#!/bin/bash
# tencent/deploy-tke.sh

set -e

# 配置腾讯云凭证
export TENCENTCLOUD_SECRET_ID="${SECRET_ID}"
export TENCENTCLOUD_SECRET_KEY="${SECRET_KEY}"
export TENCENTCLOUD_REGION="ap-beijing"

# 创建 TKE 集群
tccli tke CreateCluster \
  --ClusterCIDR "172.16.0.0/16" \
  --ClusterName "edgehub-cluster" \
  --ClusterType "MANAGED_CLUSTER" \
  --EngineConfig '{
    "Version": "1.28.4",
    "ContainerRuntime": "containerd"
  }' \
  --MasterConfig '{
    "InstanceType": "S5.LARGE8",
    "VpcId": "vpc-xxxxxxxx",
    "SubnetId": "subnet-xxxxxxxx",
    "ChargeType": "POSTPAID_BY_HOUR"
  }' \
  --WorkerConfig '{
    "InstanceType": "S5.2XLARGE16",
    "VpcId": "vpc-xxxxxxxx",
    "SubnetId": "subnet-xxxxxxxx",
    "Count": 3,
    "ChargeType": "POSTPAID_BY_HOUR"
  }'

# 获取集群凭证
tccli tke DescribeClusterKubeconfig \
  --ClusterId cls-xxxxxxxx

# 安装 edgehub
helm install edgehub \
  oci://ccr.ccs.tencentyun.com/edgehub/edgehub \
  --version 1.2.0 \
  --namespace edgehub-system \
  --create-namespace
```

### 5.3 TKE 部署配置

```yaml
# tencent/edgehub-tencent.yaml
apiVersion: v1
kind: Secret
metadata:
  name: tencent-secrets
  namespace: edgehub-system
type: Opaque
stringData:
  secret-id: "${TENCENTCLOUD_SECRET_ID}"
  secret-key: "${TENCENTCLOUD_SECRET_KEY}"
  cos-bucket: "edgehub-xxxxxxxx-125xxxxxxx"

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: edgehub-api-server
  namespace: edgehub-system
  annotations:
    # 腾讯云日志采集
    cloud.tencent.com/log-info: |
      {
        "logset_id": "ls-xxxxxxxx",
        "topic_id": "topic-xxxxxxxx"
      }
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: api-server
        image: edgehub/api-server:v1.2.0
        env:
        - name: DB_HOST
          value: "10.0.0.100"  # VPC 内网地址
        - name: DB_PORT
          value: "5432"
        - name: DB_NAME
          value: "edgehub"
        - name: REDIS_HOST
          value: "10.0.0.200"  # VPC 内网地址
        - name: COS_BUCKET
          valueFrom:
            secretKeyRef:
              name: tencent-secrets
              key: cos-bucket
        resources:
          requests:
            cpu: "1000m"
            memory: "2Gi"
          limits:
            cpu: "4000m"
            memory: "8Gi"
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sh", "-c", "sleep 10"]
```

### 5.4 腾讯云数据库配置

```bash
# 创建 PostgreSQL
tccli postgres CreateInstances \
  --SpecCode "postgres.s5.large" \
  --Storage 100 \
  --InstanceCount 1 \
  --ChargeType "POSTPAID" \
  --VpcId "vpc-xxxxxxxx" \
  --SubnetId "subnet-xxxxxxxx" \
  --DBVersion "16.3"

# 创建 Redis
tccli redis CreateInstances \
  --TypeId 6 \
  --MemSize 65536 \
  --GoodsNum 1 \
  --ChargeType "POSTPAID" \
  --VpcId "vpc-xxxxxxxx" \
  --SubnetId "subnet-xxxxxxxx"
```

---

## 6. Docker Compose 部署

### 6.1 快速部署

```yaml
# docker/docker-compose.yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: edgehub
      POSTGRES_USER: edgehub
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-edgehub_secret_password}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U edgehub"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD:-redis_secret_password}
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD:-redis_secret_password}", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  api-server:
    image: edgehub/api-server:v1.2.0
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: edgehub
      DB_PASSWORD: ${POSTGRES_PASSWORD:-edgehub_secret_password}
      DB_NAME: edgehub
      REDIS_HOST: redis
      REDIS_PORT: 6379
      REDIS_PASSWORD: ${REDIS_PASSWORD:-redis_secret_password}
      JWT_SECRET: ${JWT_SECRET:-your_jwt_secret_key_change_in_production}
    ports:
      - "8080:8080"
      - "9090:9090"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  scheduler:
    image: edgehub/scheduler:v1.2.0
    depends_on:
      - api-server
    environment:
      API_SERVER_URL: http://api-server:8080
      KUBERNETES_HOST: https://kubernetes.default.svc
    ports:
      - "9091:9091"

  web-console:
    image: edgehub/web-console:v1.2.0
    depends_on:
      - api-server
    ports:
      - "80:80"
    environment:
      API_SERVER_URL: http://api-server:8080

volumes:
  postgres_data:
  redis_data:
```

### 6.2 环境变量文件

```bash
# docker/.env
# 生产环境请修改以下密钥
POSTGRES_PASSWORD=change_this_password_in_production_123
REDIS_PASSWORD=change_this_redis_password_in_production_456
JWT_SECRET=change_this_jwt_secret_to_random_64_character_string

# 可选配置
API_SERVER_PORT=8080
REDIS_PORT=6379
TZ=Asia/Shanghai
```

### 6.3 一键启动脚本

```bash
#!/bin/bash
# docker/deploy.sh

set -e

echo "=== EdgeHub Docker Compose 部署脚本 ==="

# 创建配置目录
mkdir -p docker/config

# 生成随机密钥
if [ ! -f docker/.env ]; then
  cat > docker/.env << EOF
POSTGRES_PASSWORD=$(openssl rand -base64 32)
REDIS_PASSWORD=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 64)
EOF
  echo "已生成配置文件 docker/.env"
fi

# 拉取最新镜像
docker compose -f docker/docker-compose.yaml pull

# 启动服务
docker compose -f docker/docker-compose.yaml up -d

# 等待服务就绪
echo "等待服务启动..."
sleep 10

# 检查状态
echo "=== 服务状态 ==="
docker compose -f docker/docker-compose.yaml ps

# 显示访问信息
echo ""
echo "=== 部署完成 ==="
echo "API Server: http://localhost:8080"
echo "Web Console: http://localhost"
echo ""
echo "查看日志: docker compose -f docker/docker-compose.yaml logs -f"
echo "停止服务: docker compose -f docker/docker-compose.yaml down"
```

---

## 7. Kind 本地部署

### 7.1 环境准备

```bash
#!/bin/bash
# kind/install.sh

set -e

# 安装 Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# 安装 kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# 安装 Kind
curl -Lo kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x kind
sudo mv kind /usr/local/bin/

# 安装 Helm
curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

### 7.2 创建 Kind 集群

```bash
#!/bin/bash
# kind/create-cluster.sh

set -e

cat > kind-config.yaml << 'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: edgehub
nodes:
- role: control-plane
  image: kindest/node:v1.28.0
  extraPortMappings:
  - containerPort: 30080
    hostPort: 80
    protocol: TCP
  - containerPort: 30443
    hostPort: 443
    protocol: TCP
  extraMounts:
  - hostPath: /dev
    containerPath: /dev/mapper
- role: worker
  image: kindest/node:v1.28.0
  labels:
    edgehub.io/gpu: "true"
- role: worker
  image: kindest/node:v1.28.0
  labels:
    edgehub.io/storage: "true"
networking:
  podSubnet: "10.244.0.0/16"
  serviceSubnet: "10.96.0.0/16"
EOF

# 创建集群
kind create cluster --config kind-config.yaml --wait 5m

# 安装 ingress controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

echo "=== Kind 集群创建完成 ==="
kubectl get nodes
```

### 7.3 部署 EdgeHub

```bash
#!/bin/bash
# kind/deploy.sh

set -e

# 添加 Helm 仓库
helm repo add edgehub https://charts.edgehub.io
helm repo update

# 安装 EdgeHub
helm install edgehub edgehub/edgehub \
  --namespace edgehub-system \
  --create-namespace \
  --set apiServer.service.type=NodePort \
  --set apiServer.service.nodePort=30080 \
  --set postgres.enabled=true \
  --set redis.enabled=true \
  --set scheduler.enabled=true

# 等待就绪
kubectl wait --for=condition=ready pod -l app=edgehub-api-server \
  --namespace edgehub-system --timeout=300s

echo "=== 部署完成 ==="
echo "访问地址: http://localhost:80"
kubectl get pods -n edgehub-system
```

### 7.4 Kind GPU 支持 (可选)

```yaml
# kind/kind-gpu-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: edgehub-gpu
nodes:
- role: control-plane
  image: kindest/node:v1.28.0
  extraPortMappings:
  - containerPort: 30080
    hostPort: 80
- role: worker
  image: kindest/node:v1.28.0
  labels:
    nvidia.com/gpu: "true"
    edgehub.io/gpu: "true"
```

```bash
# 启用 NVIDIA GPU 支持
kind create cluster --config kind-gpu-config.yaml

# 安装 NVIDIA Device Plugin
kubectl apply -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/master/nvidia-device-plugin.yml
```

---

## 8. 虚拟机部署

### 8.1 系统要求

| 组件 | CPU | 内存 | 磁盘 | 操作系统 |
|------|-----|------|------|----------|
| API Server | 4核 | 8GB | 50GB SSD | Ubuntu 22.04 / CentOS 8 |
| Scheduler | 2核 | 4GB | 20GB | Ubuntu 22.04 / CentOS 8 |
| PostgreSQL | 4核 | 8GB | 100GB SSD | Ubuntu 22.04 / CentOS 8 |
| Redis | 2核 | 4GB | 20GB SSD | Ubuntu 22.04 / CentOS 8 |
| Web Console | 2核 | 2GB | 20GB | Ubuntu 22.04 / CentOS 8 |

### 8.2 系统初始化脚本

```bash
#!/bin/bash
# vm/init.sh

set -e

echo "=== 系统初始化脚本 ==="

# 更新系统
apt update && apt upgrade -y

# 安装基础软件
apt install -y curl wget git vim htop net-tools \
  apt-transport-https ca-certificates gnupg lsb-release

# 关闭防火墙 (生产环境根据需要配置)
systemctl stop ufw || true
systemctl disable ufw || true

# 设置时区
timedatectl set-timezone Asia/Shanghai

# 设置主机名
hostnamectl set-hostname edgehub-node-01

# 添加 hosts
cat >> /etc/hosts << EOF
192.168.1.100 edgehub-api
192.168.1.101 edgehub-scheduler
192.168.1.102 edgehub-postgres
192.168.1.103 edgehub-redis
192.168.1.104 edgehub-web
EOF

# 禁用 SWAP
swapoff -a
sed -i '/ swap / s/^\(.*\)$/#\1/' /etc/fstab

# 配置内核参数
cat > /etc/sysctl.d/99-edgehub.conf << EOF
net.ipv4.ip_forward = 1
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
fs.inotify.max_user_watches = 524288
vm.max_map_count = 262144
EOF

sysctl -p /etc/sysctl.d/99-edgehub.conf

echo "=== 系统初始化完成 ==="
reboot
```

### 8.3 安装 Docker

```bash
#!/bin/bash
# vm/install-docker.sh

set -e

# 添加 Docker GPG key
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

# 添加 Docker 仓库
echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# 安装 Docker
apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# 启动 Docker
systemctl start docker
systemctl enable docker

# 添加当前用户到 docker 组
usermod -aG docker $USER

echo "=== Docker 安装完成 ==="
docker --version
```

### 8.4 部署 PostgreSQL

```bash
#!/bin/bash
# vm/install-postgres.sh

set -e

# 安装 PostgreSQL 16
sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg
apt update
apt install -y postgresql-16

# 配置 PostgreSQL
cat > /etc/postgresql/16/main/pg_hba.conf << 'EOF'
# TYPE  DATABASE        USER            ADDRESS                 METHOD
local   all             postgres                                peer
local   all             all                                     peer
host    all             all             127.0.0.1/32            md5
host    all             all             10.0.0.0/8              md5
host    all             all             192.168.0.0/16          md5
EOF

# 创建数据库和用户
su - postgres -c "psql -c \"CREATE USER edgehub WITH PASSWORD 'edgehub_password' SUPERUSER;\""
su - postgres -c "psql -c \"CREATE DATABASE edgehub OWNER edgehub;\""

# 配置远程访问
sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" /etc/postgresql/16/main/postgresql.conf

# 启动服务
systemctl restart postgresql
systemctl enable postgresql

echo "=== PostgreSQL 安装完成 ==="
```

### 8.5 部署 Redis

```bash
#!/bin/bash
# vm/install-redis.sh

set -e

# 安装 Redis
apt install -y redis-server

# 配置 Redis
cat > /etc/redis/redis.conf << 'EOF'
bind 0.0.0.0
protected-mode no
port 6379
requirepass redis_password_change_me
maxmemory 2gb
maxmemory-policy allkeys-lru
appendonly yes
EOF

# 启动服务
systemctl restart redis-server
systemctl enable redis-server

echo "=== Redis 安装完成 ==="
```

### 8.6 部署 EdgeHub API Server

```bash
#!/bin/bash
# vm/install-api.sh

set -e

# 下载最新版本
VERSION="1.2.0"
cd /opt
curl -LO https://github.com/sensorcloud/edgehub/releases/download/v${VERSION}/edgehub-api-linux-amd64.tar.gz
tar -xzf edgehub-api-linux-amd64.tar.gz
rm edgehub-api-linux-amd64.tar.gz

# 创建配置
cat > /etc/edgehub/config.yaml << 'EOF'
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

database:
  host: "192.168.1.102"
  port: 5432
  user: "edgehub"
  password: "edgehub_password"
  name: "edgehub"
  max_open_conns: 100
  max_idle_conns: 10

redis:
  host: "192.168.1.103"
  port: 6379
  password: "redis_password_change_me"
  db: 0

jwt:
  secret: "change_this_to_random_64_character_secret_key"
  expiration: 24h

scheduler:
  enabled: true
  workers: 4

monitoring:
  enabled: true
  port: 9090
EOF

# 创建 systemd 服务
cat > /etc/systemd/system/edgehub-api.service << 'EOF'
[Unit]
Description=EdgeHub API Server
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/edgehub
ExecStart=/opt/edgehub/api-server -config /etc/edgehub/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
systemctl daemon-reload
systemctl start edgehub-api
systemctl enable edgehub-api

echo "=== EdgeHub API Server 安装完成 ==="
systemctl status edgehub-api
```

### 8.7 部署 Web Console

```bash
#!/bin/bash
# vm/install-web.sh

set -e

# 安装 Nginx
apt install -y nginx

# 下载 Web Console
VERSION="1.2.0"
curl -LO https://github.com/sensorcloud/edgehub/releases/download/v${VERSION}/edgehub-web.tar.gz
mkdir -p /var/www/edgehub
tar -xzf edgehub-web.tar.gz -C /var/www/edgehub
rm edgehub-web.tar.gz

# 配置 Nginx
cat > /etc/nginx/sites-available/edgehub << 'EOF'
server {
    listen 80;
    server_name edgehub.example.com;

    root /var/www/edgehub;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api {
        proxy_pass http://192.168.1.100:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /metrics {
        proxy_pass http://192.168.1.100:9090;
    }
}
EOF

# 启用站点
ln -sf /etc/nginx/sites-available/edgehub /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
systemctl enable nginx

echo "=== Web Console 安装完成 ==="
```

### 8.8 完整部署脚本

```bash
#!/bin/bash
# vm/deploy-all.sh

set -e

echo "=== EdgeHub 虚拟机完整部署 ==="

# 执行各组件安装
bash vm/init.sh
bash vm/install-docker.sh
bash vm/install-postgres.sh
bash vm/install-redis.sh
bash vm/install-api.sh
bash vm/install-web.sh

echo ""
echo "=== 部署完成 ==="
echo "Web Console: http://edgehub.example.com"
echo "API Server: http://192.168.1.100:8080"
echo "Metrics: http://192.168.1.100:9090"
echo ""
echo "管理命令:"
echo "  systemctl status edgehub-api"
echo "  journalctl -u edgehub-api -f"
```

---

## 9. 生产环境最佳实践

### 9.1 高可用部署

```yaml
# 生产环境高可用配置
# 生产环境应满足以下要求：
# - API Server: 至少 3 副本，跨可用区分布
# - PostgreSQL: 主从复制或读写分离
# - Redis: 集群模式或 Sentinel
# - 使用负载均衡器
```

### 9.2 安全配置

```bash
# 9.2.1 启用 TLS
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -out tls.crt -keyout tls.key \
  -subj "/CN=edgehub/O=EdgeHub"

kubectl create secret tls edgehub-tls \
  --cert=tls.crt --key=tls.key \
  -n edgehub-system

# 9.2.2 配置网络安全策略
cat > network-policy.yaml << 'EOF'
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: edgehub-network-policy
  namespace: edgehub-system
spec:
  podSelector:
    matchLabels:
      app: edgehub-api-server
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
EOF

# 9.2.3 启用 Pod 安全策略
kubectl apply -f security/pod-security-policy.yaml
```

### 9.3 监控与日志

```yaml
# 9.3.1 Prometheus 配置
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: edgehub-system
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s
    scrape_configs:
    - job_name: 'edgehub-api'
      static_configs:
      - targets: ['edgehub-api-server:9090']
        labels:
          service: edgehub
```

### 9.4 备份策略

```bash
#!/bin/bash
# scripts/backup.sh

set -e

BACKUP_DIR="/backups/edgehub"
DATE=$(date +%Y%m%d_%H%M%S)

# PostgreSQL 备份
pg_dump -h $DB_HOST -U edgehub edgehub | gzip > ${BACKUP_DIR}/postgres_${DATE}.sql.gz

# Redis 备份
redis-cli -h $REDIS_HOST -a $REDIS_PASSWORD BGSAVE
cp /var/lib/redis/dump.rdb ${BACKUP_DIR}/redis_${DATE}.rdb

# 保留最近 7 天备份
find ${BACKUP_DIR} -type f -mtime +7 -delete

echo "Backup completed: ${DATE}"
```

### 9.5 灾难恢复

```bash
#!/bin/bash
# scripts/restore.sh

BACKUP_FILE=$1

# 恢复 PostgreSQL
gunzip -c $BACKUP_FILE | psql -h $DB_HOST -U edgehub edgehub

# 恢复 Redis
redis-cli -h $REDIS_HOST -a $REDIS_PASSWORD SHUTDOWN NOSAVE
cp $BACKUP_DIR/redis_*.rdb /var/lib/redis/dump.rdb
redis-server --daemonize yes

echo "Restore completed"
```

---

## 10. 验证与故障排除

### 10.1 部署验证

```bash
#!/bin/bash
# scripts/verify.sh

echo "=== EdgeHub 部署验证 ==="

# 检查 Pod 状态
echo "1. 检查 Pod 状态..."
kubectl get pods -n edgehub-system

# 检查服务
echo "2. 检查服务..."
kubectl get svc -n edgehub-system

# 检查日志
echo "3. 检查 API Server 日志..."
kubectl logs -l app=edgehub-api-server -n edgehub-system --tail=50

# 健康检查
echo "4. 健康检查..."
curl -f http://localhost:8080/health || echo "API Server 不健康"

# Prometheus 指标
echo "5. Prometheus 指标..."
curl -s http://localhost:9090/metrics | head -20

echo "=== 验证完成 ==="
```

### 10.2 常见问题

| 问题 | 原因 | 解决方案 |
|------|------|----------|
| Pod 无法启动 | 镜像拉取失败 | 检查镜像仓库凭证 |
| 数据库连接失败 | 网络策略或密码错误 | 检查 ConfigMap 和 Secret |
| 服务无法访问 | Service 类型或端口错误 | 检查 Service 配置 |
| 性能问题 | 资源限制不足 | 调整 resources limits |
| Pod 不断重启 | Liveness 检查失败 | 检查健康检查配置 |

### 10.3 性能调优

```yaml
# 生产环境资源配置建议
apiVersion: apps/v1
kind: Deployment
metadata:
  name: edgehub-api-server
spec:
  template:
    spec:
      containers:
      - name: api-server
        resources:
          requests:
            cpu: "2000m"
            memory: "4Gi"
          limits:
            cpu: "4000m"
            memory: "8Gi"
        env:
        - name: GOMAXPROCS
          value: "4"
        - name: GOGC
          value: "100"
```

---

## 附录

### A. 快速参考

```bash
# Kubernetes 部署
kubectl apply -f config/manifests/
helm install edgehub charts/edgehub

# Docker Compose 部署
docker compose -f docker/docker-compose.yaml up -d

# Kind 部署
kind create cluster --config kind-config.yaml
helm install edgehub charts/edgehub
```

### B. 版本信息

- **EdgeHub**: v1.2.0
- **Kubernetes**: 1.28+
- **PostgreSQL**: 16+
- **Redis**: 7+
- **Go**: 1.21+

### C. 支持联系

- **GitHub Issues**: https://github.com/sensorcloud/edgehub/issues
- **文档**: https://docs.edgehub.io

---

*本文档最后更新: 2026-05-12*
