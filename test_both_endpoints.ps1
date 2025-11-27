# Тестирование обоих endpoint'ов контрагентов
$backendUrl = "http://127.0.0.1:9999"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Тестирование API контрагентов" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Получение списка клиентов
Write-Host "Получение списка клиентов..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$backendUrl/api/clients" -UseBasicParsing
    $clients = $response.Content | ConvertFrom-Json
    $clientId = $clients[0].id
    Write-Host "Используем клиента ID: $clientId" -ForegroundColor Green
} catch {
    Write-Host "Ошибка: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Тест 1: /api/counterparties/normalized
Write-Host "`n1. Тест /api/counterparties/normalized..." -ForegroundColor Yellow
try {
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=$clientId"
    $response = Invoke-WebRequest -Uri $uri.ToString() -UseBasicParsing
    $data = $response.Content | ConvertFrom-Json
    Write-Host "  [OK] Endpoint работает" -ForegroundColor Green
    Write-Host "  Контрагентов: $($data.counterparties.Count), Всего: $($data.total)" -ForegroundColor Gray
} catch {
    Write-Host "  [ERROR] $($_.Exception.Message)" -ForegroundColor Red
}

# Тест 2: /api/counterparties/all (используется фронтендом)
Write-Host "`n2. Тест /api/counterparties/all (используется фронтендом)..." -ForegroundColor Yellow
try {
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/all")
    $uri.Query = "client_id=$clientId&offset=0&limit=20"
    $response = Invoke-WebRequest -Uri $uri.ToString() -UseBasicParsing
    $data = $response.Content | ConvertFrom-Json
    Write-Host "  [OK] Endpoint работает" -ForegroundColor Green
    Write-Host "  Контрагентов: $($data.counterparties.Count), Всего: $($data.total)" -ForegroundColor Gray
    if ($data.stats) {
        Write-Host "  Время обработки: $($data.stats.processing_time_ms)ms" -ForegroundColor Gray
    }
} catch {
    Write-Host "  [ERROR] $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "Тестирование завершено" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

