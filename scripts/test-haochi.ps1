# Run NovaPuraAI 号池 (channel pool) dynamic tests.
# Usage:
#   .\scripts\test-haochi.ps1
#   .\scripts\test-haochi.ps1 -LogPath C:\path\to\haochi-tests.log

param(
  [string]$LogPath = ""
)

$ErrorActionPreference = "Continue"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

$run = 'Test(GetRandomSatisfiedChannel|CacheUpdateChannelStatus|ShouldDisableChannel|ShouldRetryByStatusCode|ShouldDisableByStatusCode)'
$pkgs = @("./model", "./service", "./setting/operation_setting")

$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) {
  $msg = "go toolchain not found on PATH"
  Write-Host "FAIL: $msg" -ForegroundColor Red
  if ($LogPath) {
    $msg | Set-Content -Path $LogPath -Encoding utf8
  }
  exit 127
}

Write-Host "Running 号池 tests: $($pkgs -join ' ') -run $run" -ForegroundColor Cyan
$output = & go test -count=1 -timeout 90s @pkgs -run $run 2>&1 | Out-String
Write-Host $output
$exit = $LASTEXITCODE

if ($LogPath) {
  $header = @"
# 号池 test run
# date: $(Get-Date -Format o)
# go: $((go version 2>&1 | Out-String).Trim())
# pkgs: $($pkgs -join ' ')
# run: $run
# exit: $exit

"@
  ($header + $output) | Set-Content -Path $LogPath -Encoding utf8
  Write-Host "Log written: $LogPath" -ForegroundColor Cyan
}

exit $exit
