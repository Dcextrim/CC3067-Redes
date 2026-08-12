param(
    [string]$Message = "prueba completa de la topologia local"
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$configRoot = Join-Path $projectRoot "configs\local"
$runStamp = Get-Date -Format "yyyyMMdd-HHmmss"
$runRoot = Join-Path $projectRoot "tmp\local_topology\$runStamp"
$binRoot = Join-Path $runRoot "bin"
New-Item -ItemType Directory -Force -Path $binRoot | Out-Null

$routerExe = Join-Path $binRoot "router.exe"
$clientExe = Join-Path $binRoot "client.exe"
$serverExe = Join-Path $binRoot "server.exe"

Push-Location -LiteralPath $projectRoot
try {
    go build -o $routerExe ./cmd/router
    if ($LASTEXITCODE -ne 0) { throw "No se pudo compilar router" }
    go build -o $clientExe ./cmd/client
    if ($LASTEXITCODE -ne 0) { throw "No se pudo compilar client" }
    go build -o $serverExe ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "No se pudo compilar server" }
}
finally {
    Pop-Location
}

Copy-Item -LiteralPath (Join-Path $configRoot "A.json") -Destination $runRoot
Copy-Item -LiteralPath (Join-Path $configRoot "B.json") -Destination $runRoot
Copy-Item -LiteralPath (Join-Path $configRoot "C.json") -Destination $runRoot
Copy-Item -LiteralPath (Join-Path $configRoot "D.json") -Destination $runRoot
Copy-Item -LiteralPath (Join-Path $configRoot "E.json") -Destination $runRoot
Copy-Item -LiteralPath (Join-Path $configRoot "F.json") -Destination $runRoot

$processes = @()
try {
    $serverOut = Join-Path $runRoot "server.out.log"
    $serverErr = Join-Path $runRoot "server.err.log"
    $server = Start-Process -FilePath $serverExe -ArgumentList @("-ip", "127.0.0.1", "-port", "6001") `
        -WorkingDirectory $runRoot -RedirectStandardOutput $serverOut -RedirectStandardError $serverErr `
        -WindowStyle Hidden -PassThru
    $processes += $server

    foreach ($name in @("A", "B", "C", "D", "E", "F")) {
        $stdout = Join-Path $runRoot "$name.out.log"
        $stderr = Join-Path $runRoot "$name.err.log"
        $process = Start-Process -FilePath $routerExe -ArgumentList @("-config", "$name.json", "-noise", "0") `
            -WorkingDirectory $runRoot -RedirectStandardOutput $stdout -RedirectStandardError $stderr `
            -WindowStyle Hidden -PassThru
        $processes += $process
    }

    Write-Host "Esperando 35 segundos para la convergencia de seis routers..."
    Start-Sleep -Seconds 35

    $aTablePath = Join-Path $runRoot "A_tabla_enrutamiento.csv"
    if (-not (Test-Path -LiteralPath $aTablePath)) {
        throw "El router A no genero su tabla de enrutamiento"
    }
    $routeToServer = Import-Csv -LiteralPath $aTablePath |
        Where-Object { $_.destination -eq "127.0.0.1:6001" } |
        Select-Object -First 1
    if ($null -eq $routeToServer) {
        throw "La tabla de A no contiene una ruta hacia el servidor"
    }
    if ([int]$routeToServer.next_hop_port -ne 5001 -or [int]$routeToServer.cost -ne 8) {
        throw "Ruta A->servidor no optima: next_hop=$($routeToServer.next_hop_port), cost=$($routeToServer.cost)"
    }

    & $clientExe -ip 127.0.0.1 -port 6000 -gateway-ip 127.0.0.1 -gateway-port 5000 `
        -to 127.0.0.1:6001 -message $Message -noise 0
    if ($LASTEXITCODE -ne 0) { throw "El cliente no pudo enviar el mensaje" }

    Start-Sleep -Seconds 2
    $serverLog = Get-Content -LiteralPath $serverOut -Raw
    if (-not $serverLog.Contains($Message)) {
        throw "El servidor no recibio el mensaje. Revise los logs en $runRoot"
    }

    Write-Host "OK: cliente -> A -> B -> D -> E -> F -> servidor (costo total 8)."
    Write-Host "Tablas y logs: $runRoot"
}
finally {
    foreach ($process in $processes) {
        if (-not $process.HasExited) {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
    }
}
