# Детальная проверка API контрагентов
$backendUrl = "http://127.0.0.1:9999"
$timeout = 7

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Детальная проверка API контрагентов" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. Получение списка клиентов
Write-Host "1. Получение списка клиентов..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$backendUrl/api/clients" -TimeoutSec $timeout -UseBasicParsing
    $clients = $response.Content | ConvertFrom-Json
    Write-Host "[OK] Найдено клиентов: $($clients.Count)" -ForegroundColor Green
    
    foreach ($client in $clients) {
        Write-Host "  - ID: $($client.id), Имя: $($client.name)" -ForegroundColor Gray
    }
    
    if ($clients.Count -eq 0) {
        Write-Host "[WARNING] Клиенты не найдены" -ForegroundColor Yellow
        exit 0
    }
    
    $clientId = $clients[0].id
    Write-Host "`nИспользуем клиента с ID: $clientId" -ForegroundColor Cyan
} catch {
    Write-Host "[ERROR] Ошибка получения клиентов: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# 2. Проверка основного endpoint
Write-Host "`n2. Проверка /api/counterparties/normalized?client_id=$clientId..." -ForegroundColor Yellow
try {
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=$clientId"
    $response = Invoke-WebRequest -Uri $uri.ToString() -TimeoutSec $timeout -UseBasicParsing
    $data = $response.Content | ConvertFrom-Json
    
    Write-Host "[OK] Запрос выполнен успешно" -ForegroundColor Green
    Write-Host "  Структура ответа:" -ForegroundColor Gray
    Write-Host "    - counterparties: $($data.counterparties.Count) записей" -ForegroundColor Gray
    Write-Host "    - total: $($data.total)" -ForegroundColor Gray
    Write-Host "    - projects: $($data.projects.Count) проектов" -ForegroundColor Gray
    Write-Host "    - offset: $($data.offset)" -ForegroundColor Gray
    Write-Host "    - limit: $($data.limit)" -ForegroundColor Gray
    Write-Host "    - page: $($data.page)" -ForegroundColor Gray
    
    if ($data.counterparties.Count -gt 0) {
        Write-Host "`n[OK] Контрагенты найдены!" -ForegroundColor Green
        $first = $data.counterparties[0]
        Write-Host "  Пример контрагента:" -ForegroundColor Gray
        Write-Host "    - ID: $($first.id)" -ForegroundColor Gray
        Write-Host "    - Название: $($first.name)" -ForegroundColor Gray
        if ($first.normalized_name) {
            Write-Host "    - Нормализованное название: $($first.normalized_name)" -ForegroundColor Gray
        }
    } else {
        Write-Host "`n[INFO] Контрагенты не найдены для клиента $clientId" -ForegroundColor Yellow
    }
} catch {
    Write-Host "[ERROR] Ошибка: $($_.Exception.Message)" -ForegroundColor Red
}

# 3. Проверка пагинации
Write-Host "`n3. Проверка пагинации (page=1&limit=10)..." -ForegroundColor Yellow
try {
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=$clientId&page=1&limit=10"
    $response = Invoke-WebRequest -Uri $uri.ToString() -TimeoutSec $timeout -UseBasicParsing
    $pageData = $response.Content | ConvertFrom-Json
    
    Write-Host "[OK] Пагинация работает" -ForegroundColor Green
    Write-Host "    - Получено записей: $($pageData.counterparties.Count)" -ForegroundColor Gray
    Write-Host "    - Лимит: $($pageData.limit)" -ForegroundColor Gray
    Write-Host "    - Страница: $($pageData.page)" -ForegroundColor Gray
    Write-Host "    - Всего: $($pageData.total)" -ForegroundColor Gray
    
    if ($pageData.counterparties.Count -le $pageData.limit) {
        Write-Host "  [OK] Лимит соблюдается" -ForegroundColor Green
    }
} catch {
    Write-Host "[ERROR] Ошибка проверки пагинации: $($_.Exception.Message)" -ForegroundColor Red
}

# 4. Проверка поиска
Write-Host "`n4. Проверка поиска (search=тест)..." -ForegroundColor Yellow
try {
    $searchQuery = [System.Web.HttpUtility]::UrlEncode("тест")
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=$clientId&search=$searchQuery"
    $response = Invoke-WebRequest -Uri $uri.ToString() -TimeoutSec $timeout -UseBasicParsing
    $searchData = $response.Content | ConvertFrom-Json
    
    Write-Host "[OK] Поиск работает" -ForegroundColor Green
    Write-Host "    - Найдено записей: $($searchData.counterparties.Count)" -ForegroundColor Gray
    Write-Host "    - Всего: $($searchData.total)" -ForegroundColor Gray
} catch {
    Write-Host "[ERROR] Ошибка проверки поиска: $($_.Exception.Message)" -ForegroundColor Red
}

# 5. Проверка фильтра по проекту
Write-Host "`n5. Проверка фильтра по проекту..." -ForegroundColor Yellow
try {
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=$clientId"
    $response = Invoke-WebRequest -Uri $uri.ToString() -TimeoutSec $timeout -UseBasicParsing
    $projectsData = $response.Content | ConvertFrom-Json
    
    if ($projectsData.projects -and $projectsData.projects.Count -gt 0) {
        $projectId = $projectsData.projects[0].id
        Write-Host "  Найден проект с ID: $projectId" -ForegroundColor Gray
        
        $uri2 = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
        $uri2.Query = "client_id=$clientId&project_id=$projectId"
        $response2 = Invoke-WebRequest -Uri $uri2.ToString() -TimeoutSec $timeout -UseBasicParsing
        $filteredData = $response2.Content | ConvertFrom-Json
        
        Write-Host "[OK] Фильтр по проекту работает" -ForegroundColor Green
        Write-Host "    - Проект ID: $projectId" -ForegroundColor Gray
        Write-Host "    - Найдено записей: $($filteredData.counterparties.Count)" -ForegroundColor Gray
        Write-Host "    - Всего: $($filteredData.total)" -ForegroundColor Gray
    } else {
        Write-Host "[INFO] Нет проектов для проверки фильтра" -ForegroundColor Yellow
    }
} catch {
    Write-Host "[ERROR] Ошибка проверки фильтра: $($_.Exception.Message)" -ForegroundColor Red
}

# 6. Проверка обработки ошибок
Write-Host "`n6. Проверка обработки ошибок..." -ForegroundColor Yellow

Write-Host "  6.1. Несуществующий клиент..." -ForegroundColor Cyan
try {
    $uri3 = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri3.Query = "client_id=999999"
    $response3 = Invoke-WebRequest -Uri $uri3.ToString() -TimeoutSec $timeout -UseBasicParsing
    $data3 = $response3.Content | ConvertFrom-Json
    
    if ($data3.counterparties.Count -eq 0) {
        Write-Host "  [OK] Корректно обработан несуществующий клиент (пустой список)" -ForegroundColor Green
    }
} catch {
    $statusCode1 = $_.Exception.Response.StatusCode.value__
    if ($statusCode1 -eq 404 -or $statusCode1 -eq 400) {
        Write-Host "  [OK] Корректная обработка ошибки" -ForegroundColor Green
    }
}

Write-Host "  6.2. Запрос без client_id..." -ForegroundColor Cyan
try {
    $response4 = Invoke-WebRequest -Uri "$backendUrl/api/counterparties/normalized" -TimeoutSec $timeout -UseBasicParsing
    Write-Host "  [WARNING] Запрос без client_id не вернул ошибку" -ForegroundColor Yellow
} catch {
    $statusCode2 = $_.Exception.Response.StatusCode.value__
    if ($statusCode2 -eq 400) {
        Write-Host "  [OK] Корректная обработка ошибки (400 Bad Request)" -ForegroundColor Green
    } else {
        Write-Host "  [INFO] Код ответа: $statusCode2" -ForegroundColor Gray
    }
}

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "Проверка завершена" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

