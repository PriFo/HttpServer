# Скрипт для тестирования нормализации по всем клиентам и проектам
# Проверяет нормализацию контрагентов и номенклатуры

$baseUrl = "http://localhost:8080"
$timeout = 7

Write-Host "🧪 ТЕСТИРОВАНИЕ НОРМАЛИЗАЦИИ ПО ВСЕМ КЛИЕНТАМ" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Функция для выполнения HTTP запроса
function Invoke-ApiRequest {
    param(
        [string]$Method,
        [string]$Url,
        [object]$Body = $null,
        [int]$Timeout = 7
    )
    
    try {
        $headers = @{
            "Content-Type" = "application/json"
        }
        
        $params = @{
            Method = $Method
            Uri = $Url
            Headers = $headers
            TimeoutSec = $Timeout
            ErrorAction = "Stop"
        }
        
        if ($Body) {
            $params.Body = ($Body | ConvertTo-Json -Depth 10)
        }
        
        $response = Invoke-WebRequest @params
        return @{
            Success = $true
            StatusCode = $response.StatusCode
            Content = $response.Content | ConvertFrom-Json
        }
    }
    catch {
        return @{
            Success = $false
            StatusCode = $_.Exception.Response.StatusCode.value__
            Error = $_.Exception.Message
            Content = $null
        }
    }
}

# 1. Получаем список всех клиентов
Write-Host "📋 Шаг 1: Получение списка клиентов..." -ForegroundColor Yellow
$clientsResponse = Invoke-ApiRequest -Method "GET" -Url "$baseUrl/api/clients" -Timeout $timeout

if (-not $clientsResponse.Success) {
    Write-Host "❌ Ошибка получения клиентов: $($clientsResponse.Error)" -ForegroundColor Red
    exit 1
}

$clients = $clientsResponse.Content
Write-Host "✅ Найдено клиентов: $($clients.Count)" -ForegroundColor Green
Write-Host ""

if ($clients.Count -eq 0) {
    Write-Host "⚠️  Клиенты не найдены. Создайте клиентов перед тестированием." -ForegroundColor Yellow
    exit 0
}

# 2. Для каждого клиента получаем проекты и запускаем нормализацию
$results = @()
$totalClients = $clients.Count
$currentClient = 0

