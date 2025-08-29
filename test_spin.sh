#!/bin/bash

# 测试脚本：验证 spin 动作修复

echo "=== 测试 Spin 动作处理修复 ==="
echo ""
echo "1. 启动服务器..."
go run main.go &
SERVER_PID=$!
sleep 2

echo ""
echo "2. 启动客户端并发送 spin..."
echo ""

# 运行客户端，自动发送 spin
echo "spin" | go run client/main.go &
CLIENT_PID=$!

# 等待测试完成
sleep 15

echo ""
echo "3. 停止服务..."
kill $SERVER_PID 2>/dev/null
kill $CLIENT_PID 2>/dev/null

echo ""
echo "=== 测试完成 ==="
echo "检查日志以验证："
echo "  - WaitingState 应该收到动作并转换到 GamingState"
echo "  - spin_count 应该大于 0"
echo "  - 应该有 GameSync 消息广播"