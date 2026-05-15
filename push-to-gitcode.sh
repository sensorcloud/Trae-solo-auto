#!/bin/bash

set -e

REPO_URL="https://gitcode.com/hackbomb/aiagenthub.git"
TOKEN_NAME="Deepseek-auto"
TOKEN="8NnYwUVe9mrAwCgM_jnun3qF"
TEMP_DIR="/tmp/aiagenthub-push"
PROJECT_DIR="/workspace/edgehub"

echo "=========================================="
echo "算电运协同产业互联网平台 - GitCode推送脚本"
echo "=========================================="

echo ""
echo "[1/5] 清理临时目录..."
rm -rf "$TEMP_DIR"
mkdir -p "$TEMP_DIR"

echo ""
echo "[2/5] 复制项目文件..."
cd "$TEMP_DIR"
cp -r "$PROJECT_DIR"/* .
cp "$PROJECT_DIR"/.gitignore . 2>/dev/null || true

echo ""
echo "[3/5] 初始化Git仓库..."
git init
git config user.email "aiagenthub@example.com"
git config user.name "AI Agent Hub"

git remote add origin "https://${TOKEN_NAME}:${TOKEN}@gitcode.com/hackbomb/aiagenthub.git"

echo ""
echo "[4/5] 提交更改..."
git add -A
git commit -m "feat: 算电运协同产业互联网平台 v1.0.0

核心功能模块:
- L1能源市场: 绿电交易、储能调度、虚拟电厂(VPP)
- L2算力市场: IaaS/MaaS服务、算力交易
- 智能体沙箱: 基于gVisor/Kata的安全执行环境
- IoT连接器: MQTT/Modbus/OPC-UA多协议支持
- 算电协同调度: 多目标优化引擎

技术栈:
- 后端: Go 1.21+, Kubernetes, Karmada, Volcano
- 前端: React 18, TypeScript, Ant Design 5
- 基础设施: PostgreSQL, Redis, NATS, MQTT

开源集成:
- 华为openFuyao: 多样化算力调度
- HAMi: 异构GPU虚拟化
- Kurator: 分布式云原生平台
" || echo "没有更改需要提交"

echo ""
echo "[5/5] 推送到GitCode..."
git branch -M main
git push -u origin main --force

echo ""
echo "=========================================="
echo "推送完成!"
echo "仓库地址: https://gitcode.com/hackbomb/aiagenthub"
echo "=========================================="
