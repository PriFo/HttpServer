# Детальная диагностика сервера

$ErrorActionPreference = "Continue"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Диагностика сервера" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. Проверка процессов
Write-Host "1. Проверка процессов сервера..." -ForegroundColor Yellow
$serverProcesses = Get-Process | Where-Object {$_.ProcessName -like "*httpserver*"}
if ($serverProcesses) {
    Write-Host "   Найдены процессы:" -ForegroundColor Green
    $serverProcesses | ForEach-Object {
        Write-Host "   - $($_.ProcessName) (PID: $($_.Id), CPU: $($_.CPU), StartTime: $($_.StartTime))" -ForegroundColor White
    }
} else {
    Write-Host "   Процессы сервера не найдены" -ForegroundColor Red
}

Write-Host ""

# 2. Проверка порта
Write-Host "2. Проверка порта 9999..." -ForegroundColor Yellow
$portCheck = netstat -ano | findstr :9999
if ($portCheck) {
    Write-Host "   Порт 9999 занят:" -ForegroundColor Green
    $portCheck | ForEach-Object {
        Write-Host "   $_" -ForegroundColor White
    }
} else {
    Write-Host "   Порт 9999 свободен" -ForegroundColor Red
}

Write-Host ""

# 3. Проверка доступности сервера
Write-Host "3. Проверка доступности сервера..." -ForegroundColor Yellow
try {
    $healthResponse = Invoke-WebRequest -Uri "http://localhost:9999/health" -TimeoutSec 3 -ErrorAction Stop
    Write-Host "   ✅ /health отвечает (Status: $($healthResponse.StatusCode))" -ForegroundColor Green
    Write-Host "   Ответ: $($healthResponse.Content)" -ForegroundColor White
} catch {
    Write-Host "   ❌ /health не отвечает: $($_.Exception.Message)" -ForegroundColor Red
}

try {
    $configResponse = Invoke-WebRequest -Uri "http://localhost:9999/api/config" -TimeoutSec 3 -ErrorAction Stop
    Write-Host "   ✅ /api/config отвечает (Status: $($configResponse.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "   ❌ /api/config не отвечает: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""

# 4. Проверка файлов сервера
Write-Host "4. Проверка файлов сервера..." -ForegroundColor Yellow
$rootDir = Split-Path -Parent $PSScriptRoot
$serverExe = Join-Path $rootDir "httpserver_no_gui.exe"
if (Test-Path $serverExe) {
    $fileInfo = Get-Item $serverExe
    Write-Host "   ✅ $serverExe найден" -ForegroundColor Green
    Write-Host "   Размер: $([math]::Round($fileInfo.Length / 1MB, 2)) MB" -ForegroundColor White
    Write-Host "   Изменен: $($fileInfo.LastWriteTime)" -ForegroundColor White
} else {
    Write-Host "   ❌ $serverExe не найден" -ForegroundColor Red
}

Write-Host ""

# 5. Проверка переменных окружения
Write-Host "5. Проверка переменных окружения..." -ForegroundColor Yellow
if ($env:ARLIAI_API_KEY) {
    Write-Host "   ✅ ARLIAI_API_KEY установлен (длина: $($env:ARLIAI_API_KEY.Length))" -ForegroundColor Green
} else {
    Write-Host "   ⚠️ ARLIAI_API_KEY не установлен" -ForegroundColor Yellow
}

if ($env:CGO_ENABLED) {
    Write-Host "   ✅ CGO_ENABLED = $env:CGO_ENABLED" -ForegroundColor Green
} else {
    Write-Host "   ⚠️ CGO_ENABLED не установлен (по умолчанию: 1)" -ForegroundColor Yellow
}

Write-Host ""

# 6. Проверка баз данных
Write-Host "6. Проверка баз данных..." -ForegroundColor Yellow
$dbFiles = @("1c_data.db", "data.db", "service.db", "normalized_data.db")
foreach ($db in $dbFiles) {
    $dbPath = Join-Path $rootDir $db
    if (Test-Path $dbPath) {
        $dbInfo = Get-Item $dbPath
        Write-Host "   ✅ $db найден ($([math]::Round($dbInfo.Length / 1MB, 2)) MB)" -ForegroundColor Green
    } else {
        Write-Host "   Warning: $db not found (will be created on first run)" -ForegroundColor Yellow
    }
}

Write-Host ""

# 7. Recommendations
Write-Host "7. Recommendations..." -ForegroundColor Yellow
if (-not $serverProcesses) {
    Write-Host "   -> Start server:" -ForegroundColor Cyan
    Write-Host "     cd E:\HttpServer" -ForegroundColor White
    Write-Host "     `$env:ARLIAI_API_KEY='597dbe7e-16ca-4803-ab17-5fa084909f37'" -ForegroundColor White
    Write-Host "     .\httpserver_no_gui.exe" -ForegroundColor White
} elseif (-not $portCheck) {
    Write-Host "   -> Server is running but not listening on port 9999" -ForegroundColor Yellow
    Write-Host "     Check server logs for errors" -ForegroundColor Yellow
} elseif (-not $healthResponse) {
    Write-Host "   -> Server is running but not responding to requests" -ForegroundColor Yellow
    Write-Host "     Possible issues with initialization or configuration" -ForegroundColor Yellow
} else {
    Write-Host "   OK: Server is working normally!" -ForegroundColor Green
    Write-Host "   -> You can run tests:" -ForegroundColor Cyan
    Write-Host "     .\run_tests_windows.ps1" -ForegroundColor White
}

Write-Host ""

