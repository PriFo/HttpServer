# PowerShell скрипт для запуска Chaos Monkey тестов на Windows

$ErrorActionPreference = "Continue"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Chaos Monkey Backend Testing" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Проверка сервера
Write-Host "Проверка доступности сервера..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:9999/api/config" -TimeoutSec 2 -UseBasicParsing -ErrorAction Stop
    Write-Host "✅ Сервер доступен" -ForegroundColor Green
    $serverRunning = $true
} catch {
    Write-Host "❌ Сервер недоступен на http://localhost:9999" -ForegroundColor Red
    Write-Host ""
    Write-Host "Попытка запуска сервера..." -ForegroundColor Yellow
    
    # Ищем исполняемый файл
    $serverExe = $null
    $possiblePaths = @(
        "..\httpserver_no_gui.exe",
        ".\httpserver_no_gui.exe",
        "..\httpserver.exe",
        ".\httpserver.exe"
    )
    
    foreach ($path in $possiblePaths) {
        if (Test-Path $path) {
            $serverExe = Resolve-Path $path
            break
        }
    }
    
    if ($serverExe) {
        Write-Host "Найден сервер: $serverExe" -ForegroundColor Green
        Write-Host "Запуск сервера в фоне..." -ForegroundColor Yellow
        
        $serverProcess = Start-Process -FilePath $serverExe -WindowStyle Hidden -PassThru
        Start-Sleep -Seconds 5
        
        # Проверяем снова
        try {
            $response = Invoke-WebRequest -Uri "http://localhost:9999/api/config" -TimeoutSec 2 -UseBasicParsing -ErrorAction Stop
            Write-Host "✅ Сервер запущен и доступен" -ForegroundColor Green
            $serverRunning = $true
            $needStop = $true
        } catch {
            Write-Host "❌ Сервер не ответил. Проверьте логи." -ForegroundColor Red
            $serverRunning = $false
        }
    } else {
        Write-Host "❌ Не найден исполняемый файл сервера" -ForegroundColor Red
        Write-Host "Искали в: $($possiblePaths -join ', ')" -ForegroundColor Yellow
        $serverRunning = $false
    }
}

if (-not $serverRunning) {
    Write-Host ""
    Write-Host "⚠️ Не удалось запустить или подключиться к серверу" -ForegroundColor Yellow
    Write-Host "Запустите сервер вручную и повторите попытку" -ForegroundColor Yellow
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Запуск тестов" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Определяем тест для запуска
$testName = "all"
if ($args.Count -gt 0) {
    $testName = $args[0]
}

# Проверяем наличие Python
$pythonCmd = $null
$pythonPaths = @("python", "python3", "py")

foreach ($cmd in $pythonPaths) {
    try {
        $version = & $cmd --version 2>&1
        if ($LASTEXITCODE -eq 0) {
            $pythonCmd = $cmd
            break
        }
    } catch {
        continue
    }
}

if (-not $pythonCmd) {
    Write-Host "❌ Python не найден" -ForegroundColor Red
    Write-Host "Установите Python или используйте WSL" -ForegroundColor Yellow
    exit 1
}

Write-Host "Используется Python: $pythonCmd" -ForegroundColor Green
Write-Host ""

# Запускаем тесты
try {
    & $pythonCmd chaos_monkey.py --test $testName
    $exitCode = $LASTEXITCODE
} catch {
    Write-Host "❌ Ошибка при запуске тестов: $_" -ForegroundColor Red
    $exitCode = 1
}

# Останавливаем сервер, если мы его запускали
if ($needStop -and $serverProcess) {
    Write-Host ""
    Write-Host "Остановка сервера..." -ForegroundColor Yellow
    Stop-Process -Id $serverProcess.Id -Force -ErrorAction SilentlyContinue
    Write-Host "✅ Сервер остановлен" -ForegroundColor Green
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Тесты завершены" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

exit $exitCode

