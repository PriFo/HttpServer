# Скрипт для автоматического запуска сервера и тестов

$ErrorActionPreference = "Continue"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Chaos Monkey - Автозапуск сервера и тестов" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Поиск сервера
$serverExe = $null
$possiblePaths = @(
    ".\httpserver_no_gui.exe",
    "..\httpserver_no_gui.exe",
    "E:\HttpServer\httpserver_no_gui.exe"
)

foreach ($path in $possiblePaths) {
    $fullPath = Resolve-Path $path -ErrorAction SilentlyContinue
    if ($fullPath -and (Test-Path $fullPath)) {
        $serverExe = $fullPath.Path
        break
    }
}

if (-not $serverExe) {
    Write-Host "❌ Не найден httpserver_no_gui.exe" -ForegroundColor Red
    Write-Host "Искали в:" -ForegroundColor Yellow
    foreach ($path in $possiblePaths) {
        Write-Host "  - $path" -ForegroundColor Yellow
    }
    exit 1
}

Write-Host "✅ Найден сервер: $serverExe" -ForegroundColor Green

# Останавливаем старые процессы
Write-Host ""
Write-Host "Остановка старых процессов сервера..." -ForegroundColor Yellow
Get-Process -Name "httpserver*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# Запускаем сервер
Write-Host "Запуск сервера..." -ForegroundColor Yellow
$serverProcess = Start-Process -FilePath $serverExe -WorkingDirectory (Split-Path $serverExe) -WindowStyle Hidden -PassThru

if (-not $serverProcess) {
    Write-Host "❌ Не удалось запустить сервер" -ForegroundColor Red
    exit 1
}

Write-Host "✅ Сервер запущен (PID: $($serverProcess.Id))" -ForegroundColor Green

# Ждем запуска
Write-Host "Ожидание запуска сервера (15 секунд)..." -ForegroundColor Yellow
$serverReady = $false
for ($i = 0; $i < 15; $i++) {
    Start-Sleep -Seconds 1
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:9999/api/config" -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop
        if ($response.StatusCode -eq 200) {
            Write-Host "✅ Сервер готов (HTTP 200)" -ForegroundColor Green
            $serverReady = $true
            break
        } elseif ($response.StatusCode -eq 502) {
            Write-Host "⚠️ Сервер возвращает 502 (проблемы с конфигурацией)" -ForegroundColor Yellow
        }
    } catch {
        # Продолжаем ждать
    }
    Write-Host "." -NoNewline -ForegroundColor Gray
}

Write-Host ""

if (-not $serverReady) {
    Write-Host "⚠️ Сервер не ответил за 15 секунд" -ForegroundColor Yellow
    Write-Host "Проверьте логи сервера" -ForegroundColor Yellow
}

# Запускаем тесты
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Запуск тестов" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$testScript = Join-Path $PSScriptRoot "run_tests_windows.ps1"
if (Test-Path $testScript) {
    & $testScript @args
    $testExitCode = $LASTEXITCODE
} else {
    Write-Host "❌ Не найден скрипт запуска тестов" -ForegroundColor Red
    $testExitCode = 1
}

# Останавливаем сервер
Write-Host ""
Write-Host "Остановка сервера..." -ForegroundColor Yellow
try {
    Stop-Process -Id $serverProcess.Id -Force -ErrorAction SilentlyContinue
    Write-Host "✅ Сервер остановлен" -ForegroundColor Green
} catch {
    Write-Host "⚠️ Не удалось остановить сервер" -ForegroundColor Yellow
}

exit $testExitCode

