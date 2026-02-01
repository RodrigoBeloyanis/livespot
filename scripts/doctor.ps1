param(
    [string[]]$Args
)

Write-Host "Running doctor checks..."

if ($Args -and $Args.Length -gt 0) {
    & go run .\cmd\doctor @Args
} else {
    & go run .\cmd\doctor
}

exit $LASTEXITCODE
