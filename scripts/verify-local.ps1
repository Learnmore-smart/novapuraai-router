# NovaPuraAI local health check
$ErrorActionPreference = "Continue"
$base = $env:NOVAPURA_BASE_URL
if (-not $base) { $base = "http://localhost:3000" }

Write-Host "== Docker containers ==" -ForegroundColor Cyan
docker compose ps 2>$null

Write-Host "`n== API status ==" -ForegroundColor Cyan
try {
  $r = Invoke-WebRequest -Uri "$base/api/status" -UseBasicParsing -TimeoutSec 15
  Write-Host "HTTP $($r.StatusCode)"
  Write-Host $r.Content
  if ($r.Content -match '"success"\s*:\s*true') {
    Write-Host "OK: status success" -ForegroundColor Green
    exit 0
  }
  Write-Host "WARN: response missing success:true" -ForegroundColor Yellow
  exit 2
} catch {
  Write-Host "FAIL: $_" -ForegroundColor Red
  exit 1
}
