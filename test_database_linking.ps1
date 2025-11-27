# Тест функциональности автоматической привязки баз данных
# Проверяет API endpoints и логику работы

Write-Host "=== Тест функциональности привязки баз данных ===" -ForegroundColor Cyan
Write-Host ""

$baseUrl = "http://127.0.0.1:9999"
$testClientId = 1  # Замените на реальный ID клиента

# Функция для выполнения HTTP запросов
function Invoke-ApiRequest {
    param(
        [string]$Method,
        [string]$Url,
        [object]$Body = $null
    )
    
    try {
        $headers = @{
            "Content-Type" = "application/json"
        }
        
        $params = @{
            Method = $Method
            Uri = $Url
            Headers = $headers
        }
        
        if ($Body) {
            $params.Body = ($Body | ConvertTo-Json -Depth 10)
        }
        
        $response = Invoke-RestMethod @params
        return @{
            Success = $true
            Data = $response
        }
    }
    catch {
        return @{
            Success = $false
            Error = $_.Exception.Message
            StatusCode = $_.Exception.Response.StatusCode.value__
        }
    }
}

# Тест 1: Получение списка баз данных клиента
Write-Host "Тест 1: Получение списка баз данных клиента" -ForegroundColor Yellow
$result = Invoke-ApiRequest -Method "GET" -Url "$baseUrl/api/clients/$testClientId/databases"
if ($result.Success) {
    Write-Host "✓ Успешно получен список баз данных" -ForegroundColor Green
    Write-Host "  Всего баз: $($result.Data.Count)" -ForegroundColor Gray
    $unlinked = $result.Data | Where-Object { $_.project_id -eq $null -or $_.project_id -eq 0 }
    Write-Host "  Непривязанных баз: $($unlinked.Count)" -ForegroundColor Gray
    if ($unlinked.Count -gt 0) {
        Write-Host "  Непривязанные базы:" -ForegroundColor Gray
        foreach ($db in $unlinked) {
            Write-Host "    - $($db.name) (ID: $($db.id))" -ForegroundColor Gray
        }
    }
} else {
    Write-Host "✗ Ошибка: $($result.Error)" -ForegroundColor Red
}
Write-Host ""

# Тест 2: Получение статистики клиента (проверка непривязанных баз)
Write-Host "Тест 2: Получение статистики клиента" -ForegroundColor Yellow
$result = Invoke-ApiRequest -Method "GET" -Url "$baseUrl/api/clients/$testClientId/statistics"
if ($result.Success) {
    Write-Host "✓ Успешно получена статистика" -ForegroundColor Green
    Write-Host "  Всего баз данных: $($result.Data.total_databases)" -ForegroundColor Gray
    if ($result.Data.unlinked_databases) {
        Write-Host "  Непривязанных баз: $($result.Data.unlinked_databases.Count)" -ForegroundColor Gray
        foreach ($db in $result.Data.unlinked_databases) {
            $configInfo = ""
            if ($db.config_name) {
                $configInfo = " (Конфигурация: $($db.config_name))"
            }
            Write-Host "    - $($db.name)$configInfo" -ForegroundColor Gray
        }
    }
} else {
    Write-Host "✗ Ошибка: $($result.Error)" -ForegroundColor Red
}
Write-Host ""

# Тест 3: Автоматическая привязка баз данных
Write-Host "Тест 3: Автоматическая привязка баз данных" -ForegroundColor Yellow
Write-Host "  Выполняется автоматическая привязка..." -ForegroundColor Gray
$result = Invoke-ApiRequest -Method "POST" -Url "$baseUrl/api/clients/$testClientId/databases/auto-link"
if ($result.Success) {
    Write-Host "✓ Автоматическая привязка завершена" -ForegroundColor Green
    Write-Host "  Привязано баз: $($result.Data.linked_count)" -ForegroundColor Gray
    Write-Host "  Всего баз: $($result.Data.total_databases)" -ForegroundColor Gray
    if ($result.Data.unlinked_count) {
        Write-Host "  Осталось непривязанных: $($result.Data.unlinked_count)" -ForegroundColor Gray
    }
    if ($result.Data.errors -and $result.Data.errors.Count -gt 0) {
        Write-Host "  Ошибок: $($result.Data.errors.Count)" -ForegroundColor Yellow
        foreach ($error in $result.Data.errors) {
            Write-Host "    - $error" -ForegroundColor Yellow
        }
    }
} else {
    Write-Host "✗ Ошибка: $($result.Error)" -ForegroundColor Red
    if ($result.StatusCode) {
        Write-Host "  HTTP Status: $($result.StatusCode)" -ForegroundColor Red
    }
}
Write-Host ""

# Тест 4: Повторная проверка списка баз данных
Write-Host "Тест 4: Повторная проверка списка баз данных после привязки" -ForegroundColor Yellow
$result = Invoke-ApiRequest -Method "GET" -Url "$baseUrl/api/clients/$testClientId/databases"
if ($result.Success) {
    Write-Host "✓ Успешно получен обновленный список баз данных" -ForegroundColor Green
    $unlinked = $result.Data | Where-Object { $_.project_id -eq $null -or $_.project_id -eq 0 }
    Write-Host "  Осталось непривязанных баз: $($unlinked.Count)" -ForegroundColor Gray
} else {
    Write-Host "✗ Ошибка: $($result.Error)" -ForegroundColor Red
}
Write-Host ""

Write-Host "=== Тестирование завершено ===" -ForegroundColor Cyan

