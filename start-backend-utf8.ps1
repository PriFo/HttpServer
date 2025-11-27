# Скрипт для запуска основного бэкенд-сервера с правильной кодировкой UTF-8
# Использование: .\start-backend-utf8.ps1

# Устанавливаем UTF-8 кодировку
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
chcp 65001 | Out-Null

Write-Host "=== ЗАПУСК HTTP СЕРВЕРА ===" -ForegroundColor Green
Write-Host ""

# Проверяем наличие cmd/server/main.go
if (-not (Test-Path "cmd\server\main.go")) {
    Write-Host "ОШИБКА: Файл cmd\server\main.go не найден!" -ForegroundColor Red
    Write-Host "Убедитесь, что вы находитесь в корневой директории проекта" -ForegroundColor Yellow
    exit 1
}

# Проверяем, не запущен ли уже сервер на порту 9999
$portInUse = Test-NetConnection -ComputerName localhost -Port 9999 -InformationLevel Quiet -WarningAction SilentlyContinue
if ($portInUse) {
    Write-Host "ВНИМАНИЕ: Порт 9999 уже используется!" -ForegroundColor Yellow
    Write-Host "Остановите существующий сервер или используйте другой порт" -ForegroundColor Yellow
    $response = Read-Host "Попытаться найти и остановить процесс? (y/n)"
    if ($response -eq 'y') {
        $processes = Get-NetTCPConnection -LocalPort 9999 -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess -Unique
        foreach ($pid in $processes) {
            Write-Host "Остановка процесса с PID: $pid" -ForegroundColor Yellow
            Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue
        }
        Start-Sleep -Seconds 2
    } else {
        exit 0
    }
}

# Проверяем переменные окружения
if (-not $env:ARLIAI_API_KEY) {
    Write-Host "ВНИМАНИЕ: ARLIAI_API_KEY не установлен" -ForegroundColor Yellow
    Write-Host "AI-функции будут недоступны, но сервер запустится" -ForegroundColor Yellow
    Write-Host ""
}

# Запускаем сервер
Write-Host "Запуск HTTP сервера..." -ForegroundColor Cyan
Write-Host "Порт: 9999" -ForegroundColor Cyan
Write-Host "Для остановки нажмите Ctrl+C" -ForegroundColor Gray
Write-Host ""

try {
    # Используем основной main.go из cmd/server/
    go run cmd/server/main.go
} catch {
    Write-Host ""
    Write-Host "ОШИБКА при запуске сервера: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== СЕРВЕР ОСТАНОВЛЕН ===" -ForegroundColor Red

