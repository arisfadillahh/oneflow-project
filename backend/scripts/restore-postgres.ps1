param(
  [Parameter(Mandatory = $true)]
  [string]$Path,
  [string]$Container = "oneflow-postgres",
  [string]$Database = "oneflow",
  [string]$Username = "postgres",
  [switch]$Force
)

$ErrorActionPreference = "Stop"
$inputPath = Resolve-Path $Path

if (-not $Force) {
  throw "Restore is destructive. Re-run with -Force after confirming the target database is correct."
}

Write-Host "Restoring PostgreSQL backup into $Container/$Database from $inputPath"
Get-Content -LiteralPath $inputPath | & docker exec -i $Container psql -U $Username -d $Database
if ($LASTEXITCODE -ne 0) {
  throw "psql restore failed with exit code $LASTEXITCODE"
}

Write-Host "Restore complete."
