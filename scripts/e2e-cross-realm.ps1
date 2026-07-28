# e2e-cross-realm.ps1 —— 跨 realm 物理路由端到端验收（realm §5.1）
#
# 起两套完整 realm（各自 Gateway + GameServer + MapServer）+ 共用 GlobalServer，
# 然后两个客户端各自登录**不同 realm**、进同一场跨服活动，断言：
#   1. 两边拿到的 mapServerID / mapID 完全一致 —— 落到同一台 MapServer 的同一张实例地图；
#   2. 两边的 AOI 视野里出现对方的玩家 ID —— 真在同一张图上、能互相看见。
#
# 前置：etcd + MySQL 可达（见各 config.ini 的 Etcd/Database 段）；
#       realm2 的库用 `go run ./cmd/newrealm -id 000201` 建过一次即可。
# 用法: powershell -ExecutionPolicy Bypass -File .\scripts\e2e-cross-realm.ps1

$ErrorActionPreference = "Continue"
$root = Resolve-Path "$PSScriptRoot\.."
$logDir = Join-Path $root "logs\e2e-cross"
$activityID = 77001
$mapConfigID = 5001

New-Item -ItemType Directory -Force $logDir | Out-Null
Get-ChildItem $logDir -Filter *.log -ErrorAction SilentlyContinue | Remove-Item -Force
# 各服务的 zLog 文件是**追加**的：不清掉的话，本轮的"已就绪"判定会命中上一轮的旧日志（假就绪）。
foreach ($old in @("gateway_server_101.log","gateway_server_201.log","game_server_101.log","game_server_201.log",
                   "map_server_101.log","map_server_201.log","global_server_1.log")) {
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
    $procs += Start-Svc "global"   "GlobalServer.exe"  "GlobalServer/config.ini"
    if (-not (Wait-Port 8888 30 "GlobalServer")) { throw "GlobalServer 启动失败" }

    $procs += Start-Svc "map1" "MapServer.exe" "MapServer/config_single.ini"
    $procs += Start-Svc "map2" "MapServer.exe" "MapServer/config_realm2.ini"
    if (-not (Wait-Port 30001 30 "MapServer realm1")) { throw "MapServer1 启动失败" }
    if (-not (Wait-Port 30101 30 "MapServer realm2")) { throw "MapServer2 启动失败" }

    $procs += Start-Svc "game1" "GameServer.exe" "GameServer/config.ini"
    $procs += Start-Svc "game2" "GameServer.exe" "GameServer/config_realm2.ini"
    if (-not (Wait-Port 20001 40 "GameServer realm1")) { throw "GameServer1 启动失败" }
    if (-not (Wait-Port 20101 40 "GameServer realm2")) { throw "GameServer2 启动失败" }

    $procs += Start-Svc "gw1" "GatewayServer.exe" "GatewayServer/config.ini"
    $procs += Start-Svc "gw2" "GatewayServer.exe" "GatewayServer/config_realm2.ini"
    if (-not (Wait-Port 10001 30 "Gateway realm1")) { throw "Gateway1 启动失败" }
    if (-not (Wait-Port 10101 30 "Gateway realm2")) { throw "Gateway2 启动失败" }

    # 等服务发现收敛：Gateway 要等 GameServer 的 etcd 心跳把状态刷成 healthy（心跳 30s 一拍）才会
    # 建立到 GameServer 的连接；早于此启动客户端，其登录消息会被 Gateway 排队丢弃（曾踩）。
    # 直接盯 Gateway 日志里的连接成功标志，比盲等 sleep 可靠。
    Write-Host "[wait] 等待 Gateway 连上各自 realm 的 GameServer..."
    $gwLogs = @("$root\logs\gateway_server_101.log", "$root\logs\gateway_server_201.log")
    $deadline = (Get-Date).AddSeconds(90)
    $gwReady = $false
    while ((Get-Date) -lt $deadline) {
        $hits = 0
        foreach ($gl in $gwLogs) {
            if ((Test-Path $gl) -and (Select-String -Path $gl -Pattern "Connected to GameServer" -Quiet)) { $hits++ }
        }
        if ($hits -eq 2) { $gwReady = $true; break }
        Start-Sleep -Seconds 2
    }
    if (-not $gwReady) { throw "Gateway 未在 90s 内连上 GameServer（见 logs/gateway_server_*.log）" }
    Write-Host "[ready] 两个 realm 的 Gateway 均已连上 GameServer"

    # 再给 GameServer 的跨 realm 发现一拍（doCrossRealmDiscovery 与本 realm 发现同为 30s 轮询，
    # 启动时立刻跑过一次；此处只是让 MapServer 的注册信息稳定）。
    Start-Sleep -Seconds 3

    # 分配接口自检：同一活动连问两次必须给出同一台服务器（粘性）。
    $body = @{ activity_id = $activityID; map_config_id = $mapConfigID } | ConvertTo-Json -Compress
    $a1 = Invoke-RestMethod -Uri "http://127.0.0.1:8888/api/v1/cross/allocate" -Method Post -Body $body -ContentType "application/json"
    $a2 = Invoke-RestMethod -Uri "http://127.0.0.1:8888/api/v1/cross/allocate" -Method Post -Body $body -ContentType "application/json"
    Write-Host "[alloc] #1 server=$($a1.map_server_id) group=$($a1.group_id) addr=$($a1.address)"
    Write-Host "[alloc] #2 server=$($a2.map_server_id) group=$($a2.group_id) addr=$($a2.address)"
    if ($a1.map_server_id -ne $a2.map_server_id) { throw "分配不粘性：$($a1.map_server_id) vs $($a2.map_server_id)" }

    # 两个不同 realm 的客户端进同一场跨服活动（用时间戳做唯一账号，避免复用旧角色）。
    $stamp = [int][double]::Parse((Get-Date -UFormat %s))
    $c1Log = Join-Path $logDir "client-realm1.log"
    $c2Log = Join-Path $logDir "client-realm2.log"

    $c1 = Start-Process -FilePath (Join-Path $root "bin\gclient.exe") -WorkingDirectory $root -PassThru -WindowStyle Hidden `
        -RedirectStandardOutput $c1Log -RedirectStandardError (Join-Path $logDir "client-realm1.err.log") `
        -ArgumentList @("-mode","cross","-global","127.0.0.1:8888","-gateway","127.0.0.1:10001",
            "-account","crossA$stamp","-password","123456","-name","CrossA$stamp",
            "-crossActivity","$activityID","-crossMapConfig","$mapConfigID","-idle","12")
    Start-Sleep -Seconds 3
    $c2 = Start-Process -FilePath (Join-Path $root "bin\gclient.exe") -WorkingDirectory $root -PassThru -WindowStyle Hidden `
        -RedirectStandardOutput $c2Log -RedirectStandardError (Join-Path $logDir "client-realm2.err.log") `
        -ArgumentList @("-mode","cross","-global","127.0.0.1:8888","-gateway","127.0.0.1:10101",
            "-account","crossB$stamp","-password","123456","-name","CrossB$stamp",
            "-crossActivity","$activityID","-crossMapConfig","$mapConfigID","-idle","12")

    $c1.WaitForExit(90000) | Out-Null
    $c2.WaitForExit(90000) | Out-Null

    # ==== 断言 ====
    $ok = $true
    $r1 = (Select-String -Path $c1Log -Pattern "CROSS_ENTER_OK playerID=(\d+) mapServerID=(\d+) mapID=(\d+)").Matches
    $r2 = (Select-String -Path $c2Log -Pattern "CROSS_ENTER_OK playerID=(\d+) mapServerID=(\d+) mapID=(\d+)").Matches

    if (-not $r1 -or -not $r2) {
        Write-Host "[FAIL] 有客户端未成功进入跨服实例（见 $logDir 下 client-*.log）" -ForegroundColor Red
        $ok = $false
    } else {
        $p1 = $r1[0].Groups[1].Value; $s1 = $r1[0].Groups[2].Value; $m1 = $r1[0].Groups[3].Value
        $p2 = $r2[0].Groups[1].Value; $s2 = $r2[0].Groups[2].Value; $m2 = $r2[0].Groups[3].Value
        Write-Host "[realm1] player=$p1 mapServer=$s1 map=$m1"
        Write-Host "[realm2] player=$p2 mapServer=$s2 map=$m2"

        if ($s1 -ne $s2) { Write-Host "[FAIL] 两个 realm 落到了不同 MapServer: $s1 vs $s2" -ForegroundColor Red; $ok = $false }
        elseif ($m1 -ne $m2) { Write-Host "[FAIL] 两个 realm 落到了同服不同实例图: $m1 vs $m2" -ForegroundColor Red; $ok = $false }
        else { Write-Host "[PASS] 两个 realm 的玩家落在同一台 MapServer($s1) 的同一张实例图($m1)" -ForegroundColor Green }

        # AOI 互见：各自视野里应出现对方的玩家 ID。
        $see12 = Select-String -Path $c1Log -Pattern "\[AOI\].*$p2"
        $see21 = Select-String -Path $c2Log -Pattern "\[AOI\].*$p1"
        if ($see12) { Write-Host "[PASS] realm1 玩家视野里看到了 realm2 玩家($p2)" -ForegroundColor Green }
        else { Write-Host "[FAIL] realm1 玩家视野里没有 realm2 玩家($p2)" -ForegroundColor Red; $ok = $false }
        if ($see21) { Write-Host "[PASS] realm2 玩家视野里看到了 realm1 玩家($p1)" -ForegroundColor Green }
        else { Write-Host "[FAIL] realm2 玩家视野里没有 realm1 玩家($p1)" -ForegroundColor Red; $ok = $false }
    }

    Write-Host ""
    if ($ok) {
        Write-Host "==== 跨 realm 物理路由 E2E 通过 ====" -ForegroundColor Green
    } else {
        Write-Host "==== 跨 realm 物理路由 E2E 失败（日志在 $logDir） ====" -ForegroundColor Red
    }
} catch {
    Write-Host "[ERROR] $_" -ForegroundColor Red
    $ok = $false
} finally {
    Stop-All
    Write-Host "[cleanup] 进程已停止；日志保留在 $logDir"
}

if ($ok) { exit 0 } else { exit 1 }
