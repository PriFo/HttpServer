# Скрипт для запуска E2E тестов (PowerShell)
# Использование: .\scripts\run-e2e-tests.ps1 [опции]

param(
    [string]$TestFile = "",
    [string]$Browser = "chromium",
    [switch]$Headed,
    [switch]$Debug,
    [switch]$UI,
    [switch]$Help
)

# Функция вывода справки
function Show-Help {
    Write-Host "Использование: .\scripts\run-e2e-tests.ps1 [опции]"
    Write-Host ""
    Write-Host "Опции:"
    Write-Host "  -TestFile <путь>    Запустить конкретный тест"
    Write-Host "  -Browser <name>      Браузер (chromium, firefox, webkit)"
    Write-Host "  -Headed              Запустить в видимом режиме"
    Write-Host "  -Debug               Запустить в режиме отладки"
    Write-Host "  -UI                  Запустить с UI"
    Write-Host "  -Help                Показать эту справку"
}

if ($Help) {
    Show-Help
    exit 0
}

Write-Host "🚀 Запуск E2E тестов" -ForegroundColor Green
Write-Host ""

# Проверка зависимостей
Write-Host "Проверка зависимостей..." -ForegroundColor Yellow

# Проверяем, что npx доступен
try {
    $null = Get-Command npx -ErrorAction Stop
} catch {
    Write-Host "❌ npx не найден. Установите Node.js" -ForegroundColor Red
    exit 1
}

# Проверяем, что бэкенд запущен
Write-Host "Проверка бэкенда..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://127.0.0.1:9999/health" -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop
    Write-Host "✅ Бэкенд доступен" -ForegroundColor Green
} catch {
    Write-Host "❌ Бэкенд не доступен на http://127.0.0.1:9999" -ForegroundColor Red
    Write-Host "💡 Запустите бэкенд: docker-compose up -d backend" -ForegroundColor Yellow
    exit 1
}

# Проверяем, что фронтенд запущен
Write-Host "Проверка фронтенда..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:3000" -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop
    Write-Host "✅ Фронтенд доступен" -ForegroundColor Green
} catch {
    Write-Host "❌ Фронтенд не доступен на http://localhost:3000" -ForegroundColor Red
    Write-Host "💡 Запустите фронтенд: cd frontend && npm run dev" -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# Формируем команду
$cmd = "npx playwright test"

if ($TestFile) {
    $cmd += " $TestFile"
}

$cmd += " --project=$Browser"

if ($Headed) {
    $cmd += " --headed"
}

if ($Debug) {
    $cmd += " --debug"
}

if ($UI) {
    $cmd += " --ui"
}

Write-Host "Выполняем: $cmd" -ForegroundColor Green
Write-Host ""

# Запускаем тесты
Push-Location frontend
try {
    Invoke-Expression $cmd
    $exitCode = $LASTEXITCODE
    
    if ($exitCode -eq 0) {
        Write-Host ""
        Write-Host "✅ Все тесты прошли успешно!" -ForegroundColor Green
    } else {
        Write-Host ""
        Write-Host "❌ Некоторые тесты провалились" -ForegroundColor Red
    }
    
    exit $exitCode
} finally {
    Pop-Location
}

