# loadtest.ps1 —— 闭环压测：量 GameServer 的真实请求吞吐与延迟分位
#
# 起一套单 realm（GlobalServer + MapServer + GameServer + Gateway），然后按并发梯度跑
# gclient -mode loadtest。每个压测客户端"发一条等一条"，所以：
#   吞吐 = 完成请求数/秒，延迟 = 单条请求往返。
#
# **看什么**：吞吐是否随并发上升。并发翻倍而吞吐不动、延迟等比例变长 = 服务端存在串行瓶颈。
#
# 用法:
#   powershell -ExecutionPolicy Bypass -File .\scripts\loadtest.ps1
#   powershell -ExecutionPolicy Bypass -File .\scripts\loadtest.ps1 -Op attack -Clients "1,4,16" -Seconds 15
#
# 前置：etcd + MySQL 可达；bin\ 下已有各服务与 gclient 的最新构建（本脚本会自动重建）。

param(
    [string]$Op = "move",
    [string]$Clients = "1,2,4,8,16,32",
    [int]$Seconds = 10,
    [switch]$SkipBuild
)

$ErrorActionPreference = "Continue"
$root = Resolve-Path "$PSScriptRoot\.."
$logDir = Join-Path $root "logs\loadtest"

New-Item -ItemType Directory -Force $logDir | Out-Null
Get-ChildItem $logDir -Filter *.log -ErrorAction SilentlyContinue | Remove-Item -Force
foreach ($old in @("gateway_server_101.log","game_server_101.log","map_server_101.log","global_server_1.log")) {
    Remove-Item (Join-Path $root "logs\$old") -Force -ErrorAction SilentlyContinue
}

$procs = @()
function Start-Svc($name, $exe, $cfg) {
    $out = Join-Path $logDir "$name.out.log"
    $err = Join-Path $logDir "$name.err.log"
    $p = Start-Process -FilePath (Join-Path $root "bin\$exe") -ArgumentList @("-config", $cfg) `
        -WorkingDirectory $root -RedirectStandardOutput $out -RedirectStandardError $err -PassThru -WindowStyle Hidden
    Write-Host "[start] $name pid=$($p.Id)"
    return $p
}

function Stop-All {
    foreach ($p in $script:procs) {
        if ($p -and -not $p.HasExited) { Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue }
    }
}

function Wait-Port($port, $timeoutSec, $what) {
    $deadline = (Get-Date).AddSeconds($timeoutSec)
    while ((Get-Date) -lt $deadline) {
        if (Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue) {
            Write-Host "[ready] $what (:$port)"
            return $true
        }
        Start-Sleep -Milliseconds 300
    }
    Write-Host "[FAIL] $what 未在 ${timeoutSec}s 内监听 :$port" -ForegroundColor Red
    return $false
}

try {
    if (-not $SkipBuild) {
        Write-Host "[build] 重建服务与压测客户端..."
        foreach ($m in @("GlobalServer","GatewayServer","GameServer","MapServer")) {
            Set-Location (Join-Path $root $m)
            go build -o (Join-Path $root "bin\$m.exe") .
            if ($LASTEXITCODE -ne 0) { throw "$m 构建失败" }
        }
        Set-Location (Join-Path $root "GameClient")
        go build -o (Join-Path $root "bin\gclient.exe") ./cmd
        if ($LASTEXITCODE -ne 0) { throw "gclient 构建失败" }
        Set-Location $root
    }

    $procs += Start-Svc "global" "GlobalServer.exe" "GlobalServer/config.ini"
    if (-not (Wait-Port 8888 30 "GlobalServer")) { throw "GlobalServer 启动失败" }

    $procs += Start-Svc "map" "MapServer.exe" "MapServer/config_single.ini"
    if (-not (Wait-Port 30001 30 "MapServer")) { throw "MapServer 启动失败" }

    $procs += Start-Svc "game" "GameServer.exe" "GameServer/config.ini"
    if (-not (Wait-Port 20001 40 "GameServer")) { throw "GameServer 启动失败" }

    # 用压测专用网关配置：生产的 DDoS 阈值是「每 IP 500 包/秒、超限封 24 小时」，而压测客户端
    # 全从 127.0.0.1 来，一开压就被封，量到的会是限流器行为而不是服务端吞吐。
    $procs += Start-Svc "gw" "GatewayServer.exe" "GatewayServer/config_loadtest.ini"
    if (-not (Wait-Port 10001 30 "Gateway")) { throw "Gateway 启动失败" }

    # Gateway 要等 GameServer 的 etcd 心跳刷成 healthy 才会建连；早于此压测会全量失败。
    Write-Host "[wait] 等待 Gateway 连上 GameServer..."
    $gwLog = Join-Path $root "logs\gateway_server_101.log"
    $deadline = (Get-Date).AddSeconds(90)
    $ready = $false
    while ((Get-Date) -lt $deadline) {
        if ((Test-Path $gwLog) -and (Select-String -Path $gwLog -Pattern "Connected to GameServer" -Quiet)) { $ready = $true; break }
        Start-Sleep -Seconds 2
    }
    if (-not $ready) { throw "Gateway 未在 90s 内连上 GameServer（见 logs/gateway_server_101.log）" }
    Write-Host "[ready] Gateway 已连上 GameServer`n"

    $outLog = Join-Path $logDir "loadtest.out.log"
    $p = Start-Process -FilePath (Join-Path $root "bin\gclient.exe") -WorkingDirectory $root -PassThru -NoNewWindow `
        -RedirectStandardOutput $outLog -RedirectStandardError (Join-Path $logDir "loadtest.err.log") `
        -ArgumentList @("-mode","loadtest","-loadOp",$Op,"-loadClients",$Clients,"-loadDuration","$Seconds")

    # 梯度档数 × 每档时长 + 准备开销，留足余量。
    $gradeCount = ($Clients -split ",").Count
    $budget = $gradeCount * ($Seconds + 30) + 60
    if (-not $p.WaitForExit($budget * 1000)) {
        Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue
        throw "压测客户端未在 ${budget}s 内结束"
    }

    Get-Content $outLog | Where-Object { $_ -match "并发|吞吐|===|失败|准备" }
}
catch {
    Write-Host "[ERROR] $_" -ForegroundColor Red
}
finally {
    Stop-All
    Set-Location $root
    Write-Host "`n[cleanup] 进程已停止；日志在 $logDir"
}
