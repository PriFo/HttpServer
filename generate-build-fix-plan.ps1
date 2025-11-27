#!/usr/bin/env pwsh
# PowerShell обертка для генерации плана исправления ошибок
# Пытается использовать Python скрипт, если не получается - использует PowerShell парсер

Write-Host "=== Генерация плана исправления ошибок ===" -ForegroundColor Cyan

# Проверяем наличие файла с ошибками
if (-not (Test-Path "build_errors_full.log")) {
    Write-Host "❌ Файл build_errors_full.log не найден!" -ForegroundColor Red
    Write-Host "Сначала выполните сборку проекта: go build ./... 2>&1 | Tee-Object -FilePath build_errors_full.log" -ForegroundColor Yellow
    exit 1
}

# Пытаемся найти Python
$pythonCmd = $null
$pythonCommands = @("python", "python3", "py")

Write-Host "Поиск Python..." -ForegroundColor Yellow
foreach ($cmd in $pythonCommands) {
    try {
        $null = & $cmd --version 2>&1
        if ($LASTEXITCODE -eq 0) {
            $pythonCmd = $cmd
            Write-Host "✅ Найден Python: $cmd" -ForegroundColor Green
            break
        }
    } catch {
        # Команда не найдена, пробуем следующую
    }
}

# Пробуем использовать Python скрипт
if ($pythonCmd -and (Test-Path "generate_build_fix_plan.py")) {
    Write-Host "`nЗапуск Python скрипта..." -ForegroundColor Yellow
    try {
        & $pythonCmd generate_build_fix_plan.py
        if ($LASTEXITCODE -eq 0) {
            Write-Host "`n✅ План успешно создан с помощью Python!" -ForegroundColor Green
            exit 0
        }
    } catch {
        Write-Host "⚠️  Ошибка при запуске Python скрипта: $_" -ForegroundColor Yellow
        Write-Host "Переключаюсь на PowerShell парсер..." -ForegroundColor Yellow
    }
}

# Используем PowerShell парсер
Write-Host "`nИспользование PowerShell парсера..." -ForegroundColor Yellow
if (Test-Path "parse-go-errors.ps1") {
    try {
        & .\parse-go-errors.ps1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "`n✅ План успешно создан с помощью PowerShell!" -ForegroundColor Green
            exit 0
        }
    } catch {
        Write-Host "❌ Ошибка при запуске PowerShell парсера: $_" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "❌ Файл parse-go-errors.ps1 не найден!" -ForegroundColor Red
    exit 1
}
