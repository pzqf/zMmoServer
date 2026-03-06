@echo off
echo 正在编译 Protocol Buffers...

:: 检查 protoc 命令是否可用
where protoc >nul 2>nul
if %errorlevel% neq 0 (
    echo 错误: 未找到 protoc 命令，请确保已安装 Protocol Buffers
    exit /b 1
)

:: 编译 game.proto
protoc --go_out=plugins:net/protocol *.proto

if %errorlevel% equ 0 (
    echo 编译成功！已生成 game.pb.go 到 net/protocol/ 目录
) else (
    echo 编译失败，请检查 proto 文件格式
)

pause