param(
    [switch]$live,
    [switch]$RequireOkFile,
    [string[]]$Args
)

if (-not $live) {
    Write-Error "LIVE lock missing. Re-run with --live."
    exit 1
}

if ($RequireOkFile) {
    if (-not (Test-Path -Path "var\\LIVE.ok")) {
        Write-Error "LIVE.ok missing at var\\LIVE.ok"
        exit 1
    }
}

Write-Host "LIVE lock ok."
& "$PSScriptRoot\\run.ps1" @Args
exit $LASTEXITCODE
