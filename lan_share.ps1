param(
    [string]$Directory = ".",
    [int]$Port = 8000
)

$dir = Resolve-Path $Directory -ErrorAction SilentlyContinue
if (-not $dir) {
    Write-Host "Error: Directory not found: $Directory" -ForegroundColor Red
    exit 1
}

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "  LAN Share" -ForegroundColor Cyan
Write-Host "  Share files securely over your local network" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

python -m lan_share $dir --port $Port
