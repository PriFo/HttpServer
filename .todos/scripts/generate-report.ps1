# PowerShell скрипт для генерации отчета (Windows)
# Использование: .\.todos\scripts\generate-report.ps1

$ErrorActionPreference = "Stop"

Write-Host "📊 Генерация отчета TODO..." -ForegroundColor Cyan

# Проверка наличия Python
$python = Get-Command python -ErrorAction SilentlyContinue
if (-not $python) {
    $python = Get-Command python3 -ErrorAction SilentlyContinue
}

if (-not $python) {
    Write-Host "❌ Python не найден! Установите Python для работы системы." -ForegroundColor Red
    exit 1
}

# Запуск Python скрипта
$scriptPath = Join-Path $PSScriptRoot "generate-report.py"
& $python.Path $scriptPath

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Отчет сгенерирован!" -ForegroundColor Green
    Write-Host "📄 Откройте TODO_REPORT.md для просмотра" -ForegroundColor Yellow
} else {
    Write-Host "❌ Ошибка при генерации отчета!" -ForegroundColor Red
    exit $LASTEXITCODE
}


