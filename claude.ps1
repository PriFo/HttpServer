# Claude helper function
# Usage: .\claude.ps1 [command]

param(
    [string]$Command = "help"
)

function Show-Help {
    Write-Host "Claude Helper - Available commands:" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  help              - Show this help message"
    Write-Host "  server            - Start the HTTP server"
    Write-Host "  build             - Build the Go application"
    Write-Host "  test              - Run tests"
    Write-Host "  status            - Check server status"
    Write-Host ""
    Write-Host "Note: This is a helper script. For AI assistance, use Cursor IDE." -ForegroundColor Yellow
}

function Start-Server {
    Write-Host "Starting HTTP server..." -ForegroundColor Green
    if (Test-Path ".\http_server.exe") {
        & .\http_server.exe
    } elseif (Test-Path ".\server.exe") {
        & .\server.exe
    } else {
        Write-Host "Server executable not found. Building..." -ForegroundColor Yellow
        go build -o server.exe .
        if (Test-Path ".\server.exe") {
            & .\server.exe
        } else {
            Write-Host "Failed to build server" -ForegroundColor Red
        }
    }
}

function Build-Application {
    Write-Host "Building Go application..." -ForegroundColor Green
    go build -o server.exe .
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build successful!" -ForegroundColor Green
    } else {
        Write-Host "Build failed!" -ForegroundColor Red
    }
}

function Run-Tests {
    Write-Host "Running tests..." -ForegroundColor Green
    go test ./...
}

function Check-Status {
    Write-Host "Checking server status..." -ForegroundColor Green
    $response = try {
        Invoke-WebRequest -Uri "http://localhost:8080/health" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
    } catch {
        $null
    }
    
    if ($response -and $response.StatusCode -eq 200) {
        Write-Host "Server is running!" -ForegroundColor Green
    } else {
        Write-Host "Server is not running" -ForegroundColor Red
    }
}

switch ($Command.ToLower()) {
    "help" { Show-Help }
    "server" { Start-Server }
    "build" { Build-Application }
    "test" { Run-Tests }
    "status" { Check-Status }
    default {
        Write-Host "Unknown command: $Command" -ForegroundColor Red
        Write-Host ""
        Show-Help
    }
}

