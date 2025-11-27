# Быстрый запуск сервера для тестирования

$ErrorActionPreference = "Continue"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Быстрый запуск сервера" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Переходим в корневую директорию проекта
$rootDir = Split-Path -Parent $PSScriptRoot
Set-Location $rootDir

# Устанавливаем переменные окружения
$env:ARLIAI_API_KEY = "597dbe7e-16ca-4803-ab17-5fa084909f37"
$env:CGO_ENABLED = "1"

# Проверяем наличие сервера
if (-not (Test-Path "httpserver_no_gui.exe")) {
    Write-Host "❌ httpserver_no_gui.exe не найден" -ForegroundColor Red
    Write-Host "Собираю сервер..." -ForegroundColor Yellow
    go build -tags no_gui -o httpserver_no_gui.exe main_no_gui.go
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Ошибка сборки сервера" -ForegroundColor Red
        exit 1
    }
}

# Останавливаем старые процессы
Write-Host "Остановка старых процессов..." -ForegroundColor Yellow
Get-Process | Where-Object {$_.ProcessName -like "*httpserver*"} | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# Запускаем сервер
Write-Host "Запуск сервера..." -ForegroundColor Yellow
$serverProcess = Start-Process -FilePath ".\httpserver_no_gui.exe" -WorkingDirectory "." -PassThru -NoNewWindow

if (-not $serverProcess) {
    Write-Host "❌ Не удалось запустить сервер" -ForegroundColor Red
    exit 1
}

Write-Host "✅ Сервер запущен (PID: $($serverProcess.Id))" -ForegroundColor Green

# Ждем запуска
Write-Host "Ожидание запуска сервера..." -ForegroundColor Yellow
$maxAttempts = 10
$attempt = 0
$serverReady = $false

while ($attempt -lt $maxAttempts -and -not $serverReady) {
    Start-Sleep -Seconds 2
    $attempt++
    
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:9999/health" -TimeoutSec 2 -ErrorAction Stop
        if ($response.StatusCode -eq 200) {
            $serverReady = $true
            Write-Host "✅ Сервер готов!" -ForegroundColor Green
        }
    } catch {
        Write-Host "." -NoNewline -ForegroundColor Gray
    }
}

if (-not $serverReady) {
    Write-Host ""
    Write-Host "⚠️ Сервер не отвечает на /health" -ForegroundColor Yellow
    Write-Host "Проверьте логи сервера" -ForegroundColor Yellow
} else {
    Write-Host ""
    Write-Host "Сервер доступен по адресу: http://localhost:9999" -ForegroundColor Green
}

Write-Host ""
Write-Host "Для остановки сервера нажмите Ctrl+C или закройте это окно" -ForegroundColor Cyan

# Ждем завершения
try {
    Wait-Process -Id $serverProcess.Id
} catch {
    Write-Host "Сервер остановлен" -ForegroundColor Yellow
}

