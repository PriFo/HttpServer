# PowerShell скрипт для мониторинга backend сервера
# Проверяет статус backend и автоматически перезапускает при необходимости

param(
    [int]$CheckInterval = 30,  # Интервал проверки в секундах
    [int]$MaxFailures = 3,      # Максимум неудачных проверок перед перезапуском
    [switch]$AutoRestart        # Автоматически перезапускать при сбое
)

$ErrorActionPreference = "Continue"
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
$backendExe = Join-Path $scriptPath "httpserver_no_gui.exe"
$failureCount = 0

function Test-BackendHealth {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:9999/health" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
        if ($response.StatusCode -eq 200) {
            $data = $response.Content | ConvertFrom-Json
            return $data.status -eq "healthy"
        }
        return $false
    } catch {
        return $false
    }
}

function Start-BackendServer {
    Write-Host "[$(Get-Date -Format 'HH:mm:ss')] Запуск backend сервера..." -ForegroundColor Yellow
    
    if (-not (Test-Path $backendExe)) {
        Write-Host "[ОШИБКА] Файл $backendExe не найден!" -ForegroundColor Red
        return $false
    }
    
    $env:ARLIAI_API_KEY = "597dbe7e-16ca-4803-ab17-5fa084909f37"
    Start-Process -FilePath $backendExe -WindowStyle Minimized -ErrorAction Stop
    
    # Ждем запуска
    Start-Sleep -Seconds 3
    
    # Проверяем, что сервер запустился
    $attempts = 0
    while ($attempts -lt 10) {
        if (Test-BackendHealth) {
            Write-Host "[$(Get-Date -Format 'HH:mm:ss')] Backend сервер успешно запущен" -ForegroundColor Green
            return $true
        }
        Start-Sleep -Seconds 1
        $attempts++
    }
    
    Write-Host "[ОШИБКА] Backend сервер не отвечает после запуска" -ForegroundColor Red
    return $false
}

function Stop-BackendServer {
    Write-Host "[$(Get-Date -Format 'HH:mm:ss')] Остановка backend сервера..." -ForegroundColor Yellow
    
    $processes = Get-Process -Name "httpserver_no_gui" -ErrorAction SilentlyContinue
    if ($processes) {
        $processes | Stop-Process -Force
        Start-Sleep -Seconds 2
        Write-Host "[$(Get-Date -Format 'HH:mm:ss')] Backend сервер остановлен" -ForegroundColor Green
    }
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Мониторинг Backend сервера" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Интервал проверки: $CheckInterval секунд" -ForegroundColor Yellow
Write-Host "Максимум неудач: $MaxFailures" -ForegroundColor Yellow
Write-Host "Автоперезапуск: $(if ($AutoRestart) { 'Включен' } else { 'Выключен' })" -ForegroundColor Yellow
Write-Host ""
Write-Host "Для остановки нажмите Ctrl+C" -ForegroundColor Gray
Write-Host ""

# Проверяем начальное состояние
if (-not (Test-BackendHealth)) {
    Write-Host "[ПРЕДУПРЕЖДЕНИЕ] Backend сервер не отвечает при старте" -ForegroundColor Yellow
    if ($AutoRestart) {
        Start-BackendServer
    }
}

# Основной цикл мониторинга
try {
    while ($true) {
        $isHealthy = Test-BackendHealth
        
        if ($isHealthy) {
            Write-Host "[$(Get-Date -Format 'HH:mm:ss')] ✓ Backend работает нормально" -ForegroundColor Green
            $failureCount = 0
        } else {
            $failureCount++
            Write-Host "[$(Get-Date -Format 'HH:mm:ss')] ✗ Backend не отвечает (попытка $failureCount/$MaxFailures)" -ForegroundColor Red
            
            if ($failureCount -ge $MaxFailures) {
                Write-Host "[$(Get-Date -Format 'HH:mm:ss')] Достигнут максимум неудачных попыток" -ForegroundColor Red
                
                if ($AutoRestart) {
                    Write-Host "[$(Get-Date -Format 'HH:mm:ss')] Перезапуск backend сервера..." -ForegroundColor Yellow
                    Stop-BackendServer
                    Start-Sleep -Seconds 2
                    if (Start-BackendServer) {
                        $failureCount = 0
                    }
                } else {
                    Write-Host "[$(Get-Date -Format 'HH:mm:ss')] Автоперезапуск выключен. Требуется ручное вмешательство." -ForegroundColor Yellow
                }
            }
        }
        
        Start-Sleep -Seconds $CheckInterval
    }
} catch {
    Write-Host ""
    Write-Host "[$(Get-Date -Format 'HH:mm:ss')] Мониторинг остановлен" -ForegroundColor Yellow
    exit 0
}

