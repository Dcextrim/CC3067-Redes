param(
    [switch]$FullTopology
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$goRoot = Join-Path $projectRoot "go-side"
$binRoot = Join-Path $goRoot "bin"
New-Item -ItemType Directory -Force -Path $binRoot | Out-Null

Push-Location -LiteralPath $goRoot
try {
    go run ./cmd/vector-gen
    if ($LASTEXITCODE -ne 0) { throw "No se pudieron generar los vectores canonicos" }

    go test -count=1 ./...
    if ($LASTEXITCODE -ne 0) { throw "Fallaron las pruebas Go" }

    go build -o (Join-Path $binRoot "router.exe") ./cmd/router
    if ($LASTEXITCODE -ne 0) { throw "Fallo la compilacion del router" }
    go build -o (Join-Path $binRoot "client.exe") ./cmd/client
    if ($LASTEXITCODE -ne 0) { throw "Fallo la compilacion del cliente" }
    go build -o (Join-Path $binRoot "server.exe") ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "Fallo la compilacion del servidor" }
}
finally {
    Pop-Location
}

if ($FullTopology) {
    & (Join-Path $PSScriptRoot "run_local_topology.ps1")
    if ($LASTEXITCODE -ne 0) { throw "Fallo la topologia local completa" }
}

Write-Host "Validacion Go finalizada correctamente."
