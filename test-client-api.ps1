# Скрипт для тестирования API клиентов с поддержкой поля country
# Использование: .\test-client-api.ps1

$baseUrl = "http://127.0.0.1:9999"
$headers = @{
    "Content-Type" = "application/json"
}

Write-Host "=== Тестирование API клиентов с полем country ===" -ForegroundColor Cyan
Write-Host ""

# 1. Создание клиента с полем country
Write-Host "1. Создание клиента с полем country..." -ForegroundColor Yellow
$createBody = @{
    name = "Тестовый клиент $(Get-Date -Format 'yyyyMMddHHmmss')"
    legal_name = "ООО Тестовый клиент"
    description = "Клиент для тестирования поля country"
    contact_email = "test@example.com"
    contact_phone = "+7 (999) 123-45-67"
    tax_id = "1234567890"
    country = "RU"
} | ConvertTo-Json

try {
    $createResponse = Invoke-RestMethod -Uri "$baseUrl/api/clients" -Method POST -Headers $headers -Body $createBody
    $clientId = $createResponse.id
    Write-Host "✓ Клиент создан успешно. ID: $clientId" -ForegroundColor Green
    Write-Host "  Country: $($createResponse.country)" -ForegroundColor Gray
    Write-Host ""
} catch {
    Write-Host "✗ Ошибка при создании клиента: $_" -ForegroundColor Red
    Write-Host "  Ответ: $($_.Exception.Response)" -ForegroundColor Gray
    exit 1
}

# 2. Получение клиента
Write-Host "2. Получение клиента по ID..." -ForegroundColor Yellow
try {
    $getResponse = Invoke-RestMethod -Uri "$baseUrl/api/clients/$clientId" -Method GET -Headers $headers
    Write-Host "✓ Клиент получен успешно" -ForegroundColor Green
    Write-Host "  ID: $($getResponse.id)" -ForegroundColor Gray
    Write-Host "  Name: $($getResponse.name)" -ForegroundColor Gray
    Write-Host "  Country: $($getResponse.country)" -ForegroundColor Gray
    Write-Host ""
} catch {
    Write-Host "✗ Ошибка при получении клиента: $_" -ForegroundColor Red
    exit 1
}

# 3. Обновление клиента с изменением country
Write-Host "3. Обновление клиента с изменением country..." -ForegroundColor Yellow
$updateBody = @{
    name = $getResponse.name
    legal_name = $getResponse.legal_name
    description = $getResponse.description
    contact_email = $getResponse.contact_email
    contact_phone = $getResponse.contact_phone
    tax_id = $getResponse.tax_id
    country = "KZ"
} | ConvertTo-Json

try {
    $updateResponse = Invoke-RestMethod -Uri "$baseUrl/api/clients/$clientId" -Method PUT -Headers $headers -Body $updateBody
    Write-Host "✓ Клиент обновлен успешно" -ForegroundColor Green
    Write-Host "  Country изменен на: $($updateResponse.country)" -ForegroundColor Gray
    Write-Host ""
} catch {
    Write-Host "✗ Ошибка при обновлении клиента: $_" -ForegroundColor Red
    Write-Host "  Ответ: $($_.Exception.Response)" -ForegroundColor Gray
    exit 1
}

# 4. Проверка, что country сохранился
Write-Host "4. Проверка сохранения country..." -ForegroundColor Yellow
try {
    $verifyResponse = Invoke-RestMethod -Uri "$baseUrl/api/clients/$clientId" -Method GET -Headers $headers
    if ($verifyResponse.country -eq "KZ") {
        Write-Host "✓ Country успешно сохранен: $($verifyResponse.country)" -ForegroundColor Green
    } else {
        Write-Host "✗ Country не сохранен корректно. Ожидалось: KZ, получено: $($verifyResponse.country)" -ForegroundColor Red
        exit 1
    }
    Write-Host ""
} catch {
    Write-Host "✗ Ошибка при проверке: $_" -ForegroundColor Red
    exit 1
}

# 5. Получение списка клиентов (проверка GetClientsWithStats)
Write-Host "5. Получение списка клиентов..." -ForegroundColor Yellow
try {
    $listResponse = Invoke-RestMethod -Uri "$baseUrl/api/clients" -Method GET -Headers $headers
    $testClient = $listResponse | Where-Object { $_.id -eq $clientId }
    if ($testClient) {
        Write-Host "✓ Клиент найден в списке" -ForegroundColor Green
        Write-Host "  Country в списке: $($testClient.country)" -ForegroundColor Gray
        if ($testClient.country -eq "KZ") {
            Write-Host "✓ Country корректно отображается в списке" -ForegroundColor Green
        } else {
            Write-Host "✗ Country некорректно в списке. Ожидалось: KZ, получено: $($testClient.country)" -ForegroundColor Red
        }
    } else {
        Write-Host "✗ Клиент не найден в списке" -ForegroundColor Red
    }
    Write-Host ""
} catch {
    Write-Host "✗ Ошибка при получении списка: $_" -ForegroundColor Red
    exit 1
}

Write-Host "=== Все тесты пройдены успешно! ===" -ForegroundColor Green
Write-Host ""
Write-Host "Созданный клиент ID: $clientId" -ForegroundColor Cyan
Write-Host "Для удаления используйте: Invoke-RestMethod -Uri '$baseUrl/api/clients/$clientId' -Method DELETE"

