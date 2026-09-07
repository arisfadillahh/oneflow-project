param(
  [string]$ComposeFile = "docker-compose.prod.yml",
  [switch]$AllowLocalUrls
)

$ErrorActionPreference = "Stop"
$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$errors = New-Object System.Collections.Generic.List[string]
$warnings = New-Object System.Collections.Generic.List[string]

function Read-EnvFile([string]$fileName) {
  $path = Join-Path $root $fileName
  if (-not (Test-Path $path)) {
    $errors.Add("Missing $fileName. Copy $fileName.example and fill real values.")
    return @{}
  }
  $map = @{}
  Get-Content -LiteralPath $path | ForEach-Object {
    $line = $_.Trim()
    if ($line -eq "" -or $line.StartsWith("#") -or -not $line.Contains("=")) { return }
    $parts = $line.Split("=", 2)
    $map[$parts[0].Trim()] = $parts[1].Trim()
  }
  return $map
}

function Require-Key($map, [string]$fileName, [string[]]$keys) {
  foreach ($key in $keys) {
    if (-not $map.ContainsKey($key) -or [string]::IsNullOrWhiteSpace($map[$key])) {
      $errors.Add("$fileName missing required $key")
    }
  }
}

function Reject-Default($map, [string]$fileName, [string[]]$keys) {
  foreach ($key in $keys) {
    if (-not $map.ContainsKey($key)) { continue }
    $value = $map[$key]
    if ($value -match "replace_with|change_me|local-internal-gateway-token|minioadmin|postgres:postgres@|prodqa-|mock-group") {
      $errors.Add("$fileName has dev/default value for $key")
    }
  }
}

$infra = Read-EnvFile ".env.prod.infra"
$backend = Read-EnvFile ".env.prod.app-backend"
$wa = Read-EnvFile ".env.prod.wa-gateway"
$ai = Read-EnvFile ".env.prod.ai-service"
$worker = Read-EnvFile ".env.prod.worker"

Require-Key $infra ".env.prod.infra" @("POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD", "MINIO_ROOT_USER", "MINIO_ROOT_PASSWORD")
Require-Key $backend ".env.prod.app-backend" @("APP_ENV", "POSTGRES_DSN", "JWT_SECRET", "WEBSOCKET_ALLOWED_ORIGINS", "INTERNAL_GATEWAY_TOKEN", "OBJECT_STORAGE_SECRET_KEY")
Require-Key $wa ".env.prod.wa-gateway" @("WA_ENV", "POSTGRES_DSN", "APP_BACKEND_BASE_URL", "INTERNAL_GATEWAY_TOKEN", "WA_MODE", "WA_ALLOWED_ORIGINS")
Require-Key $ai ".env.prod.ai-service" @("AI_ENV", "POSTGRES_DSN", "OPENROUTER_API_KEY")
Require-Key $worker ".env.prod.worker" @("WORKER_ENV", "POSTGRES_DSN", "OBJECT_STORAGE_SECRET_KEY")

Reject-Default $infra ".env.prod.infra" @("POSTGRES_PASSWORD", "MINIO_ROOT_USER", "MINIO_ROOT_PASSWORD")
Reject-Default $backend ".env.prod.app-backend" @("POSTGRES_DSN", "JWT_SECRET", "INTERNAL_GATEWAY_TOKEN", "OBJECT_STORAGE_ACCESS_KEY", "OBJECT_STORAGE_SECRET_KEY")
Reject-Default $wa ".env.prod.wa-gateway" @("POSTGRES_DSN", "WA_SESSION_DSN", "INTERNAL_GATEWAY_TOKEN", "ESCALATION_GROUP_ID")
Reject-Default $ai ".env.prod.ai-service" @("POSTGRES_DSN", "OPENROUTER_API_KEY")
Reject-Default $worker ".env.prod.worker" @("POSTGRES_DSN", "OPENROUTER_API_KEY", "OBJECT_STORAGE_SECRET_KEY")

foreach ($item in @(
  @(".env.prod.app-backend", "APP_ENV", $backend["APP_ENV"]),
  @(".env.prod.wa-gateway", "WA_ENV", $wa["WA_ENV"]),
  @(".env.prod.ai-service", "AI_ENV", $ai["AI_ENV"]),
  @(".env.prod.worker", "WORKER_ENV", $worker["WORKER_ENV"])
)) {
  if ($item[2] -ne "production") {
    $errors.Add("$($item[0]) $($item[1]) must be production")
  }
}

if (-not $AllowLocalUrls) {
  foreach ($item in @(
    @(".env.prod.app-backend", "WEBSOCKET_ALLOWED_ORIGINS", $backend["WEBSOCKET_ALLOWED_ORIGINS"]),
    @(".env.prod.wa-gateway", "WA_ALLOWED_ORIGINS", $wa["WA_ALLOWED_ORIGINS"])
  )) {
    if ($item[2] -match "localhost|127\.0\.0\.1|\*") {
      $errors.Add("$($item[0]) $($item[1]) must use public HTTPS origin in production")
    }
  }
}

$composePath = Join-Path $root $ComposeFile
if (-not (Test-Path $composePath)) {
  $errors.Add("Missing compose file $ComposeFile")
} else {
  Push-Location $root
  try {
    & docker compose -f $ComposeFile config --quiet
    if ($LASTEXITCODE -ne 0) {
      $errors.Add("docker compose config failed")
    }
  } finally {
    Pop-Location
  }
}

foreach ($url in @("http://localhost:8080/health", "http://localhost:8090/healthz")) {
  try {
    $response = Invoke-WebRequest -UseBasicParsing -Uri $url -TimeoutSec 3
    if ($response.StatusCode -ge 400) { $warnings.Add("$url returned HTTP $($response.StatusCode)") }
  } catch {
    $warnings.Add("$url not reachable; skip if stack is not running locally")
  }
}

if ($warnings.Count -gt 0) {
  Write-Host "Warnings:"
  $warnings | ForEach-Object { Write-Host " - $_" }
}

if ($errors.Count -gt 0) {
  Write-Host "Preflight failed:"
  $errors | ForEach-Object { Write-Host " - $_" }
  exit 1
}

Write-Host "Preflight passed."
