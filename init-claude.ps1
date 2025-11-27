# Initialize Claude function in current PowerShell session
# Run this script: . .\init-claude.ps1

$scriptPath = Join-Path $PSScriptRoot "claude.ps1"

if (-not (Test-Path $scriptPath)) {
    Write-Host "Error: claude.ps1 not found!" -ForegroundColor Red
    return
}

# Remove existing function if it exists
if (Get-Command claude -ErrorAction SilentlyContinue) {
    Remove-Item Function:\claude -Force
}

# Create the function
function global:claude {
    param([string]$Command = "help")
    $scriptPath = "$PSScriptRoot\claude.ps1"
    & $scriptPath $Command
}

Write-Host "Claude function loaded! You can now use 'claude [command]'" -ForegroundColor Green
Write-Host "Try: claude help" -ForegroundColor Cyan

