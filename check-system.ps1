# Скрипт для проверки состояния системы
# Использование: .\check-system.ps1

Write-Host "`n╔══════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║        🔍 ПРОВЕРКА СОСТОЯНИЯ СИСТЕМЫ                    ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════════╝" -ForegroundColor Cyan

$allOk = $true

# 1. Проверка Go
Write-Host "`n1. Проверка Go..." -ForegroundColor Yellow
$goVersion = go version 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "   ✅ $goVersion" -ForegroundColor Green
} else {
    Write-Host "   ❌ Go не установлен" -ForegroundColor Red
    $allOk = $false
}

# 2. Проверка компиляции
Write-Host "`n2. Проверка компиляции..." -ForegroundColor Yellow
$buildOutput = go build ./cmd/server 2>&1 | Out-String
if ($LASTEXITCODE -eq 0) {
    Write-Host "   ✅ Backend компилируется успешно" -ForegroundColor Green
} else {
    Write-Host "   ❌ Ошибки компиляции:" -ForegroundColor Red
    $buildOutput | Select-String "error:" | Select-Object -First 5 | ForEach-Object {
        Write-Host "      $_" -ForegroundColor Red
    }
    $allOk = $false
}

# 3. Проверка директорий
Write-Host "`n3. Проверка директорий..." -ForegroundColor Yellow
$dirs = @("data", "data/uploads", "data/backups", "data/temp")
foreach ($dir in $dirs) {
    if (Test-Path $dir) {
        Write-Host "   ✅ $dir" -ForegroundColor Green
    } else {
        Write-Host "   ⚠️  $dir не существует" -ForegroundColor Yellow
    }
}

# 4. Проверка баз данных
Write-Host "`n4. Проверка баз данных..." -ForegroundColor Yellow
$dbs = @("service.db", "1c_data.db", "normalized_data.db")
foreach ($db in $dbs) {
    if (Test-Path $db) {
        $size = (Get-Item $db -ErrorAction SilentlyContinue).Length / 1MB
        Write-Host "   ✅ $db ($([math]::Round($size, 2)) MB)" -ForegroundColor Green
    } else {
        Write-Host "   ⚠️  $db не найден (будет создан при первом запуске)" -ForegroundColor Yellow
    }
}

# 5. Проверка порта
Write-Host "`n5. Проверка порта 9999..." -ForegroundColor Yellow
$portInUse = netstat -ano | Select-String ":9999" | Select-Object -First 1
if ($portInUse) {
    $pid = ($portInUse -split '\s+')[-1]
    $proc = Get-Process -Id $pid -ErrorAction SilentlyContinue
    if ($proc) {
        Write-Host "   ⚠️  Порт 9999 занят процессом: $($proc.ProcessName) (PID: $pid)" -ForegroundColor Yellow
    } else {
        Write-Host "   ⚠️  Порт 9999 занят (PID: $pid)" -ForegroundColor Yellow
    }
} else {
    Write-Host "   ✅ Порт 9999 свободен" -ForegroundColor Green
}

# 6. Проверка backend (если запущен)
Write-Host "`n6. Проверка backend..." -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "http://localhost:9999/health" -Method GET -TimeoutSec 2
    Write-Host "   ✅ Backend работает" -ForegroundColor Green
    Write-Host "      Status: $($health.status)" -ForegroundColor White
    
    # Проверка API endpoints
    Write-Host "`n7. Проверка API endpoints..." -ForegroundColor Yellow
    try {
        $clients = Invoke-RestMethod -Uri "http://localhost:9999/api/clients" -Method GET -TimeoutSec 2
        Write-Host "   ✅ /api/clients работает ($($clients.clients.Count) клиентов)" -ForegroundColor Green
    } catch {
        Write-Host "   ⚠️  /api/clients не отвечает: $_" -ForegroundColor Yellow
    }
} catch {
    Write-Host "   ⚠️  Backend не запущен" -ForegroundColor Yellow
    Write-Host "      Запустите: .\start-backend.ps1" -ForegroundColor White
}

# Итог
Write-Host "`n" + ("="*60) -ForegroundColor Cyan
if ($allOk) {
    Write-Host "✅ Система готова к работе!" -ForegroundColor Green
} else {
    Write-Host "⚠️  Обнаружены проблемы. Исправьте их перед запуском." -ForegroundColor Yellow
}
Write-Host "`n"

