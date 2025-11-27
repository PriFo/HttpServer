# Скрипт для запуска всех E2E тестов
# Использование: .\scripts\run-all-e2e-tests.ps1 [--headed] [--ui] [--debug] [--grep "pattern"]

param(
    [switch]$Headed,
    [switch]$UI,
    [switch]$Debug,
    [string]$Grep = "",
    [string]$Project = "frontend"
)

Write-Host "🚀 Запуск всех E2E тестов..." -ForegroundColor Green
Write-Host ""

# Проверяем, что мы в правильной директории
if (-not (Test-Path "package.json")) {
    Write-Host "❌ Ошибка: package.json не найден. Запустите скрипт из корня проекта." -ForegroundColor Red
    exit 1
}

# Проверяем, что фронтенд существует
if (-not (Test-Path $Project)) {
    Write-Host "❌ Ошибка: Директория $Project не найдена." -ForegroundColor Red
    exit 1
}

# Переходим в директорию фронтенда
Push-Location $Project

try {
    # Проверяем наличие Playwright
    $playwrightInstalled = npm list @playwright/test 2>$null
    if (-not $playwrightInstalled) {
        Write-Host "📦 Установка Playwright..." -ForegroundColor Yellow
        npm install
        npx playwright install
    }

    # Формируем команду
    $command = "npx playwright test tests/e2e"
    
    if ($UI) {
        $command += " --ui"
        Write-Host "🎨 Запуск в UI режиме..." -ForegroundColor Cyan
    } elseif ($Debug) {
        $command += " --debug"
        Write-Host "🐛 Запуск в режиме отладки..." -ForegroundColor Cyan
    } elseif ($Headed) {
        $command += " --headed"
        Write-Host "👀 Запуск в видимом режиме..." -ForegroundColor Cyan
    }
    
    if ($Grep) {
        $command += " --grep `"$Grep`""
        Write-Host "🔍 Фильтр: $Grep" -ForegroundColor Cyan
    }

    Write-Host ""
    Write-Host "Выполняется: $command" -ForegroundColor Gray
    Write-Host ""

    # Запускаем тесты
    Invoke-Expression $command
    
    $exitCode = $LASTEXITCODE
    
    if ($exitCode -eq 0) {
        Write-Host ""
        Write-Host "✅ Все тесты прошли успешно!" -ForegroundColor Green
    } else {
        Write-Host ""
        Write-Host "❌ Некоторые тесты провалились. Код выхода: $exitCode" -ForegroundColor Red
        Write-Host "📊 Просмотр отчета: npx playwright show-report" -ForegroundColor Yellow
    }
    
    exit $exitCode
} finally {
    Pop-Location
}

