#!/bin/bash

# EdgeAgent-Hub 推送脚本
# 使用方法: ./push.sh <your-gitcode-username> <your-gitcode-token>

set -e

REPO_URL="https://gitcode.com/ywtech/EdgeAgent-Hub.git"
USERNAME="${1:-}"
TOKEN="${2:-}"

if [ -z "$USERNAME" ] || [ -z "$TOKEN" ]; then
    echo "Usage: ./push.sh <username> <token>"
    echo ""
    echo "请提供您的GitCode用户名和访问令牌"
    echo "访问令牌可以在 GitCode -> 设置 -> 访问令牌 中创建"
    exit 1
fi

AUTH_URL="https://${USERNAME}:${TOKEN}@gitcode.com/ywtech/EdgeAgent-Hub.git"

echo "正在推送代码到 ${REPO_URL} ..."

git remote set-url origin "$AUTH_URL"
git push -u origin master --force

echo ""
echo "✅ 推送成功！"
echo "仓库地址: https://gitcode.com/ywtech/EdgeAgent-Hub"