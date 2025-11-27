# Скрипт для запуска бэкенда с правильной кодировкой UTF-8
# Использование: .\start-backend-simple.ps1

# Устанавливаем UTF-8 кодировку для вывода
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$env:CHCP = "65001"

Write-Host "=== ЗАПУСК БЭКЕНДА ===" -ForegroundColor Green
Write-Host ""

# Проверяем, не запущен ли уже сервер
$existingProcess = Get-Process -Name "go" -ErrorAction SilentlyContinue | Where-Object {
    $_.CommandLine -like "*main_no_gui.go*"
}

if ($existingProcess) {
    Write-Host "ВНИМАНИЕ: Обнаружен запущенный процесс бэкенда (PID: $($existingProcess.Id))" -ForegroundColor Yellow
    $response = Read-Host "Остановить существующий процесс? (y/n)"
    if ($response -eq 'y') {
        Stop-Process -Id $existingProcess.Id -Force
        Write-Host "Процесс остановлен" -ForegroundColor Green
        Start-Sleep -Seconds 2
    } else {
        Write-Host "Выход..." -ForegroundColor Yellow
        exit 0
    }
}

# Устанавливаем переменные окружения, если нужны
if (-not $env:ARLIAI_API_KEY) {
    Write-Host "ВНИМАНИЕ: ARLIAI_API_KEY не установлен (AI функции будут недоступны)" -ForegroundColor Yellow
}

# Запускаем сервер с правильными флагами
Write-Host "Запуск сервера на порту 9999..." -ForegroundColor Cyan
Write-Host ""

# Запускаем с тегом no_gui
go run -tags no_gui main_no_gui.go

# Если скрипт завершился, показываем сообщение
Write-Host ""
Write-Host "=== СЕРВЕР ОСТАНОВЛЕН ===" -ForegroundColor Red

