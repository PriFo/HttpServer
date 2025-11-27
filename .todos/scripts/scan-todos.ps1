# PowerShell скрипт для сканирования TODO (Windows)
# Использование: .\.todos\scripts\scan-todos.ps1

$ErrorActionPreference = "Stop"

Write-Host "🚀 Запуск автоматизированного сканирования TODO..." -ForegroundColor Cyan

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
$scriptPath = Join-Path $PSScriptRoot "scan-todos.py"
& $python.Path $scriptPath

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Сканирование завершено успешно!" -ForegroundColor Green
} else {
    Write-Host "❌ Ошибка при сканировании!" -ForegroundColor Red
    exit $LASTEXITCODE
}


