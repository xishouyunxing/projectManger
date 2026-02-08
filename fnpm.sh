#!/bin/bash
# 前端npm命令包装脚本
# 解决npm总是在根目录检索的问题

cd "$(dirname "$0")/frontend"
if [ -f "package.json" ]; then
    echo "在frontend目录执行: npm $*"
    npm "$@"
else
    echo "错误: frontend/package.json 不存在"
    echo "请确保在正确的项目根目录运行此脚本"
    exit 1
fi