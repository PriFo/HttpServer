# PowerShell скрипт для тестирования endpoints КПВЭД после перезапуска сервера

$BaseUrl = "http://localhost:9999"

Write-Host "=== Тестирование endpoints КПВЭД ===" -ForegroundColor Cyan
Write-Host ""

# Тест 1: Статистика КПВЭД
Write-Host "1. Проверка GET /api/kpved/stats" -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$BaseUrl/api/kpved/stats" -Method Get -ErrorAction Stop
    Write-Host "✓ Endpoint доступен" -ForegroundColor Green
    Write-Host "Ответ:" -ForegroundColor Gray
    $response | ConvertTo-Json -Depth 10
    
    if ($response.total -gt 0) {
        Write-Host "✓ Всего кодов КПВЭД: $($response.total)" -ForegroundColor Green
        Write-Host "✓ Данные присутствуют в базе" -ForegroundColor Green
        
        if ($response.levels_distribution) {
            Write-Host "`nРаспределение по уровням:" -ForegroundColor Cyan
            foreach ($level in $response.levels_distribution) {
                Write-Host "  Уровень $($level.level): $($level.count) записей" -ForegroundColor Gray
            }
        }
    } else {
        Write-Host "⚠ Данные отсутствуют в базе" -ForegroundColor Yellow
    }
} catch {
    Write-Host "✗ Endpoint недоступен: $_" -ForegroundColor Red
    Write-Host "  (Возможно, сервер не запущен или endpoint не зарегистрирован)" -ForegroundColor Yellow
}
Write-Host ""

Write-Host "=== Тестирование завершено ===" -ForegroundColor Cyan

