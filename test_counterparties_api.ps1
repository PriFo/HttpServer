# Тестирование API endpoints для контрагентов
$baseUrl = "http://localhost:9999"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Тестирование API контрагентов" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. Получить список клиентов
Write-Host "[1/8] Получение списка клиентов..." -ForegroundColor Yellow
try {
    $clientsResponse = Invoke-RestMethod -Uri "$baseUrl/api/clients" -Method GET -ErrorAction Stop
    Write-Host "  ✓ Клиенты получены: $($clientsResponse.Count)" -ForegroundColor Green
    
    if ($clientsResponse -and $clientsResponse.Count -gt 0) {
        $firstClient = $clientsResponse[0]
        $clientId = $firstClient.id
        Write-Host "  Используем клиента ID: $clientId, Name: $($firstClient.name)" -ForegroundColor Gray
        
        # 2. Проверка /api/counterparties/normalized?client_id={id}
        Write-Host ""
        Write-Host "[2/8] Проверка /api/counterparties/normalized?client_id=$clientId..." -ForegroundColor Yellow
        try {
            $normalizedResponse = Invoke-RestMethod -Uri "$baseUrl/api/counterparties/normalized?client_id=$clientId" -Method GET -ErrorAction Stop
            Write-Host "  ✓ Данные получены" -ForegroundColor Green
            Write-Host "  Структура ответа:" -ForegroundColor Gray
            Write-Host "    - counterparties: $($normalizedResponse.counterparties.Count)" -ForegroundColor Gray
            Write-Host "    - projects: $($normalizedResponse.projects.Count)" -ForegroundColor Gray
            Write-Host "    - total: $($normalizedResponse.total)" -ForegroundColor Gray
            Write-Host "    - page: $($normalizedResponse.page)" -ForegroundColor Gray
            Write-Host "    - limit: $($normalizedResponse.limit)" -ForegroundColor Gray
            
            if ($normalizedResponse.counterparties -and $normalizedResponse.counterparties.Count -gt 0) {
                $firstCp = $normalizedResponse.counterparties[0]
                Write-Host "  Пример контрагента:" -ForegroundColor Gray
                Write-Host "    - ID: $($firstCp.id)" -ForegroundColor Gray
                Write-Host "    - Name: $($firstCp.normalized_name)" -ForegroundColor Gray
                Write-Host "    - Tax ID: $($firstCp.tax_id)" -ForegroundColor Gray
            }
        } catch {
            Write-Host "  ✗ Ошибка: $($_.Exception.Message)" -ForegroundColor Red
        }
        
        # 3. Проверка пагинации
        Write-Host ""
        Write-Host "[3/8] Проверка пагинации: ?client_id=$clientId page=1 limit=10..." -ForegroundColor Yellow
        try {
            $paginatedUrl = "$baseUrl/api/counterparties/normalized?client_id=$clientId" + "`&page=1" + "`&limit=10"
            $paginatedResponse = Invoke-RestMethod -Uri $paginatedUrl -Method GET -ErrorAction Stop
            Write-Host "  ✓ Пагинация работает" -ForegroundColor Green
            Write-Host "    - Получено: $($paginatedResponse.counterparties.Count)" -ForegroundColor Gray
            Write-Host "    - Всего: $($paginatedResponse.total)" -ForegroundColor Gray
            Write-Host "    - Страница: $($paginatedResponse.page)" -ForegroundColor Gray
            Write-Host "    - Лимит: $($paginatedResponse.limit)" -ForegroundColor Gray
        } catch {
            Write-Host "  ✗ Ошибка: $($_.Exception.Message)" -ForegroundColor Red
        }
        
        # 4. Проверка поиска
        Write-Host ""
        Write-Host "[4/8] Проверка поиска: ?client_id=$clientId&search=тест..." -ForegroundColor Yellow
        try {
            $searchUrl = "$baseUrl/api/counterparties/normalized?client_id=$clientId" + "`&search=тест"
            $searchResponse = Invoke-RestMethod -Uri $searchUrl -Method GET -ErrorAction Stop
            Write-Host "  ✓ Поиск работает" -ForegroundColor Green
            Write-Host "    - Найдено: $($searchResponse.counterparties.Count)" -ForegroundColor Gray
            Write-Host "    - Всего: $($searchResponse.total)" -ForegroundColor Gray
        } catch {
            Write-Host "  ✗ Ошибка: $($_.Exception.Message)" -ForegroundColor Red
        }
        
        # 5. Проверка фильтра по проекту (если есть проекты)
        Write-Host ""
        Write-Host "[5/8] Получение проектов клиента..." -ForegroundColor Yellow
        try {
            $projectsResponse = Invoke-RestMethod -Uri "$baseUrl/api/clients/$clientId/projects" -Method GET -ErrorAction Stop
            if ($projectsResponse -and $projectsResponse.Count -gt 0) {
                $firstProject = $projectsResponse[0]
                $projectId = $firstProject.id
                Write-Host "  ✓ Проекты получены: $($projectsResponse.Count)" -ForegroundColor Green
                Write-Host "  Используем проект ID: $projectId, Name: $($firstProject.name)" -ForegroundColor Gray
                
                Write-Host ""
                Write-Host "[6/8] Проверка фильтра по проекту: ?client_id=$clientId project_id=$projectId..." -ForegroundColor Yellow
                try {
                    $projectFilterUrl = "$baseUrl/api/counterparties/normalized?client_id=$clientId" + "`&project_id=$projectId"
                    $projectFilterResponse = Invoke-RestMethod -Uri $projectFilterUrl -Method GET -ErrorAction Stop
                    Write-Host "  ✓ Фильтр по проекту работает" -ForegroundColor Green
                    Write-Host "    - Найдено: $($projectFilterResponse.counterparties.Count)" -ForegroundColor Gray
                    Write-Host "    - Всего: $($projectFilterResponse.total)" -ForegroundColor Gray
                } catch {
                    Write-Host "  ✗ Ошибка: $($_.Exception.Message)" -ForegroundColor Red
                }
            } else {
                Write-Host "  ⚠ Проекты не найдены, пропускаем проверку фильтра" -ForegroundColor Yellow
            }
        } catch {
            Write-Host "  ✗ Ошибка получения проектов: $($_.Exception.Message)" -ForegroundColor Red
        }
        
        # 6. Проверка /api/counterparties/all
        Write-Host ""
        Write-Host "[7/8] Проверка /api/counterparties/all?client_id=$clientId..." -ForegroundColor Yellow
        try {
            $allUrl = "$baseUrl/api/counterparties/all?client_id=$clientId" + "`&limit=10"
            $allResponse = Invoke-RestMethod -Uri $allUrl -Method GET -ErrorAction Stop
            Write-Host "  ✓ /api/counterparties/all работает" -ForegroundColor Green
            Write-Host "    - Получено: $($allResponse.counterparties.Count)" -ForegroundColor Gray
            Write-Host "    - Всего: $($allResponse.total)" -ForegroundColor Gray
            Write-Host "    - Проекты: $($allResponse.projects.Count)" -ForegroundColor Gray
            if ($allResponse.stats) {
                Write-Host "    - Stats: $($allResponse.stats | ConvertTo-Json -Compress)" -ForegroundColor Gray
            }
        } catch {
            Write-Host "  ✗ Ошибка: $($_.Exception.Message)" -ForegroundColor Red
        }
        
        # 7. Проверка обработки ошибок
        Write-Host ""
        Write-Host "[8/8] Проверка обработки ошибок (несуществующий клиент)..." -ForegroundColor Yellow
        try {
            $errorUrl = "$baseUrl/api/counterparties/normalized?client_id=999999"
            $errorResponse = Invoke-WebRequest -Uri $errorUrl -Method GET -ErrorAction Stop
            Write-Host "  ⚠ Неожиданный успех для несуществующего клиента" -ForegroundColor Yellow
        } catch {
            $statusCode = $_.Exception.Response.StatusCode.value__
            if ($statusCode -eq 400 -or $statusCode -eq 404) {
                Write-Host "  ✓ Ошибка корректно обработана: $statusCode" -ForegroundColor Green
            } else {
                Write-Host "  ⚠ Неожиданный код ошибки: $statusCode" -ForegroundColor Yellow
            }
        }
        
    } else {
        Write-Host "  ⚠ Клиенты не найдены. Создайте клиента и проект для тестирования." -ForegroundColor Yellow
    }
} catch {
    Write-Host "  ✗ Ошибка получения клиентов: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "  Убедитесь, что сервер запущен на $baseUrl" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Тестирование завершено" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

