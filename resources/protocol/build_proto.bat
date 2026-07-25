@echo off
chcp 65001 >nul
echo ============================================
echo 协议文件构建脚本
echo ============================================
echo.

set PROTO_DIR=%~dp0
set OUTPUT_DIR=%PROTO_DIR%..\..\zCommon\protocol

echo [1/5] 检查输出目录..
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
echo [3/5] 编译 mmo_auth.proto (认证协议)...
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative -I"%PROTO_DIR%" "%PROTO_DIR%mmo_auth.proto"
if errorlevel 1 (
    echo [错误] mmo_auth.proto 编译失败
    exit /b 1
)
echo [成功] mmo_auth.proto 编译完成

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
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative -I"%PROTO_DIR%" "%PROTO_DIR%internal.proto"
if errorlevel 1 (
    echo [错误] internal.proto 编译失败
    exit /b 1
)
echo [成功] internal.proto 编译完成

echo.
echo [7/7] 编译 item.proto (物品/仓库协议)...
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative -I"%PROTO_DIR%" "%PROTO_DIR%item.proto"
if errorlevel 1 (
    echo [错误] item.proto 编译失败
    exit /b 1
)
echo [成功] item.proto 编译完成

echo.
echo [8/8] 编译 skill.proto (技能协议)...
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative -I"%PROTO_DIR%" "%PROTO_DIR%skill.proto"
if errorlevel 1 (
    echo [错误] skill.proto 编译失败
    exit /b 1
)
echo [成功] skill.proto 编译完成

echo.
echo [9/9] 编译 chat.proto (聊天协议)...
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative -I"%PROTO_DIR%" "%PROTO_DIR%chat.proto"
if errorlevel 1 (
    echo [错误] chat.proto 编译失败
    exit /b 1
)
echo [成功] chat.proto 编译完成

echo.
echo [10/10] 编译 team.proto (组队协议)...
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative -I"%PROTO_DIR%" "%PROTO_DIR%team.proto"
if errorlevel 1 (
    echo [错误] team.proto 编译失败
    exit /b 1
)
echo [成功] team.proto 编译完成

echo.
echo [11/11] 编译 trade.proto (交易协议)...
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative -I"%PROTO_DIR%" "%PROTO_DIR%trade.proto"
if errorlevel 1 (
    echo [错误] trade.proto 编译失败
    exit /b 1
)
echo [成功] trade.proto 编译完成

echo.
echo [12/12] 编译 mail.proto (邮件协议)...
protoc --go_out="%OUTPUT_DIR%" --go_opt=paths=source_relative -I"%PROTO_DIR%" "%PROTO_DIR%mail.proto"
if errorlevel 1 (
    echo [错误] mail.proto 编译失败
    exit /b 1
)
echo [成功] mail.proto 编译完成

echo.
echo ============================================
echo 所有协议文件编译完成！
echo 输出目录: %OUTPUT_DIR%
echo ============================================