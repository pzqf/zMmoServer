# 联调 Smoke 指南

## 目标

提供最小可重复的联调验证入口，快速确认核心链路没有回归：

- MapServer 连接层请求/响应
- GameServer 与 MapServer 编译与基础联通
- 可选：客户端地图入图/移动/攻击流程

## 命令

在项目根目录执行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-integration.ps1
```

该命令会执行：

1. `MapServer/connection` 集成测试
2. `GameServer` 全量 `go test ./...`
3. `MapServer` 全量 `go test ./...`

## 可选：包含客户端地图战斗 Smoke

当 `Global/Gateway/Game/Map` 服务已在线时，可执行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-integration.ps1 -RunClientSmoke
```

脚本会额外运行：

```powershell
go run ./testclient/client.go -smoke-map-combat
```

`-smoke-map-combat` 模式下，任一步失败会返回非 0 退出码，可直接用于 CI。

