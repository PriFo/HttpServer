# Скрипт для проверки API просмотра контрагентов по клиенту
# Использование: .\test_counterparties_by_client.ps1 [client_id] [base_url]

param(
    [int]$ClientId = 0,
    [string]$BaseUrl = "http://localhost:9999"
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Проверка API контрагентов по клиенту" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Функция для выполнения запроса
function Test-ApiEndpoint {
    param(
        [string]$Url,
        [string]$Description
    )
    
    Write-Host "`n[$Description]" -ForegroundColor Yellow
    Write-Host "URL: $Url" -ForegroundColor Gray
    
    try {
        $response = Invoke-WebRequest -Uri $Url -Method Get -TimeoutSec 7 -UseBasicParsing -ErrorAction Stop
        $statusCode = $response.StatusCode
        $content = $response.Content | ConvertFrom-Json
        
        Write-Host "✓ Статус: $statusCode" -ForegroundColor Green
        
        if ($content.counterparties) {
            $count = $content.counterparties.Count
            $total = $content.total
            Write-Host "✓ Контрагентов в ответе: $count" -ForegroundColor Green
            Write-Host "✓ Всего контрагентов: $total" -ForegroundColor Green
            
            if ($content.projects) {
                Write-Host "✓ Проектов: $($content.projects.Count)" -ForegroundColor Green
            }
            
            if ($count -gt 0) {
                Write-Host "`nПервый контрагент:" -ForegroundColor Cyan
                $first = $content.counterparties[0]
                Write-Host "  ID: $($first.id)" -ForegroundColor White
                Write-Host "  Название: $($first.name)" -ForegroundColor White
                if ($first.normalized_name) {
                    Write-Host "  Нормализованное: $($first.normalized_name)" -ForegroundColor White
                }
                if ($first.tax_id) {
                    Write-Host "  ИНН: $($first.tax_id)" -ForegroundColor White
                }
            }
            
            return $true
        } else {
            Write-Host "⚠ Контрагенты не найдены в ответе" -ForegroundColor Yellow
            return $false
        }
    } catch {
        Write-Host "✗ Ошибка: $($_.Exception.Message)" -ForegroundColor Red
        if ($_.Exception.Response) {
            $statusCode = $_.Exception.Response.StatusCode.value__
            Write-Host "  Статус: $statusCode" -ForegroundColor Red
        }
        return $false
    }
}

# 1. Получаем список клиентов
Write-Host "[1] Получение списка клиентов..." -ForegroundColor Yellow
try {
    $clientsUrl = "$BaseUrl/api/clients"
    $clientsResponse = Invoke-WebRequest -Uri $clientsUrl -Method Get -TimeoutSec 7 -UseBasicParsing -ErrorAction Stop
    $clients = $clientsResponse.Content | ConvertFrom-Json
    
    if ($clients.clients -and $clients.clients.Count -gt 0) {
        Write-Host "✓ Найдено клиентов: $($clients.clients.Count)" -ForegroundColor Green
        
        if ($ClientId -eq 0) {
            $ClientId = $clients.clients[0].id
            Write-Host "Используется первый клиент: ID=$ClientId, Имя=$($clients.clients[0].name)" -ForegroundColor Cyan
        } else {
            $client = $clients.clients | Where-Object { $_.id -eq $ClientId }
            if ($client) {
                Write-Host "Используется клиент: ID=$ClientId, Имя=$($client.name)" -ForegroundColor Cyan
            } else {
                Write-Host "⚠ Клиент с ID=$ClientId не найден, используем первый" -ForegroundColor Yellow
                $ClientId = $clients.clients[0].id
            }
        }
    } else {
        Write-Host "✗ Клиенты не найдены" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "✗ Ошибка получения списка клиентов: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# 2. Проверка базового запроса
$test1 = Test-ApiEndpoint -Url "$BaseUrl/api/counterparties/normalized?client_id=$ClientId" -Description "Базовый запрос контрагентов по клиенту"

# 3. Проверка пагинации
$test2 = Test-ApiEndpoint -Url "$BaseUrl/api/counterparties/normalized?client_id=$ClientId&page=1&limit=10" -Description "Запрос с пагинацией (page=1, limit=10)"

# 4. Проверка поиска
$test3 = Test-ApiEndpoint -Url "$BaseUrl/api/counterparties/normalized?client_id=$ClientId&search=тест" -Description "Поиск контрагентов (search=тест)"

# 5. Получаем проекты клиента для проверки фильтра
Write-Host "`n[5] Получение проектов клиента..." -ForegroundColor Yellow
try {
    $projectsUrl = "$BaseUrl/api/clients/$ClientId"
    $clientResponse = Invoke-WebRequest -Uri $projectsUrl -Method Get -TimeoutSec 7 -UseBasicParsing -ErrorAction Stop
    $clientData = $clientResponse.Content | ConvertFrom-Json
    
    if ($clientData.projects -and $clientData.projects.Count -gt 0) {
        $projectId = $clientData.projects[0].id
        Write-Host "✓ Найдено проектов: $($clientData.projects.Count)" -ForegroundColor Green
        Write-Host "Используется первый проект: ID=$projectId" -ForegroundColor Cyan
        
        # 6. Проверка фильтра по проекту
        $test4 = Test-ApiEndpoint -Url "$BaseUrl/api/counterparties/normalized?client_id=$ClientId&project_id=$projectId" -Description "Фильтр по проекту (project_id=$projectId)"
    } else {
        Write-Host "⚠ Проекты не найдены, пропускаем проверку фильтра по проекту" -ForegroundColor Yellow
    }
} catch {
    Write-Host "⚠ Не удалось получить проекты: $($_.Exception.Message)" -ForegroundColor Yellow
}

# Итоги
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "Итоги проверки" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Клиент ID: $ClientId" -ForegroundColor White
Write-Host "Base URL: $BaseUrl" -ForegroundColor White
Write-Host ""

if ($test1) {
    Write-Host "✓ Базовый запрос работает" -ForegroundColor Green
} else {
    Write-Host "✗ Базовый запрос не работает" -ForegroundColor Red
}

if ($test2) {
    Write-Host "✓ Пагинация работает" -ForegroundColor Green
} else {
    Write-Host "✗ Пагинация не работает" -ForegroundColor Red
}

if ($test3) {
    Write-Host "✓ Поиск работает" -ForegroundColor Green
} else {
    Write-Host "⚠ Поиск может не работать (возможно, нет результатов)" -ForegroundColor Yellow
}

Write-Host "`nПроверка завершена!" -ForegroundColor Cyan

