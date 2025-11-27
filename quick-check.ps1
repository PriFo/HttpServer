# Быстрая проверка состояния системы
# Проверяет статус backend и frontend серверов

$ErrorActionPreference = "Continue"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Быстрая проверка системы" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Проверка Backend
Write-Host "Backend (port 9999):" -ForegroundColor Yellow
try {
    $backendResponse = Invoke-WebRequest -Uri "http://localhost:9999/health" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
    if ($backendResponse.StatusCode -eq 200) {
        $health = $backendResponse.Content | ConvertFrom-Json
        Write-Host "  OK - Running" -ForegroundColor Green
        Write-Host "  Status: $($health.status)" -ForegroundColor Gray
        Write-Host "  Time: $($health.time)" -ForegroundColor Gray
        
        # Проверка API endpoints
        try {
            $clientsResponse = Invoke-WebRequest -Uri "http://localhost:9999/api/clients" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
            $clients = $clientsResponse.Content | ConvertFrom-Json
            Write-Host "  API /api/clients: OK ($($clients.Count) clients)" -ForegroundColor Green
        } catch {
            Write-Host "  API /api/clients: ERROR" -ForegroundColor Red
        }
    }
} catch {
    Write-Host "  ✗ Недоступен" -ForegroundColor Red
    Write-Host "  Ошибка: $($_.Exception.Message)" -ForegroundColor Gray
}

Write-Host ""

# Проверка Frontend
Write-Host "Frontend (port 3000):" -ForegroundColor Yellow
try {
    $frontendResponse = Invoke-WebRequest -Uri "http://localhost:3000" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
    if ($frontendResponse.StatusCode -eq 200) {
        Write-Host "  OK - Running" -ForegroundColor Green
        
        # Проверка API routes
        try {
            $apiResponse = Invoke-WebRequest -Uri "http://localhost:3000/api/dashboard/stats" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
            if ($apiResponse.StatusCode -eq 200) {
                Write-Host "  API /api/dashboard/stats: OK" -ForegroundColor Green
            }
        } catch {
            Write-Host "  API /api/dashboard/stats: ERROR" -ForegroundColor Red
        }
    }
} catch {
    Write-Host "  ✗ Недоступен" -ForegroundColor Red
    Write-Host "  Ошибка: $($_.Exception.Message)" -ForegroundColor Gray
}

Write-Host ""

# Проверка портов
Write-Host "Ports:" -ForegroundColor Yellow
$port9999 = Get-NetTCPConnection -LocalPort 9999 -ErrorAction SilentlyContinue
$port3000 = Get-NetTCPConnection -LocalPort 3000 -ErrorAction SilentlyContinue

if ($port9999) {
    Write-Host "  Port 9999: OK - In use (PID: $($port9999.OwningProcess))" -ForegroundColor Green
} else {
    Write-Host "  Port 9999: FREE" -ForegroundColor Red
}

if ($port3000) {
    Write-Host "  Port 3000: OK - In use (PID: $($port3000.OwningProcess))" -ForegroundColor Green
} else {
    Write-Host "  Port 3000: FREE" -ForegroundColor Red
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan

