$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$goRoot = Join-Path $projectRoot "go-side"

$explicitTargets = @(
    (Join-Path $goRoot "bin"),
    (Join-Path $goRoot "router"),
    (Join-Path $goRoot "router.exe"),
    (Join-Path $goRoot "client"),
    (Join-Path $goRoot "client.exe"),
    (Join-Path $goRoot "server"),
    (Join-Path $goRoot "server.exe"),
    (Join-Path $projectRoot "tmp\local_topology")
)

foreach ($target in $explicitTargets) {
    if (Test-Path -LiteralPath $target) {
        Remove-Item -LiteralPath $target -Recurse -Force
        Write-Host "Eliminado: $target"
    }
}

Get-ChildItem -LiteralPath $projectRoot -Filter "*_tabla_enrutamiento.csv" -File -ErrorAction SilentlyContinue |
    ForEach-Object {
        Remove-Item -LiteralPath $_.FullName -Force
        Write-Host "Eliminado: $($_.FullName)"
    }
