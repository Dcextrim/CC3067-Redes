[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$ProjectId,

    [string]$Region = "us-central1",

    [string]$ServiceName = "kaeser-mcp",

    [string]$AuthToken = $env:MCP_AUTH_TOKEN
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command gcloud -ErrorAction SilentlyContinue)) {
    throw "Google Cloud CLI (gcloud) is not installed or is not available in PATH."
}

if ([string]::IsNullOrWhiteSpace($AuthToken)) {
    throw "Set MCP_AUTH_TOKEN or pass -AuthToken. Use a long random value and do not commit it."
}

$serverDirectory = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path

& gcloud config set project $ProjectId
if ($LASTEXITCODE -ne 0) { throw "Unable to select Google Cloud project $ProjectId." }

& gcloud services enable run.googleapis.com cloudbuild.googleapis.com artifactregistry.googleapis.com
if ($LASTEXITCODE -ne 0) { throw "Unable to enable the Google Cloud services required by Cloud Run." }

$environmentVariables = "MCP_AUTH_TOKEN=$AuthToken"
& gcloud run deploy $ServiceName `
    --source $serverDirectory `
    --region $Region `
    --allow-unauthenticated `
    --set-env-vars $environmentVariables `
    --quiet
if ($LASTEXITCODE -ne 0) { throw "Cloud Run deployment failed." }

$serviceUrl = & gcloud run services describe $ServiceName --region $Region --format "value(status.url)"
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($serviceUrl)) {
    throw "The service was deployed, but its public URL could not be read."
}

$health = Invoke-RestMethod -Uri "$serviceUrl/health" -Method Get
if ($health.status -ne "ok") { throw "The deployed health endpoint did not return the expected status." }

Write-Host "Cloud Run deployment completed successfully."
Write-Host "Health URL: $serviceUrl/health"
Write-Host "MCP URL: $serviceUrl/mcp"
Write-Host "Set KAESER_REMOTE_URL to $serviceUrl/mcp before running the chatbot."
