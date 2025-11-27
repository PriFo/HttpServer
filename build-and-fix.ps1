#!/usr/bin/env pwsh
# Скрипт для автоматической сборки проекта и генерации плана исправления ошибок

# Проверяем наличие Go
Write-Host "=== Проверка окружения ===" -ForegroundColor Cyan
try {
    $goVersion = go version 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Go не найден! Установите Go и добавьте его в PATH." -ForegroundColor Red
        exit 1
    }
    Write-Host "✅ $goVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ Go не найден! Установите Go и добавьте его в PATH." -ForegroundColor Red
    exit 1
}

Write-Host "`n=== Сборка проекта Go ===" -ForegroundColor Cyan

# Собираем проект и сохраняем ошибки
Write-Host "Выполняю сборку проекта..." -ForegroundColor Yellow
$buildOutput = go build ./... 2>&1
$buildExitCode = $LASTEXITCODE

# Сохраняем вывод в файл
$errorLogFile = "build_errors_full.log"
$buildOutput | Out-File -FilePath $errorLogFile -Encoding UTF8

# Показываем краткую статистику ошибок
$errorCount = ($buildOutput | Select-String -Pattern "\.go:\d+:\d+:").Count
if ($errorCount -gt 0) {
    Write-Host "Найдено ошибок: $errorCount" -ForegroundColor Yellow
}

if ($buildExitCode -eq 0) {
    Write-Host "`n✅ Проект собрался без ошибок!" -ForegroundColor Green
    Write-Host "Никаких действий не требуется." -ForegroundColor Green
    exit 0
}

Write-Host "`n❌ Обнаружены ошибки компиляции" -ForegroundColor Red
Write-Host "Ошибки сохранены в build_errors_full.log" -ForegroundColor Yellow

# Пытаемся найти Python
$pythonCmd = $null
$pythonCommands = @("python", "python3", "py")

Write-Host "`n=== Поиск Python ===" -ForegroundColor Cyan
foreach ($cmd in $pythonCommands) {
    try {
        $version = & $cmd --version 2>&1
        if ($LASTEXITCODE -eq 0) {
            $pythonCmd = $cmd
            Write-Host "✅ Найден Python: $cmd ($version)" -ForegroundColor Green
            break
        }
    } catch {
        # Команда не найдена, пробуем следующую
    }
}

# Генерируем план
if ($pythonCmd) {
    Write-Host "`n=== Генерация плана с помощью Python ===" -ForegroundColor Cyan
    try {
        & $pythonCmd generate_build_fix_plan.py
        if ($LASTEXITCODE -eq 0) {
            Write-Host "`n✅ План успешно создан!" -ForegroundColor Green
            exit 0
        }
    } catch {
        Write-Host "❌ Ошибка при запуске Python скрипта: $_" -ForegroundColor Red
    }
}

# Пробуем PowerShell скрипт
Write-Host "`n=== Попытка использовать PowerShell скрипт ===" -ForegroundColor Cyan
if (Test-Path "generate-build-fix-plan.ps1") {
    try {
        & .\generate-build-fix-plan.ps1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "`n✅ План успешно создан!" -ForegroundColor Green
            exit 0
        }
    } catch {
        Write-Host "❌ Ошибка при запуске PowerShell скрипта: $_" -ForegroundColor Red
    }
}

# Пробуем парсер напрямую
Write-Host "`n=== Попытка использовать PowerShell парсер ===" -ForegroundColor Cyan
if (Test-Path "parse-go-errors.ps1") {
    try {
        & .\parse-go-errors.ps1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "`n✅ План успешно создан!" -ForegroundColor Green
            exit 0
        }
    } catch {
        Write-Host "❌ Ошибка при запуске парсера: $_" -ForegroundColor Red
    }
}

Write-Host "`n❌ Не удалось сгенерировать план автоматически" -ForegroundColor Red
Write-Host "Попробуйте запустить вручную:" -ForegroundColor Yellow
Write-Host "  - python generate_build_fix_plan.py" -ForegroundColor Yellow
Write-Host "  - .\generate-build-fix-plan.ps1" -ForegroundColor Yellow
Write-Host "  - .\parse-go-errors.ps1" -ForegroundColor Yellow
exit 1
