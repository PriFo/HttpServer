# PowerShell скрипт для запуска Chaos Monkey тестов на Windows
# Решает проблемы с Python PATH и WSL

$ErrorActionPreference = "Continue"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Chaos Monkey Testing - Windows Launcher" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Поиск Python
$pythonCmd = $null
$pythonPaths = @(
    "python",
    "python3",
    "py",
    "C:\Users\eugin\.local\bin\python3.11.exe",
    "$env:LOCALAPPDATA\Programs\Python\Python*\python.exe",
    "$env:ProgramFiles\Python*\python.exe"
)

if (-not $pythonCmd) {
    Write-Host "Поиск Python..." -ForegroundColor Yellow
    foreach ($path in $pythonPaths) {
        try {
            if ($path -like "*\*") {
                # Glob pattern
                $found = Get-ChildItem -Path $path -ErrorAction SilentlyContinue | Select-Object -First 1
                if ($found) {
                    $pythonCmd = $found.FullName
                    break
                }
            } else {
                $result = & $path --version 2>&1
                if ($LASTEXITCODE -eq 0 -or $result -like "Python*") {
                    $pythonCmd = $path
                    break
                }
            }
        } catch {
            continue
        }
    }
}

if (-not $pythonCmd) {
    Write-Host "❌ Python не найден" -ForegroundColor Red
    Write-Host ""
    Write-Host "Установите Python одним из способов:"
    Write-Host "  1. Скачайте с python.org"
    Write-Host "  2. Используйте: winget install Python.Python.3.11"
    Write-Host "  3. Или используйте WSL: wsl bash chaos_tests/run_all_tests.sh"
    exit 1
}

Write-Host "✅ Найден Python: $pythonCmd" -ForegroundColor Green

# Проверка зависимостей
Write-Host ""
Write-Host "Проверка зависимостей..." -ForegroundColor Yellow
try {
    $requests = & $pythonCmd -c "import requests; print('OK')" 2>&1
    if ($requests -eq "OK") {
        Write-Host "✅ requests установлен" -ForegroundColor Green
    } else {
        Write-Host "⚠️ requests не установлен, устанавливаю..." -ForegroundColor Yellow
        & $pythonCmd -m pip install requests --quiet
    }
} catch {
    Write-Host "⚠️ Не удалось проверить requests" -ForegroundColor Yellow
}

try {
    $psutil = & $pythonCmd -c "import psutil; print('OK')" 2>&1
    if ($psutil -eq "OK") {
        Write-Host "✅ psutil установлен" -ForegroundColor Green
    } else {
        Write-Host "⚠️ psutil не установлен (мониторинг ресурсов будет недоступен)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "⚠️ psutil не установлен" -ForegroundColor Yellow
}

# Проверка сервера
Write-Host ""
Write-Host "Проверка сервера..." -ForegroundColor Yellow
$serverRunning = $false
try {
    $response = Invoke-WebRequest -Uri "http://localhost:9999/api/config" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
    if ($response.StatusCode -eq 200) {
        Write-Host "✅ Сервер доступен (HTTP 200)" -ForegroundColor Green
        $serverRunning = $true
    } elseif ($response.StatusCode -eq 502) {
        Write-Host "⚠️ Сервер возвращает 502 Bad Gateway" -ForegroundColor Yellow
        Write-Host "   Возможные причины:" -ForegroundColor Yellow
        Write-Host "   - Проблемы с конфигурацией сервера" -ForegroundColor Yellow
        Write-Host "   - Недоступность базы данных" -ForegroundColor Yellow
        Write-Host "   - Ошибки инициализации" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "   Тесты будут запущены, но могут провалиться" -ForegroundColor Yellow
    }
} catch {
    $errorMsg = $_.Exception.Message
    if ($errorMsg -like "*Connection refused*" -or $errorMsg -like "*не удалось*") {
        Write-Host "❌ Сервер не запущен или недоступен" -ForegroundColor Red
        Write-Host ""
        Write-Host "Запустите сервер:" -ForegroundColor Yellow
        Write-Host "  .\httpserver_no_gui.exe" -ForegroundColor Cyan
        Write-Host ""
        $continue = Read-Host "Продолжить тесты? (y/n)"
        if ($continue -ne "y") {
            exit 1
        }
    } else {
        Write-Host "⚠️ Ошибка при проверке сервера: $errorMsg" -ForegroundColor Yellow
    }
}

# Определяем тест
$testName = "all"
if ($args.Count -gt 0) {
    $testName = $args[0]
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Запуск тестов: $testName" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Запуск тестов
$scriptPath = Join-Path $PSScriptRoot "chaos_monkey.py"
if (-not (Test-Path $scriptPath)) {
    Write-Host "❌ Не найден скрипт: $scriptPath" -ForegroundColor Red
    exit 1
}

try {
    $testArgs = @("--test", $testName)
    if ($args -contains "--quick") {
        $testArgs += "--quick"
    }
    
    & $pythonCmd $scriptPath @testArgs
    $exitCode = $LASTEXITCODE
    
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    if ($exitCode -eq 0) {
        Write-Host "✅ Тесты завершены успешно" -ForegroundColor Green
    } else {
        Write-Host "⚠️ Некоторые тесты провалены" -ForegroundColor Yellow
        Write-Host "Проверьте отчеты в chaos_tests/reports/" -ForegroundColor Yellow
    }
    Write-Host "========================================" -ForegroundColor Cyan
    
    exit $exitCode
} catch {
    Write-Host "❌ Ошибка при запуске тестов: $_" -ForegroundColor Red
    exit 1
}

