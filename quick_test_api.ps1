# Быстрая проверка API /api/counterparties/all
# Использование: .\quick_test_api.ps1 [client_id]

param(
    [int]$ClientID = 1
)

Write-Host "`n=== Быстрая проверка API /api/counterparties/all ===" -ForegroundColor Cyan

$baseUrl = "http://localhost:9999"
$endpoint = "$baseUrl/api/counterparties/all"

# Тест 1: Базовый запрос
Write-Host "`n1. Базовый запрос..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$endpoint?client_id=$ClientID&limit=5" -Method GET
    Write-Host "   ✓ Успешно! Total: $($response.total), Returned: $($response.counterparties.Count)" -ForegroundColor Green
    Write-Host "   Stats: DBs=$($response.stats.databases_processed), Projects=$($response.stats.projects_processed), Time=$($response.stats.processing_time_ms)ms" -ForegroundColor Gray
} catch {
    Write-Host "   ✗ Ошибка: $($_.Exception.Message)" -ForegroundColor Red
}

# Тест 2: Фильтр по источнику
Write-Host "`n2. Фильтр source=database..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$endpoint?client_id=$ClientID&source=database&limit=3" -Method GET
    Write-Host "   ✓ Успешно! Total: $($response.total)" -ForegroundColor Green
} catch {
    Write-Host "   ✗ Ошибка: $($_.Exception.Message)" -ForegroundColor Red
}

# Тест 3: Сортировка
Write-Host "`n3. Сортировка sort_by=name..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$endpoint?client_id=$ClientID&sort_by=name&order=asc&limit=3" -Method GET
    Write-Host "   ✓ Успешно! Returned: $($response.counterparties.Count)" -ForegroundColor Green
} catch {
    Write-Host "   ✗ Ошибка: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n✅ Проверка завершена!" -ForegroundColor Green
