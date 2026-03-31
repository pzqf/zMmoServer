param(
    [switch]$RunClientSmoke
)

$ErrorActionPreference = "Stop"

Write-Host "[1/3] MapServer connection 集成测试..."
Push-Location "../MapServer"
try {
    go test ./connection -v
} finally {
    Pop-Location
}

Write-Host "[2/3] GameServer 编译测试..."
Push-Location "../GameServer"
try {
    go test ./...
} finally {
    Pop-Location
}

Write-Host "[3/3] MapServer 编译测试..."
Push-Location "../MapServer"
try {
    go test ./...
} finally {
    Pop-Location
}

Write-Host "Smoke integration checks passed."

if ($RunClientSmoke) {
    Write-Host "[4/4] testclient map-combat 烟测..."
    Push-Location ".."
    try {
        go run ./testclient/client.go -smoke-map-combat
    } finally {
        Pop-Location
    }
}
