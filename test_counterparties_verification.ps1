# Скрипт для проверки функциональности контрагентов
# Проверяет API и фронтенд

$ErrorActionPreference = "Continue"
$backendUrl = "http://localhost:9999"
$frontendUrl = "http://localhost:3000"
$timeout = 7

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Проверка функциональности контрагентов" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Функция для проверки доступности сервера
function Test-Server {
    param($url, $name)
    try {
        $response = Invoke-WebRequest -Uri "$url/health" -TimeoutSec $timeout -UseBasicParsing -ErrorAction Stop
        Write-Host "[OK] $name доступен" -ForegroundColor Green
        return $true
    } catch {
        Write-Host "[ERROR] $name недоступен: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

# Проверка доступности серверов
Write-Host "1. Проверка доступности серверов..." -ForegroundColor Yellow
$backendAvailable = Test-Server $backendUrl "Backend"
$frontendAvailable = Test-Server $frontendUrl "Frontend"

if (-not $backendAvailable) {
    Write-Host ""
    Write-Host "ВНИМАНИЕ: Backend недоступен. Убедитесь, что сервер запущен на порту 9999" -ForegroundColor Yellow
    Write-Host "Для запуска используйте: go run main_no_gui.go" -ForegroundColor Yellow
    Write-Host ""
}

if (-not $frontendAvailable) {
    Write-Host ""
    Write-Host "ВНИМАНИЕ: Frontend недоступен. Убедитесь, что сервер запущен на порту 3000" -ForegroundColor Yellow
    Write-Host "Для запуска используйте: cd frontend && npm run dev" -ForegroundColor Yellow
    Write-Host ""
}

if (-not $backendAvailable) {
    Write-Host "Проверка API невозможна без backend сервера" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "2. Проверка API endpoints..." -ForegroundColor Yellow
Write-Host ""

# Получение списка клиентов
Write-Host "2.1. Получение списка клиентов..." -ForegroundColor Cyan
try {
    $clientsResponse = Invoke-WebRequest -Uri "$backendUrl/api/clients" -TimeoutSec $timeout -UseBasicParsing
    $clients = $clientsResponse.Content | ConvertFrom-Json
    
    if ($clients -and $clients.Count -gt 0) {
        $firstClient = $clients[0]
        $clientId = $firstClient.id
        Write-Host "  [OK] Найден клиент с ID: $clientId" -ForegroundColor Green
        Write-Host "  Имя: $($firstClient.name)" -ForegroundColor Gray
    } else {
        Write-Host "  [WARNING] Клиенты не найдены" -ForegroundColor Yellow
        Write-Host "  Используем тестовый client_id=1" -ForegroundColor Gray
        $clientId = 1
    }
} catch {
    Write-Host "  [ERROR] Не удалось получить список клиентов: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "  Используем тестовый client_id=1" -ForegroundColor Gray
    $clientId = 1
}

Write-Host ""
Write-Host "2.2. Проверка /api/counterparties/normalized?client_id=$clientId..." -ForegroundColor Cyan
try {
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=$clientId"
    $url = $uri.ToString()
    
    $response = Invoke-WebRequest -Uri $url -TimeoutSec $timeout -UseBasicParsing
    $data = $response.Content | ConvertFrom-Json
    
    Write-Host "  [OK] Запрос выполнен успешно" -ForegroundColor Green
    Write-Host "  Структура ответа:" -ForegroundColor Gray
    Write-Host "    - counterparties: $($data.counterparties.Count) записей" -ForegroundColor Gray
    Write-Host "    - total: $($data.total)" -ForegroundColor Gray
    Write-Host "    - projects: $($data.projects.Count) проектов" -ForegroundColor Gray
    Write-Host "    - offset: $($data.offset)" -ForegroundColor Gray
    Write-Host "    - limit: $($data.limit)" -ForegroundColor Gray
    Write-Host "    - page: $($data.page)" -ForegroundColor Gray
    
    if ($data.counterparties.Count -gt 0) {
        Write-Host "  [OK] Контрагенты найдены" -ForegroundColor Green
        $firstCounterparty = $data.counterparties[0]
        Write-Host "  Пример контрагента:" -ForegroundColor Gray
        Write-Host "    - ID: $($firstCounterparty.id)" -ForegroundColor Gray
        Write-Host "    - Название: $($firstCounterparty.name)" -ForegroundColor Gray
        if ($firstCounterparty.normalized_name) {
            Write-Host "    - Нормализованное название: $($firstCounterparty.normalized_name)" -ForegroundColor Gray
        }
    } else {
        Write-Host "  [WARNING] Контрагенты не найдены для клиента $clientId" -ForegroundColor Yellow
    }
} catch {
    Write-Host "  [ERROR] Ошибка запроса: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "  Ответ сервера: $responseBody" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "2.3. Проверка пагинации (page=1&limit=10)..." -ForegroundColor Cyan
try {
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=$clientId&page=1&limit=10"
    $url = $uri.ToString()
    
    $response = Invoke-WebRequest -Uri $url -TimeoutSec $timeout -UseBasicParsing
    $pageData = $response.Content | ConvertFrom-Json
    
    Write-Host "  [OK] Пагинация работает" -ForegroundColor Green
    Write-Host "    - Получено записей: $($pageData.counterparties.Count)" -ForegroundColor Gray
    Write-Host "    - Лимит: $($pageData.limit)" -ForegroundColor Gray
    Write-Host "    - Страница: $($pageData.page)" -ForegroundColor Gray
    Write-Host "    - Всего: $($pageData.total)" -ForegroundColor Gray
    
    if ($pageData.counterparties.Count -le $pageData.limit) {
        Write-Host "  [OK] Лимит соблюдается" -ForegroundColor Green
    } else {
        Write-Host "  [WARNING] Получено больше записей, чем указано в лимите" -ForegroundColor Yellow
    }
} catch {
    Write-Host "  [ERROR] Ошибка проверки пагинации: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "2.4. Проверка поиска (search=тест)..." -ForegroundColor Cyan
try {
    $searchQuery = [System.Web.HttpUtility]::UrlEncode("тест")
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=$clientId&search=$searchQuery"
    $url = $uri.ToString()
    
    $response = Invoke-WebRequest -Uri $url -TimeoutSec $timeout -UseBasicParsing
    $searchData = $response.Content | ConvertFrom-Json
    
    Write-Host "  [OK] Поиск работает" -ForegroundColor Green
    Write-Host "    - Найдено записей: $($searchData.counterparties.Count)" -ForegroundColor Gray
    Write-Host "    - Всего: $($searchData.total)" -ForegroundColor Gray
} catch {
    Write-Host "  [ERROR] Ошибка проверки поиска: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "2.5. Проверка фильтра по проекту..." -ForegroundColor Cyan
try {
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=$clientId"
    $url = $uri.ToString()
    $projectsResponse = Invoke-WebRequest -Uri $url -TimeoutSec $timeout -UseBasicParsing
    $projectsData = $projectsResponse.Content | ConvertFrom-Json
    
    if ($projectsData.projects -and $projectsData.projects.Count -gt 0) {
        $projectId = $projectsData.projects[0].id
        try {
            $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
            $uri.Query = "client_id=$clientId&project_id=$projectId"
            $url = $uri.ToString()
            
            $response = Invoke-WebRequest -Uri $url -TimeoutSec $timeout -UseBasicParsing
            $filteredData = $response.Content | ConvertFrom-Json
            
            Write-Host "  [OK] Фильтр по проекту работает" -ForegroundColor Green
            Write-Host "    - Проект ID: $projectId" -ForegroundColor Gray
            Write-Host "    - Найдено записей: $($filteredData.counterparties.Count)" -ForegroundColor Gray
            Write-Host "    - Всего: $($filteredData.total)" -ForegroundColor Gray
        } catch {
            Write-Host "  [ERROR] Ошибка проверки фильтра: $($_.Exception.Message)" -ForegroundColor Red
        }
    } else {
        Write-Host "  [SKIP] Нет проектов для проверки фильтра" -ForegroundColor Yellow
    }
} catch {
    Write-Host "  [ERROR] Не удалось получить список проектов: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "3. Проверка обработки ошибок..." -ForegroundColor Yellow
Write-Host ""

Write-Host "3.1. Проверка несуществующего клиента..." -ForegroundColor Cyan
try {
    $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
    $uri.Query = "client_id=999999"
    $url = $uri.ToString()
    
    $response = Invoke-WebRequest -Uri $url -TimeoutSec $timeout -UseBasicParsing
    $data = $response.Content | ConvertFrom-Json
    
    if ($data.counterparties.Count -eq 0) {
        Write-Host "  [OK] Корректно обработан несуществующий клиент (пустой список)" -ForegroundColor Green
    } else {
        Write-Host "  [WARNING] Для несуществующего клиента возвращены данные" -ForegroundColor Yellow
    }
} catch {
    if ($_.Exception.Response.StatusCode -eq 404 -or $_.Exception.Response.StatusCode -eq 400) {
        Write-Host "  [OK] Корректная обработка ошибки для несуществующего клиента" -ForegroundColor Green
    } else {
        Write-Host "  [ERROR] Неожиданная ошибка: $($_.Exception.Message)" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "3.2. Проверка запроса без client_id..." -ForegroundColor Cyan
try {
    $url = "$backendUrl/api/counterparties/normalized"
    $response = Invoke-WebRequest -Uri $url -TimeoutSec $timeout -UseBasicParsing
    Write-Host "  [WARNING] Запрос без client_id не вернул ошибку" -ForegroundColor Yellow
} catch {
    if ($_.Exception.Response.StatusCode -eq 400) {
        Write-Host "  [OK] Корректная обработка ошибки (400 Bad Request)" -ForegroundColor Green
    } else {
        Write-Host "  [WARNING] Неожиданный код ошибки: $($_.Exception.Response.StatusCode)" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Проверка завершена" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

if ($frontendAvailable) {
    Write-Host "Для проверки фронтенда откройте:" -ForegroundColor Yellow
    Write-Host "  $frontendUrl/clients/$clientId" -ForegroundColor Cyan
    Write-Host "  и перейдите на вкладку Контрагенты" -ForegroundColor Cyan
    Write-Host ""
}
