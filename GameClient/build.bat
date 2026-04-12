@echo off

rem 构建GameClient脚本

rem 设置工作目录
cd /d "d:\GitHub\zMmoServer\GameClient"

rem 确保依赖正确
echo 正在更新依赖...
go mod tidy

rem 构建程序
echo 正在构建GameClient...
go build -o "../bin/GameClient" ./cmd/main.go

rem 检查构建结果
if %errorlevel% equ 0 (
    echo 构建成功！GameClient已输出到 ../bin/GameClient
) else (
    echo 构建失败！
    exit /b %errorlevel%
)
