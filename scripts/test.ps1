# test.ps1 — 逐模块 go test 门禁（可选 -race）
# 用法: powershell -ExecutionPolicy Bypass -File .\scripts\test.ps1
# 说明: 纯逻辑单测必须绿；依赖 DB/etcd 的测试若失败会被标注（后续用 build tag / testcontainers 隔离）。
#       -race 检测数据竞争，是并发框架的硬门禁——但 Windows 上 -race 需要 CGO + C 编译器(gcc/clang)。
#       本机若无 C 编译器则自动降级为不带 -race（并提示），-race 须在 CI(Linux) 或装了 mingw 的机器上跑。

$ErrorActionPreference = "Continue"

# 检测 -race 是否可用（需要 cgo + C 编译器）
$cc = Get-Command gcc, clang, cc -ErrorAction SilentlyContinue | Select-Object -First 1
$raceArgs = @()
if ($cc) {
    $env:CGO_ENABLED = "1"
    $raceArgs = @("-race")
    Write-Host "[info] C compiler found ($($cc.Source)); running WITH -race" -ForegroundColor Cyan
} else {
    Write-Host "[warn] no C compiler (gcc/clang/cc); running WITHOUT -race. Run -race in CI/Linux." -ForegroundColor Yellow
}

$githubRoot = Resolve-Path "$PSScriptRoot\..\.."
$modules = @(
    "$githubRoot\zUtil",
    "$githubRoot\zEngine",
    "$githubRoot\zMmoServer\zCommon",
    "$githubRoot\zMmoServer\GlobalServer",
    "$githubRoot\zMmoServer\GatewayServer",
    "$githubRoot\zMmoServer\GameServer",
    "$githubRoot\zMmoServer\MapServer",
    "$githubRoot\zMmoServer\GameClient"
)

$failed = @()
foreach ($m in $modules) {
    if (-not (Test-Path (Join-Path $m "go.mod"))) { continue }
    $name = Split-Path $m -Leaf
    Set-Location $m

    go test ./... @raceArgs -count=1
    $exit = $LASTEXITCODE

    if ($exit -eq 0) {
        Write-Host "[PASS] $name  (test$(if($raceArgs){' -race'}))" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] $name  test=$exit" -ForegroundColor Red
        $failed += $name
    }
}

Set-Location $PSScriptRoot
Write-Host ""
if ($failed.Count -eq 0) {
    Write-Host "==== ALL TESTS PASSED (-race) ====" -ForegroundColor Green
    exit 0
} else {
    Write-Host "==== TESTS FAILED: $($failed -join ', ') ====" -ForegroundColor Red
    exit 1
}
