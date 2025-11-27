# Скрипт для запуска backend сервера
# Использование: .\start-backend.ps1

Write-Host "`n╔══════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║           🚀 ЗАПУСК BACKEND СЕРВЕРА                      ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════════╝" -ForegroundColor Cyan

# Проверка, что мы в правильной директории
if (-not (Test-Path "cmd/server/main.go")) {
    Write-Host "`n❌ Ошибка: файл cmd/server/main.go не найден" -ForegroundColor Red
    Write-Host "Убедитесь, что вы находитесь в корневой директории проекта" -ForegroundColor Yellow
    exit 1
}

# Проверка порта 9999
Write-Host "`n📡 Проверка порта 9999..." -ForegroundColor Yellow
$portInUse = netstat -ano | Select-String ":9999" | Select-Object -First 1
if ($portInUse) {
    Write-Host "⚠️  Порт 9999 уже занят!" -ForegroundColor Yellow
    $pid = ($portInUse -split '\s+')[-1]
    Write-Host "   Процесс PID: $pid" -ForegroundColor White
    $response = Read-Host "Остановить процесс и продолжить? (y/n)"
    if ($response -eq "y" -or $response -eq "Y") {
        try {
            Stop-Process -Id $pid -Force -ErrorAction Stop
            Write-Host "✅ Процесс остановлен" -ForegroundColor Green
            Start-Sleep -Seconds 2
        } catch {
            Write-Host "❌ Не удалось остановить процесс: $_" -ForegroundColor Red
            exit 1
        }
    } else {
        Write-Host "Отмена запуска" -ForegroundColor Yellow
        exit 0
    }
} else {
    Write-Host "✅ Порт 9999 свободен" -ForegroundColor Green
}

# Проверка Go
Write-Host "`n🔍 Проверка Go..." -ForegroundColor Yellow
$goVersion = go version 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ $goVersion" -ForegroundColor Green
} else {
    Write-Host "❌ Go не установлен или не найден в PATH" -ForegroundColor Red
    exit 1
}

# Проверка компиляции
Write-Host "`n🔨 Проверка компиляции..." -ForegroundColor Yellow
$buildOutput = go build ./cmd/server 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Компиляция успешна" -ForegroundColor Green
} else {
    Write-Host "❌ Ошибка компиляции:" -ForegroundColor Red
    $buildOutput | Select-String "error:" | ForEach-Object { Write-Host "   $_" -ForegroundColor Red }
    exit 1
}

# Создание необходимых директорий
Write-Host "`n📁 Проверка директорий..." -ForegroundColor Yellow
$dirs = @("data", "data/uploads", "data/backups", "data/temp")
foreach ($dir in $dirs) {
    if (-not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
        Write-Host "   ✅ Создана директория: $dir" -ForegroundColor Green
    } else {
        Write-Host "   ✅ Директория существует: $dir" -ForegroundColor Green
    }
}

# Установка переменных окружения
Write-Host "`n⚙️  Настройка окружения..." -ForegroundColor Yellow
$env:GIN_MODE = "release"
Write-Host "   ✅ GIN_MODE=release" -ForegroundColor Green

# Запуск сервера
Write-Host "`n🚀 Запуск backend сервера..." -ForegroundColor Cyan
Write-Host "   Порт: 9999" -ForegroundColor White
Write-Host "   API: http://localhost:9999" -ForegroundColor White
Write-Host "   Health: http://localhost:9999/health" -ForegroundColor White
Write-Host "`n   Для остановки нажмите Ctrl+C" -ForegroundColor Yellow
Write-Host "`n" + ("="*60) -ForegroundColor Cyan

# Запуск
go run cmd/server/main.go

