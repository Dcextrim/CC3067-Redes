$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot

$explicitTargets = @(
    (Join-Path $projectRoot "bin"),
    (Join-Path $projectRoot "router"),
    (Join-Path $projectRoot "router.exe"),
    (Join-Path $projectRoot "client"),
    (Join-Path $projectRoot "client.exe"),
    (Join-Path $projectRoot "server"),
    (Join-Path $projectRoot "server.exe"),
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

$tmpRoot = Join-Path $projectRoot "tmp"
if ((Test-Path -LiteralPath $tmpRoot) -and
    @(Get-ChildItem -LiteralPath $tmpRoot -Force -ErrorAction SilentlyContinue).Count -eq 0) {
    Remove-Item -LiteralPath $tmpRoot -Force
    Write-Host "Eliminado directorio temporal vacio: $tmpRoot"
}
