# NovaPuraAI local dev starter (for people used to npm, not Go).
# Usage (from repo root):
#   .\scripts\start-dev.ps1
#   .\scripts\start-dev.ps1 -SkipFrontendBuild   # if dist already exists
#   .\scripts\start-dev.ps1 -FrontendOnly        # only rsbuild dev (needs API separately)

param(
  [switch]$SkipFrontendBuild,
  [switch]$FrontendOnly
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

Write-Host "Repo: $root" -ForegroundColor Cyan

# --- tools ---
$goCmd = Get-Command go -ErrorAction SilentlyContinue
$go = if ($goCmd) { $goCmd.Source } else { $null }
if (-not $go -and (Test-Path "C:\Program Files\Go\bin\go.exe")) {
  $env:Path = "C:\Program Files\Go\bin;" + $env:Path
  $go = "C:\Program Files\Go\bin\go.exe"
}
$nodeCmd = Get-Command node -ErrorAction SilentlyContinue
$node = if ($nodeCmd) { $nodeCmd.Source } else { $null }
$npmCmd = Get-Command npm -ErrorAction SilentlyContinue
$npm = if ($npmCmd) { $npmCmd.Source } else { $null }
$bunCmd = Get-Command bun -ErrorAction SilentlyContinue
$bun = if ($bunCmd) { $bunCmd.Source } else { $null }
if (-not $bun -and (Test-Path "$env:USERPROFILE\.bun\bin\bun.exe")) {
  $env:Path = "$env:USERPROFILE\.bun\bin;" + $env:Path
  $bun = "$env:USERPROFILE\.bun\bin\bun.exe"
}
if (-not $bun) {
  $npmBun = Join-Path $env:APPDATA "npm\bun.cmd"
  if (Test-Path $npmBun) {
    $env:Path = (Join-Path $env:APPDATA "npm") + ";" + $env:Path
    $bunCmd2 = Get-Command bun -ErrorAction SilentlyContinue
    $bun = if ($bunCmd2) { $bunCmd2.Source } else { $null }
  }
}

if (-not $go) {
  Write-Host "FAIL: Go not found. Install from https://go.dev/dl/ then re-open the terminal." -ForegroundColor Red
  exit 127
}

# Frontend uses Bun workspaces + "catalog:" protocol — plain npm install WILL FAIL.
if (-not $bun) {
  Write-Host @"
FAIL: Bun is required for the frontend.

This repo does NOT support `npm install` in web/default (you will see
  Unsupported URL Type "catalog:"
because package.json uses Bun's catalog: protocol).

Install Bun once, then re-run this script:

  npm install -g bun

  # or:
  powershell -c "irm bun.sh/install.ps1 | iex"

Then open a NEW terminal and:

  cd D:\Noah\文档\Coding\NovaPuraAI-router
  .\scripts\start-dev.ps1
"@ -ForegroundColor Red
  exit 127
}

Write-Host "go:  $go" -ForegroundColor DarkGray
Write-Host "bun: $bun" -ForegroundColor DarkGray
if ($npm) { Write-Host "npm: $npm (do NOT use npm install here — use bun)" -ForegroundColor DarkGray }

# --- .env ---
if (-not (Test-Path ".env")) {
  if (Test-Path ".env.example") {
    Copy-Item ".env.example" ".env"
    Write-Host "Created .env from .env.example — set SESSION_SECRET if empty." -ForegroundColor Yellow
  }
}

# Ensure SESSION_SECRET
$envText = Get-Content ".env" -Raw -ErrorAction SilentlyContinue
if ($envText -match '(?m)^SESSION_SECRET=\s*$' -or $envText -match 'SESSION_SECRET=change_me') {
  $secret = -join ((1..64) | ForEach-Object { '{0:x}' -f (Get-Random -Max 16) })
  if ($envText -match '(?m)^SESSION_SECRET=') {
    $envText = [regex]::Replace($envText, '(?m)^SESSION_SECRET=.*$', "SESSION_SECRET=$secret")
  } else {
    $envText += "`nSESSION_SECRET=$secret`n"
  }
  Set-Content -Path ".env" -Value $envText -Encoding utf8
  Write-Host "Wrote a random SESSION_SECRET into .env" -ForegroundColor Yellow
}

New-Item -ItemType Directory -Force -Path "data", "logs" | Out-Null

function Ensure-DistStub($path) {
  New-Item -ItemType Directory -Force -Path $path | Out-Null
  $index = Join-Path $path "index.html"
  if (-not (Test-Path $index)) {
    Set-Content -Path $index -Value "<!doctype html><html><body><p>NovaPuraAI frontend not built yet.</p></body></html>" -Encoding utf8
  }
}

if ($FrontendOnly) {
  Write-Host "Starting frontend only (rsbuild) on its own port..." -ForegroundColor Cyan
  # Install from web/ workspace root (has bun.lock + catalog)
  Set-Location "web"
  if (-not (Test-Path "node_modules") -or -not (Test-Path "default\node_modules")) {
    & bun install
  }
  Set-Location "default"
  & bun run dev
  exit $LASTEXITCODE
}

# --- build default frontend into web/default/dist (required by go:embed) ---
if (-not $SkipFrontendBuild -or -not (Test-Path "web\default\dist\index.html")) {
  Write-Host "`nBuilding frontend with Bun (not npm)..." -ForegroundColor Cyan
  Push-Location "web"
  try {
    # MUST install from web/ (workspace root), not web/default alone
    Write-Host "bun install in web/ ..." -ForegroundColor DarkGray
    & bun install
    if ($LASTEXITCODE -ne 0) { throw "bun install failed with exit $LASTEXITCODE" }
    Set-Location "default"
    Write-Host "bun run build in web/default ..." -ForegroundColor DarkGray
    & bun run build
    if ($LASTEXITCODE -ne 0) { throw "frontend build failed with exit $LASTEXITCODE" }
  } finally {
    Pop-Location
  }
} else {
  Write-Host "Skipping frontend build (dist exists / -SkipFrontendBuild)" -ForegroundColor DarkGray
}

if (-not (Test-Path "web\default\dist\index.html") -or -not (Test-Path "web\default\dist\static")) {
  Write-Host "FAIL: web/default/dist incomplete after build." -ForegroundColor Red
  exit 1
}
Write-Host "`nStarting Go API on http://localhost:3000 ..." -ForegroundColor Green
Write-Host "First registered user becomes admin." -ForegroundColor DarkGray
Write-Host "Press Ctrl+C to stop.`n" -ForegroundColor DarkGray

# SQLite local mode: leave SQL_DSN empty in .env
& go run .
exit $LASTEXITCODE