foreach ($client in $clients) {
    $currentClient++
    $clientId = $client.id
    $clientName = $client.name
    
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
    Write-Host "👤 Клиент $currentClient/$totalClients: $clientName (ID: $clientId)" -ForegroundColor Cyan
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
    
    # Получаем проекты клиента
    Write-Host "  📁 Получение проектов клиента..." -ForegroundColor Yellow
    $projectsResponse = Invoke-ApiRequest -Method "GET" -Url "$baseUrl/api/clients/$clientId/projects" -Timeout $timeout
    
    if (-not $projectsResponse.Success) {
        Write-Host "  ❌ Ошибка получения проектов: $($projectsResponse.Error)" -ForegroundColor Red
        $results += @{
            ClientID = $clientId
            ClientName = $clientName
            Status = "Error"
            Message = "Failed to get projects: $($projectsResponse.Error)"
        }
        continue
    }
    
    $projects = $projectsResponse.Content
    Write-Host "  ✅ Найдено проектов: $($projects.Count)" -ForegroundColor Green
    
    if ($projects.Count -eq 0) {
        Write-Host "  ⚠️  Проекты не найдены для клиента $clientName" -ForegroundColor Yellow
        $results += @{
            ClientID = $clientId
            ClientName = $clientName
            Status = "Skipped"
            Message = "No projects found"
        }
        continue
    }
    
    # Для каждого проекта запускаем нормализацию
    foreach ($project in $projects) {
        $projectId = $project.id
        $projectName = $project.name
        $projectType = $project.project_type
        
        Write-Host ""
        Write-Host "  📦 Проект: $projectName (ID: $projectId, Тип: $projectType)" -ForegroundColor Magenta
        
        # Проверяем тип проекта
        $isCounterparty = $projectType -eq "counterparty"
        $isNomenclature = $projectType -ne "counterparty"
        
        # Запускаем нормализацию для всех активных БД
        Write-Host "    🚀 Запуск нормализации (all_active=true)..." -ForegroundColor Yellow
        
        $startBody = @{
            all_active = $true
            use_kpved = $true
            use_okpd2 = $false
        }
        
        $startResponse = Invoke-ApiRequest -Method "POST" `
            -Url "$baseUrl/api/clients/$clientId/projects/$projectId/normalization/start" `
            -Body $startBody `
            -Timeout $timeout
        
        if (-not $startResponse.Success) {
            Write-Host "    ❌ Ошибка запуска нормализации: $($startResponse.Error)" -ForegroundColor Red
            $results += @{
                ClientID = $clientId
                ClientName = $clientName
                ProjectID = $projectId
                ProjectName = $projectName
                ProjectType = $projectType
                Status = "Error"
                Message = "Failed to start: $($startResponse.Error)"
            }
            continue
        }
        
        Write-Host "    ✅ Нормализация запущена" -ForegroundColor Green
        
        # Ждем немного перед проверкой статуса
        Start-Sleep -Seconds 2
        
        # Проверяем статус нормализации
        Write-Host "    📊 Проверка статуса нормализации..." -ForegroundColor Yellow
        $statusResponse = Invoke-ApiRequest -Method "GET" `
            -Url "$baseUrl/api/clients/$clientId/projects/$projectId/normalization/status" `
            -Timeout $timeout
        
        if ($statusResponse.Success) {
            $status = $statusResponse.Content
            Write-Host "    📈 Статус: $($status.status)" -ForegroundColor Cyan
            if ($status.processed -ne $null) {
                Write-Host "    📊 Обработано: $($status.processed)" -ForegroundColor Cyan
            }
        }
        
        # Получаем статистику
        Write-Host "    📈 Получение статистики..." -ForegroundColor Yellow
        $statsResponse = Invoke-ApiRequest -Method "GET" `
            -Url "$baseUrl/api/clients/$clientId/projects/$projectId/normalization/stats" `
            -Timeout $timeout
        
        if ($statsResponse.Success) {
            $stats = $statsResponse.Content
            Write-Host "    ✅ Статистика получена" -ForegroundColor Green
        }
        
        $results += @{
            ClientID = $clientId
            ClientName = $clientName
            ProjectID = $projectId
            ProjectName = $projectName
            ProjectType = $projectType
            Status = "Started"
            Message = "Normalization started successfully"
        }
        
        # Небольшая пауза между проектами
        Start-Sleep -Seconds 1
    }
    
    Write-Host ""
}

# 3. Итоговый отчет
Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "📊 ИТОГОВЫЙ ОТЧЕТ" -ForegroundColor Cyan
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host ""

$total = $results.Count
$started = ($results | Where-Object { $_.Status -eq "Started" }).Count
$errors = ($results | Where-Object { $_.Status -eq "Error" }).Count
$skipped = ($results | Where-Object { $_.Status -eq "Skipped" }).Count

Write-Host "Всего обработано: $total" -ForegroundColor White
Write-Host "  ✅ Успешно запущено: $started" -ForegroundColor Green
Write-Host "  ❌ Ошибок: $errors" -ForegroundColor Red
Write-Host "  ⚠️  Пропущено: $skipped" -ForegroundColor Yellow
Write-Host ""

# Детали по типам проектов
$counterpartyProjects = ($results | Where-Object { $_.ProjectType -eq "counterparty" }).Count
$nomenclatureProjects = ($results | Where-Object { $_.ProjectType -ne "counterparty" -and $_.ProjectType -ne $null }).Count

Write-Host "По типам проектов:" -ForegroundColor White
Write-Host "  👥 Контрагенты: $counterpartyProjects" -ForegroundColor Cyan
Write-Host "  📦 Номенклатура: $nomenclatureProjects" -ForegroundColor Cyan
Write-Host ""

# Сохраняем результаты в файл
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$reportFile = "normalization_test_report_$timestamp.json"
$results | ConvertTo-Json -Depth 10 | Out-File -FilePath $reportFile -Encoding UTF8
Write-Host "📄 Отчет сохранен: $reportFile" -ForegroundColor Green
Write-Host ""

Write-Host "✅ Тестирование завершено!" -ForegroundColor Green


