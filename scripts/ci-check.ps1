# ci-check.ps1 — 逐模块 go build + go vet 门禁
# 用法: powershell -ExecutionPolicy Bypass -File .\scripts\ci-check.ps1
# 说明: 本仓库为多模块（无 go.work），无法在根一次 build，须逐模块循环。
#       go 的 stderr 会被 PowerShell 包装成 error，故用 $LASTEXITCODE 判成败，不用 2>&1。

$ErrorActionPreference = "Continue"

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
    if (-not (Test-Path (Join-Path $m "go.mod"))) {
        Write-Host "[SKIP] $m (no go.mod)" -ForegroundColor DarkGray
        continue
    }
    $name = Split-Path $m -Leaf
    Set-Location $m

    go build ./...
    $buildExit = $LASTEXITCODE
    go vet ./...
    $vetExit = $LASTEXITCODE

    if ($buildExit -eq 0 -and $vetExit -eq 0) {
        Write-Host "[PASS] $name  (build+vet)" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] $name  build=$buildExit vet=$vetExit" -ForegroundColor Red
        $failed += $name
    }
}

Set-Location $PSScriptRoot
Write-Host ""
if ($failed.Count -eq 0) {
    Write-Host "==== CI CHECK PASSED (build+vet, all modules) ====" -ForegroundColor Green
    exit 0
} else {
    Write-Host "==== CI CHECK FAILED: $($failed -join ', ') ====" -ForegroundColor Red
    exit 1
}
