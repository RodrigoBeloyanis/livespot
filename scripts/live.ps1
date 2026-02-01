param(
    [switch]$live
)

if (-not $live) {
    Write-Error "LIVE lock missing. Re-run with --live."
    exit 1
}

Write-Host "LIVE lock ok."
