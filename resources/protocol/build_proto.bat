@echo off
chcp 65001 >nul
echo ============================================
echo 协议文件构建脚本
echo ============================================
echo.

set PROTO_DIR=%~dp0
set OUTPUT_DIR=%PROTO_DIR%..\..\zMmoShared\protocol

echo [1/5] 检查输出目录...
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"
if not exist "%OUTPUT_DIR%\interop" mkdir "%OUTPUT_DIR%\interop"

echo.
echo [2/5] 编译 common.proto (通用定义)...
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative "%PROTO_DIR%common.proto"
if errorlevel 1 (
    echo [错误] common.proto 编译失败
    exit /b 1
)
echo [成功] common.proto 编译完成

echo.
echo [3/5] 编译 auth.proto (认证协议)...
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative -I"%PROTO_DIR%" "%PROTO_DIR%auth.proto"
if errorlevel 1 (
    echo [错误] auth.proto 编译失败
    exit /b 1
)
echo [成功] auth.proto 编译完成

echo.
echo [4/5] 编译 player.proto (玩家协议)...
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative -I"%PROTO_DIR%" "%PROTO_DIR%player.proto"
if errorlevel 1 (
    echo [错误] player.proto 编译失败
    exit /b 1
)
echo [成功] player.proto 编译完成

echo.
echo [5/5] 编译 game.proto (游戏协议)...
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative -I"%PROTO_DIR%" "%PROTO_DIR%game.proto"
if errorlevel 1 (
    echo [错误] game.proto 编译失败
    exit /b 1
)
echo [成功] game.proto 编译完成

echo.
echo [6/6] 编译 internal.proto (服务间协议)...
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative "%PROTO_DIR%internal.proto"
if errorlevel 1 (
    echo [错误] internal.proto 编译失败
    exit /b 1
)
echo [成功] internal.proto 编译完成

echo.
echo ============================================
echo 所有协议文件编译完成！
echo 输出目录: %OUTPUT_DIR%
echo ============================================
