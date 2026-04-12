# 构建GameClient脚本

# 设置工作目录
Set-Location "d:\GitHub\zMmoServer\GameClient"

# 确保依赖正确
echo "正在更新依赖..."
go mod tidy

# 构建程序
echo "正在构建GameClient..."
go build -o "../bin/GameClient" ./cmd/main.go

# 检查构建结果
if ($LASTEXITCODE -eq 0) {
    echo "构建成功！GameClient已输出到 ../bin/GameClient"
} else {
    echo "构建失败！"
    exit $LASTEXITCODE
}
