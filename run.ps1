# Double-click friendly / simple start from repo root.
# Usage:  .\run.ps1

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

$go = "C:\Program Files\Go\bin\go.exe"
if (-not (Test-Path $go)) {
  Write-Host "Go not found at $go" -ForegroundColor Red
  Write-Host "Install from https://go.dev/dl/ then re-run." -ForegroundColor Yellow
  exit 1
}

$env:Path = "C:\Program Files\Go\bin;$env:USERPROFILE\.local\bin;$env:APPDATA\npm;" + $env:Path

# SQLite needs data/; go:embed needs both theme dists
New-Item -ItemType Directory -Force -Path data, logs, web\classic\dist, web\default\dist | Out-Null
if (-not (Test-Path "web\default\dist\static")) {
  Write-Host "ERROR: web\default\dist is not a real frontend build." -ForegroundColor Red
  Write-Host "From repo root run:" -ForegroundColor Yellow
  Write-Host "  cd web ; bun install ; cd default ; bun run build" -ForegroundColor Yellow
  exit 1
}
# Keep classic theme in sync with default so a classic theme setting never serves a stub page
Copy-Item -Recurse -Force "web\default\dist\*" "web\classic\dist\"

Write-Host "Using: $(& $go version)" -ForegroundColor Cyan
Write-Host "Starting NovaPuraAI on http://localhost:3000 ..." -ForegroundColor Green
Write-Host "Press Ctrl+C to stop.`n"

& $go run .
