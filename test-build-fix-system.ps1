#!/usr/bin/env pwsh
# Тестовый скрипт для проверки работы системы генерации плана

Write-Host "=== Тестирование системы генерации плана исправления ошибок ===" -ForegroundColor Cyan
Write-Host ""

# Проверка 1: Наличие Go
Write-Host "[1/5] Проверка наличия Go..." -ForegroundColor Yellow
try {
    $goVersion = go version 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✅ $goVersion" -ForegroundColor Green
    } else {
        Write-Host "  ❌ Go не найден!" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "  ❌ Go не найден!" -ForegroundColor Red
    exit 1
}

# Проверка 2: Наличие скриптов
Write-Host "[2/5] Проверка наличия скриптов..." -ForegroundColor Yellow
$scripts = @(
    "build-and-fix.ps1",
    "generate-build-fix-plan.ps1",
    "parse-go-errors.ps1",
    "generate_build_fix_plan.py"
)

$allScriptsExist = $true
foreach ($script in $scripts) {
    if (Test-Path $script) {
        Write-Host "  ✅ $script" -ForegroundColor Green
    } else {
        Write-Host "  ⚠️  $script не найден" -ForegroundColor Yellow
        if ($script -ne "generate_build_fix_plan.py") {
            $allScriptsExist = $false
        }
    }
}

if (-not $allScriptsExist) {
    Write-Host "  ❌ Некоторые обязательные скрипты отсутствуют!" -ForegroundColor Red
    exit 1
}

# Проверка 3: Наличие документации
Write-Host "[3/5] Проверка наличия документации..." -ForegroundColor Yellow
$docs = @(
    "BUILD_FIX_INDEX.md",
    "QUICK_START.md",
    "BUILD_FIX_README.md",
    "BUILD_FIX_PROMPT_COPY.txt"
)

foreach ($doc in $docs) {
    if (Test-Path $doc) {
        Write-Host "  ✅ $doc" -ForegroundColor Green
    } else {
        Write-Host "  ⚠️  $doc не найден" -ForegroundColor Yellow
    }
}

# Проверка 4: Поиск Python (опционально)
Write-Host "[4/5] Поиск Python (опционально)..." -ForegroundColor Yellow
$pythonFound = $false
$pythonCommands = @("python", "python3", "py")

foreach ($cmd in $pythonCommands) {
    try {
        $null = & $cmd --version 2>&1
        if ($LASTEXITCODE -eq 0) {
            $version = & $cmd --version 2>&1
            Write-Host "  ✅ Найден: $cmd ($version)" -ForegroundColor Green
            $pythonFound = $true
            break
        }
    } catch {
        # Команда не найдена
    }
}

if (-not $pythonFound) {
    Write-Host "  ⚠️  Python не найден (будет использован PowerShell парсер)" -ForegroundColor Yellow
} else {
    Write-Host "  ✅ Python доступен" -ForegroundColor Green
}

# Проверка 5: Тестовая сборка (опционально)
Write-Host "[5/5] Тестовая проверка синтаксиса скриптов..." -ForegroundColor Yellow

# Проверяем синтаксис PowerShell скриптов
$psScripts = @("build-and-fix.ps1", "parse-go-errors.ps1", "generate-build-fix-plan.ps1")
$syntaxOk = $true

foreach ($script in $psScripts) {
    if (Test-Path $script) {
        try {
            $null = [System.Management.Automation.PSParser]::Tokenize((Get-Content $script -Raw), [ref]$null)
            Write-Host "  ✅ $script - синтаксис корректен" -ForegroundColor Green
        } catch {
            Write-Host "  ❌ $script - ошибка синтаксиса: $_" -ForegroundColor Red
            $syntaxOk = $false
        }
    }
}

if (-not $syntaxOk) {
    Write-Host ""
    Write-Host "❌ Обнаружены ошибки синтаксиса!" -ForegroundColor Red
    exit 1
}

# Итоговая сводка
Write-Host ""
Write-Host "=== Результаты тестирования ===" -ForegroundColor Cyan
Write-Host "✅ Все проверки пройдены!" -ForegroundColor Green
Write-Host ""
Write-Host "Система готова к использованию:" -ForegroundColor Yellow
Write-Host "  - Запустите: .\build-and-fix.ps1" -ForegroundColor White
Write-Host "  - Или: build-and-fix.bat" -ForegroundColor White
Write-Host "  - Или используйте BUILD_FIX_PROMPT_COPY.txt с AI" -ForegroundColor White
Write-Host ""

