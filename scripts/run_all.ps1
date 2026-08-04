$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $repoRoot

$env:PYTHONPATH = Join-Path $repoRoot "python-side"
python -m unittest discover -s tests -p "test_python_*.py" -v
if ($LASTEXITCODE -ne 0) { throw "Fallaron las pruebas de Python" }

Push-Location -LiteralPath (Join-Path $repoRoot "go-side")
try {
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "Fallaron las pruebas de Go" }
    go build -o bank-server.exe ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "Fallo la compilacion del servidor" }
}
finally {
    Pop-Location
}

python tests\integration_cross_language.py
if ($LASTEXITCODE -ne 0) { throw "Fallo la prueba de interoperabilidad" }

python scripts\run_experiments.py
if ($LASTEXITCODE -ne 0) { throw "Fallaron los experimentos" }

Write-Host "Todo finalizo correctamente."
