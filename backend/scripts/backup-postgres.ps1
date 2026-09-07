param(
  [string]$Container = "oneflow-postgres",
  [string]$Database = "oneflow",
  [string]$Username = "postgres",
  [string]$OutputDir = "backups"
)

$ErrorActionPreference = "Stop"
$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$backupDir = Join-Path $root $OutputDir
New-Item -ItemType Directory -Force $backupDir | Out-Null

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$outputPath = Join-Path $backupDir "oneflow-$timestamp.sql"

Write-Host "Creating PostgreSQL backup: $outputPath"
& docker exec $Container pg_dump -U $Username -d $Database --clean --if-exists | Out-File -FilePath $outputPath -Encoding utf8
if ($LASTEXITCODE -ne 0) {
  Remove-Item -LiteralPath $outputPath -ErrorAction SilentlyContinue
  throw "pg_dump failed with exit code $LASTEXITCODE"
}

Write-Host "Backup complete: $outputPath"
