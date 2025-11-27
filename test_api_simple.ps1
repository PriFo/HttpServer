# Простой скрипт для тестирования API контрагентов
$backendUrl = "http://127.0.0.1:9999"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Тестирование API контрагентов" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. Получение списка клиентов
Write-Host "1. Получение списка клиентов..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$backendUrl/api/clients" -UseBasicParsing
    $clients = $response.Content | ConvertFrom-Json
    Write-Host "   Найдено клиентов: $($clients.Count)" -ForegroundColor Green
    
    if ($clients.Count -eq 0) {
        Write-Host "   [WARNING] Клиенты не найдены" -ForegroundColor Yellow
        exit 0
    }
    
    $clientId = $clients[0].id
    Write-Host "   Используем клиента ID: $clientId" -ForegroundColor Gray
} catch {
    Write-Host "   [ERROR] $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# 2. Получение контрагентов
Write-Host "`n2. Получение контрагентов для клиента $clientId..." -ForegroundColor Yellow
try {
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=$clientId"
    $response = Invoke-WebRequest -Uri $uri.ToString() -UseBasicParsing
    $data = $response.Content | ConvertFrom-Json
    
    Write-Host "   [OK] Запрос выполнен успешно" -ForegroundColor Green
    Write-Host "   Контрагентов: $($data.counterparties.Count)" -ForegroundColor Gray
    Write-Host "   Всего: $($data.total)" -ForegroundColor Gray
    Write-Host "   Проектов: $($data.projects.Count)" -ForegroundColor Gray
    Write-Host "   Лимит: $($data.limit)" -ForegroundColor Gray
    Write-Host "   Страница: $($data.page)" -ForegroundColor Gray
    
    if ($data.projects.Count -gt 0) {
        Write-Host "`n   Проекты:" -ForegroundColor Cyan
        foreach ($project in $data.projects) {
            Write-Host "     - ID: $($project.id), Имя: $($project.name)" -ForegroundColor Gray
        }
    }
} catch {
    Write-Host "   [ERROR] $($_.Exception.Message)" -ForegroundColor Red
}

# 3. Проверка пагинации
Write-Host "`n3. Проверка пагинации (page=1, limit=5)..." -ForegroundColor Yellow
try {
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=$clientId&page=1&limit=5"
    $response = Invoke-WebRequest -Uri $uri.ToString() -UseBasicParsing
    $data = $response.Content | ConvertFrom-Json
    
    Write-Host "   [OK] Пагинация работает" -ForegroundColor Green
    Write-Host "   Получено: $($data.counterparties.Count), Лимит: $($data.limit), Страница: $($data.page)" -ForegroundColor Gray
} catch {
    Write-Host "   [ERROR] $($_.Exception.Message)" -ForegroundColor Red
}

# 4. Проверка обработки ошибок
Write-Host "`n4. Проверка обработки ошибок..." -ForegroundColor Yellow

Write-Host "   4.1. Запрос без client_id..." -ForegroundColor Cyan
try {
    $response = Invoke-WebRequest -Uri "$backendUrl/api/counterparties/normalized" -UseBasicParsing
    Write-Host "      [WARNING] Должна быть ошибка 400" -ForegroundColor Yellow
} catch {
    $code = $_.Exception.Response.StatusCode.value__
    if ($code -eq 400) {
        Write-Host "      [OK] Корректная обработка (400)" -ForegroundColor Green
    } else {
        Write-Host "      [INFO] Код ответа: $code" -ForegroundColor Gray
    }
}

Write-Host "   4.2. Несуществующий клиент..." -ForegroundColor Cyan
try {
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=999999"
    $response = Invoke-WebRequest -Uri $uri.ToString() -UseBasicParsing
    $data = $response.Content | ConvertFrom-Json
    if ($data.counterparties.Count -eq 0) {
        Write-Host "      [OK] Пустой список для несуществующего клиента" -ForegroundColor Green
    }
} catch {
    Write-Host "      [INFO] $($_.Exception.Message)" -ForegroundColor Gray
}

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "Тестирование завершено" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
