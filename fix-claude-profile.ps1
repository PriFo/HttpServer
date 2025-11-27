# Fix Claude function in PowerShell profile
# This script will properly add the claude function to your profile

$ErrorActionPreference = "Stop"

# Get profile path
try {
    $profilePath = $PROFILE
    Write-Host "Profile path: $profilePath" -ForegroundColor Cyan
} catch {
    Write-Host "Error getting profile path: $_" -ForegroundColor Red
    exit 1
}

# Get script directory
$scriptDir = $PSScriptRoot
$claudeScriptPath = Join-Path $scriptDir "claude.ps1"

if (-not (Test-Path $claudeScriptPath)) {
    Write-Host "Error: claude.ps1 not found at $claudeScriptPath" -ForegroundColor Red
    exit 1
}

# Create profile directory if needed
$profileDir = Split-Path $profilePath -Parent
if (-not (Test-Path $profileDir)) {
    try {
        New-Item -ItemType Directory -Path $profileDir -Force | Out-Null
        Write-Host "Created profile directory: $profileDir" -ForegroundColor Green
    } catch {
        Write-Host "Error creating profile directory: $_" -ForegroundColor Red
        exit 1
    }
}

# Function code to add
$functionCode = @"

# Claude helper function - Added by fix-claude-profile.ps1
function claude {
    param([string]`$Command = "help")
    `$scriptPath = "$claudeScriptPath"
    if (Test-Path `$scriptPath) {
        & `$scriptPath `$Command
    } else {
        Write-Host "Claude script not found at: `$scriptPath" -ForegroundColor Red
        Write-Host "Please run fix-claude-profile.ps1 from the HttpServer directory." -ForegroundColor Yellow
    }
}
"@

# Read existing profile
$profileContent = ""
if (Test-Path $profilePath) {
    try {
        $profileContent = Get-Content $profilePath -Raw -Encoding UTF8 -ErrorAction Stop
        Write-Host "Found existing profile" -ForegroundColor Green
    } catch {
        Write-Host "Warning: Could not read existing profile: $_" -ForegroundColor Yellow
        $profileContent = ""
    }
}

# Remove old claude function if exists
if ($profileContent -and $profileContent -match "(?s)# Claude helper function.*?^}") {
    Write-Host "Removing old claude function..." -ForegroundColor Yellow
    $profileContent = $profileContent -replace "(?s)# Claude helper function.*?^}", ""
    $profileContent = $profileContent.Trim()
}

# Add new function
if ($profileContent -and $profileContent.Length -gt 0) {
    # Check if function already exists (new version)
    if ($profileContent -match "function claude") {
        Write-Host "Claude function already exists in profile (new version)." -ForegroundColor Yellow
        Write-Host "Do you want to update it? (Y/N): " -NoNewline -ForegroundColor Cyan
        $response = Read-Host
        if ($response -ne "Y" -and $response -ne "y") {
            Write-Host "Update cancelled." -ForegroundColor Yellow
            exit 0
        }
        # Remove existing function
        $profileContent = $profileContent -replace "(?s)function claude.*?^}", ""
        $profileContent = $profileContent.Trim()
    }
    
    # Add newline if needed
    if (-not $profileContent.EndsWith("`n") -and -not $profileContent.EndsWith("`r`n")) {
        $profileContent += "`n`n"
    } else {
        $profileContent += "`n"
    }
    $profileContent += $functionCode
} else {
    $profileContent = $functionCode
}

# Write to profile
try {
    # Use UTF8 with BOM for better compatibility
    $utf8WithBom = New-Object System.Text.UTF8Encoding $true
    [System.IO.File]::WriteAllText($profilePath, $profileContent, $utf8WithBom)
    Write-Host "`nProfile updated successfully!" -ForegroundColor Green
    Write-Host "Profile location: $profilePath" -ForegroundColor Cyan
    
    Write-Host "`nTo use the 'claude' command:" -ForegroundColor Yellow
    Write-Host "  1. Restart PowerShell, OR" -ForegroundColor White
    Write-Host "  2. Run: . `$PROFILE" -ForegroundColor White
    Write-Host "  3. Or run: . .\init-claude.ps1 (for current session only)" -ForegroundColor White
    
    Write-Host "`nThen you can use: claude [command]" -ForegroundColor Green
    Write-Host "  Examples:" -ForegroundColor Cyan
    Write-Host "    claude help" -ForegroundColor White
    Write-Host "    claude server" -ForegroundColor White
    Write-Host "    claude build" -ForegroundColor White
    Write-Host "    claude status" -ForegroundColor White
    
} catch {
    Write-Host "Error writing to profile: $_" -ForegroundColor Red
    Write-Host "You may need to run PowerShell as Administrator." -ForegroundColor Yellow
    exit 1
}

