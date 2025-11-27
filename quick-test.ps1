# Быстрый тест API клиентов с полем country
# Использование: .\quick-test.ps1

$baseUrl = "http://127.0.0.1:9999"
$headers = @{"Content-Type" = "application/json"}

Write-Host "Быстрый тест API клиентов" -ForegroundColor Cyan
Write-Host ""

# Проверка доступности сервера
Write-Host "Проверка доступности сервера..." -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "$baseUrl/api/clients" -Method GET -Headers $headers -TimeoutSec 2
    Write-Host "✓ Сервер доступен" -ForegroundColor Green
} catch {
    Write-Host "✗ Сервер недоступен на $baseUrl" -ForegroundColor Red
    Write-Host "  Убедитесь, что сервер запущен: go run main.go" -ForegroundColor Yellow
    exit 1
}

# Создание клиента
Write-Host "`nСоздание клиента с country=RU..." -ForegroundColor Yellow
$createBody = @{
    name = "Тест $(Get-Date -Format 'HHmmss')"
    country = "RU"
} | ConvertTo-Json

try {
    $client = Invoke-RestMethod -Uri "$baseUrl/api/clients" -Method POST -Headers $headers -Body $createBody
    Write-Host "✓ Клиент создан. ID: $($client.id), Country: $($client.country)" -ForegroundColor Green
    
    # Обновление country
    Write-Host "`nОбновление country на KZ..." -ForegroundColor Yellow
    $updateBody = @{
        name = $client.name
        country = "KZ"
    } | ConvertTo-Json
    
    $updated = Invoke-RestMethod -Uri "$baseUrl/api/clients/$($client.id)" -Method PUT -Headers $headers -Body $updateBody
    Write-Host "✓ Клиент обновлен. Country: $($updated.country)" -ForegroundColor Green
    
    # Проверка
    if ($updated.country -eq "KZ") {
        Write-Host "`n✓ Тест пройден успешно!" -ForegroundColor Green
    } else {
        Write-Host "`n✗ Тест не пройден. Country должен быть KZ, получен: $($updated.country)" -ForegroundColor Red
    }
} catch {
    Write-Host "✗ Ошибка: $_" -ForegroundColor Red
    Write-Host $_.Exception.Response -ForegroundColor Gray
}

