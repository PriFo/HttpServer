# Setup Claude alias in PowerShell profile
# Run this script once to add 'claude' command to your PowerShell profile

$profilePath = $PROFILE
$profileDir = Split-Path $profilePath -Parent

# Create profile directory if it doesn't exist
if (-not (Test-Path $profileDir)) {
    New-Item -ItemType Directory -Path $profileDir -Force | Out-Null
    Write-Host "Created profile directory: $profileDir" -ForegroundColor Green
}

# Get the current script directory (where claude.ps1 is located)
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$claudeScriptPath = Join-Path $scriptDir "claude.ps1"

# Function to add to profile
$functionCode = @"
# Claude helper function
function claude {
    param([string]`$Command = "help")
    `$scriptPath = "$claudeScriptPath"
    if (Test-Path `$scriptPath) {
        & `$scriptPath `$Command
    } else {
        Write-Host "Claude script not found at: `$scriptPath" -ForegroundColor Red
        Write-Host "Please run this setup script from the HttpServer directory." -ForegroundColor Yellow
    }
}
"@

# Check if function already exists
$profileContent = ""
if (Test-Path $profilePath) {
    $profileContent = Get-Content $profilePath -Raw -ErrorAction SilentlyContinue
}

if ($profileContent -and $profileContent -match "function claude") {
    Write-Host "Claude function already exists in profile." -ForegroundColor Yellow
    Write-Host "Do you want to update it? (Y/N): " -NoNewline -ForegroundColor Cyan
    $response = Read-Host
    if ($response -ne "Y" -and $response -ne "y") {
        Write-Host "Setup cancelled." -ForegroundColor Yellow
        exit
    }
    # Remove old function
    $profileContent = $profileContent -replace "(?s)# Claude helper function.*?^}", ""
    $profileContent = $profileContent.Trim()
}

# Add function to profile
if ($profileContent) {
    $profileContent += "`n`n" + $functionCode
} else {
    $profileContent = $functionCode
}

# Write to profile
try {
    [System.IO.File]::WriteAllText($profilePath, $profileContent, [System.Text.Encoding]::UTF8)
    Write-Host "`nClaude function added to PowerShell profile!" -ForegroundColor Green
    Write-Host "Profile location: $profilePath" -ForegroundColor Cyan
    Write-Host "`nTo use the 'claude' command:" -ForegroundColor Yellow
    Write-Host "  1. Restart PowerShell, OR" -ForegroundColor White
    Write-Host "  2. Run: . `$PROFILE" -ForegroundColor White
    Write-Host "`nThen you can use: claude [command]" -ForegroundColor Green
    Write-Host "  Examples:" -ForegroundColor Cyan
    Write-Host "    claude help" -ForegroundColor White
    Write-Host "    claude server" -ForegroundColor White
    Write-Host "    claude build" -ForegroundColor White
    Write-Host "    claude status" -ForegroundColor White
} catch {
    Write-Host "Error writing to profile: $_" -ForegroundColor Red
    Write-Host "You may need to run PowerShell as Administrator." -ForegroundColor Yellow
}

