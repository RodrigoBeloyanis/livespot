param(
    [string[]]$Args
)

Write-Host "Starting LiveSpot..."

if ($Args -and $Args.Length -gt 0) {
    & go run .\cmd\livespot @Args
} else {
    & go run .\cmd\livespot
}

exit $LASTEXITCODE
