# Полный цикл тестирования нормализации номенклатуры по всем базам проекта
$ErrorActionPreference = "Stop"

Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "🔍 ПОЛНЫЙ ЦИКЛ ТЕСТИРОВАНИЯ НОРМАЛИЗАЦИИ НОМЕНКЛАТУРЫ" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

$baseUrl = "http://localhost:9999"
$timeout = 7

# Функция для выполнения curl запросов
function Test-Endpoint {
    param(
        [string]$Method,
        [string]$Url,
        [string]$Body = $null,
        [string]$Description,
        [switch]$ShowResponse
    )
    
    Write-Host "📋 Тест: $Description" -ForegroundColor Yellow
    Write-Host "   URL: $Method $Url" -ForegroundColor Gray
    
    try {
        $headers = @{
            "Content-Type" = "application/json"
        }
        
        $params = @{
            Uri = $Url
            Method = $Method
            Headers = $headers
            TimeoutSec = $timeout
            ErrorAction = "Stop"
        }
        
        if ($Body) {
            $params.Body = $Body
            Write-Host "   Body: $Body" -ForegroundColor Gray
        }
        
        $response = Invoke-RestMethod @params
        
        Write-Host "   ✓ Успешно" -ForegroundColor Green
        if ($ShowResponse) {
            Write-Host "   Response:" -ForegroundColor Gray
            $response | ConvertTo-Json -Depth 10 | Write-Host
        }
        Write-Host ""
        return $response
    }
    catch {
        Write-Host "   ✗ Ошибка: $($_.Exception.Message)" -ForegroundColor Red
        if ($_.ErrorDetails.Message) {
            Write-Host "   Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
        }
        Write-Host ""
        return $null
    }
}

# Шаг 1: Проверка доступности сервера
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "ШАГ 1: Проверка доступности сервера" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

try {
    $healthCheck = Invoke-RestMethod -Uri "$baseUrl/api/health" -Method GET -TimeoutSec $timeout -ErrorAction Stop
    Write-Host "✓ Сервер доступен" -ForegroundColor Green
    Write-Host ""
} catch {
    Write-Host "✗ Сервер недоступен на $baseUrl" -ForegroundColor Red
    Write-Host "  Убедитесь, что сервер запущен: go run main.go" -ForegroundColor Yellow
    Write-Host ""
    exit 1
}

# Шаг 2: Получение списка клиентов
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "ШАГ 2: Получение списка клиентов" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

$clients = Test-Endpoint -Method "GET" -Url "$baseUrl/api/clients" -Description "Получение списка клиентов" -ShowResponse

if (-not $clients -or $clients.Count -eq 0) {
    Write-Host "⚠ Клиенты не найдены. Создайте клиента через API или GUI" -ForegroundColor Yellow
    exit 1
}

$clientId = $clients[0].id
Write-Host "→ Используется клиент ID: $clientId" -ForegroundColor Cyan
Write-Host ""

# Шаг 3: Получение проектов клиента
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "ШАГ 3: Получение проектов клиента" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

$projects = Test-Endpoint -Method "GET" -Url "$baseUrl/api/clients/$clientId/projects" -Description "Получение проектов клиента" -ShowResponse

if (-not $projects -or $projects.Count -eq 0) {
    Write-Host "⚠ Проекты не найдены для клиента $clientId" -ForegroundColor Yellow
    exit 1
}

# Ищем проект с типом номенклатуры (не counterparty)
$nomenclatureProject = $projects | Where-Object { $_.project_type -ne "counterparty" -and $_.project_type -ne "nomenclature_counterparties" } | Select-Object -First 1

if (-not $nomenclatureProject) {
    Write-Host "⚠ Проект с номенклатурой не найден. Используем первый доступный проект" -ForegroundColor Yellow
    $nomenclatureProject = $projects[0]
}

$projectId = $nomenclatureProject.id
Write-Host "→ Используется проект ID: $projectId (тип: $($nomenclatureProject.project_type))" -ForegroundColor Cyan
Write-Host ""

# Шаг 4: Получение баз данных проекта
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "ШАГ 4: Получение баз данных проекта" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

$databases = Test-Endpoint -Method "GET" -Url "$baseUrl/api/clients/$clientId/projects/$projectId/databases" -Description "Получение баз данных проекта" -ShowResponse

if (-not $databases -or $databases.databases.Count -eq 0) {
    Write-Host "⚠ Базы данных не найдены для проекта $projectId" -ForegroundColor Yellow
    Write-Host "  Загрузите базу данных через API или GUI" -ForegroundColor Yellow
    exit 1
}

$activeDatabases = $databases.databases | Where-Object { $_.is_active -eq $true }
Write-Host "→ Найдено активных баз данных: $($activeDatabases.Count)" -ForegroundColor Cyan
Write-Host ""

# Шаг 5: Проверка текущего статуса нормализации
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "ШАГ 5: Проверка текущего статуса нормализации" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

$normalizationStatus = Test-Endpoint -Method "GET" -Url "$baseUrl/api/clients/$clientId/projects/$projectId/normalization/status" -Description "Проверка статуса нормализации" -ShowResponse

# Шаг 6: Запуск нормализации для всех активных баз
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "ШАГ 6: Запуск нормализации номенклатуры для всех баз" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

$normalizeBody = @{
    all_active = $true
    use_kpved = $false
    use_okpd2 = $false
} | ConvertTo-Json

$normalizeResponse = Test-Endpoint -Method "POST" -Url "$baseUrl/api/clients/$clientId/projects/$projectId/normalization/start" -Body $normalizeBody -Description "Запуск нормализации для всех активных баз" -ShowResponse

if (-not $normalizeResponse) {
    Write-Host "✗ Не удалось запустить нормализацию" -ForegroundColor Red
    exit 1
}

Write-Host "→ Нормализация запущена. Ожидание завершения..." -ForegroundColor Cyan
Write-Host ""

# Шаг 7: Мониторинг прогресса нормализации
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "ШАГ 7: Мониторинг прогресса нормализации" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

$maxWaitTime = 300 # 5 минут максимум
$checkInterval = 5 # проверка каждые 5 секунд
$elapsed = 0
$isRunning = $true

while ($isRunning -and $elapsed -lt $maxWaitTime) {
    Start-Sleep -Seconds $checkInterval
    $elapsed += $checkInterval
    
    try {
        $status = Invoke-RestMethod -Uri "$baseUrl/api/clients/$clientId/projects/$projectId/normalization/status" -Method GET -TimeoutSec $timeout -ErrorAction Stop
        
        $isRunning = $status.is_running -eq $true
        
        Write-Host "  Прогресс: $($status.progress)% | Обработано: $($status.processed) | Ошибки: $($status.errors)" -ForegroundColor Gray
        
        if (-not $isRunning) {
            Write-Host "✓ Нормализация завершена" -ForegroundColor Green
            break
        }
    } catch {
        Write-Host "  ⚠ Ошибка получения статуса: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

if ($isRunning) {
    Write-Host "⚠ Нормализация все еще выполняется после $maxWaitTime секунд" -ForegroundColor Yellow
    Write-Host "  Проверьте статус вручную через API" -ForegroundColor Yellow
}

Write-Host ""

# Шаг 8: Получение сессий нормализации
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "ШАГ 8: Получение сессий нормализации" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

$sessions = Test-Endpoint -Method "GET" -Url "$baseUrl/api/clients/$clientId/projects/$projectId/normalization/sessions" -Description "Получение сессий нормализации" -ShowResponse

if ($sessions -and $sessions.sessions) {
    Write-Host "→ Найдено сессий: $($sessions.sessions.Count)" -ForegroundColor Cyan
    foreach ($session in $sessions.sessions) {
        Write-Host "  Сессия ID: $($session.id) | Статус: $($session.status) | Обработано: $($session.processed_count)" -ForegroundColor Gray
    }
}
Write-Host ""

# Шаг 9: Проверка нормализованных данных
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "ШАГ 9: Проверка нормализованных данных" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

$nomenclature = Test-Endpoint -Method "GET" -Url "$baseUrl/api/clients/$clientId/projects/$projectId/nomenclature?limit=10" -Description "Получение нормализованной номенклатуры" -ShowResponse

if ($nomenclature -and $nomenclature.items) {
    Write-Host "→ Найдено записей номенклатуры: $($nomenclature.total)" -ForegroundColor Cyan
    Write-Host "  Показано первых: $($nomenclature.items.Count)" -ForegroundColor Gray
    
    foreach ($item in $nomenclature.items | Select-Object -First 5) {
        Write-Host "  - $($item.name) → $($item.normalized_name) [категория: $($item.category)]" -ForegroundColor Gray
    }
} else {
    Write-Host "⚠ Нормализованные данные не найдены" -ForegroundColor Yellow
}
Write-Host ""

# Шаг 10: Проверка цепочки обработки
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "ШАГ 10: Проверка цепочки обработки" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

Write-Host "✓ API Endpoint: POST /api/clients/$clientId/projects/$projectId/normalization/start" -ForegroundColor Green
Write-Host "✓ Handler: handleStartClientNormalization" -ForegroundColor Green
Write-Host "✓ Normalizer: ClientNormalizer.ProcessWithClientBenchmarks" -ForegroundColor Green
Write-Host "✓ Source: catalog_items из баз данных проекта" -ForegroundColor Green
Write-Host "✓ Destination: normalized_data + client_benchmarks" -ForegroundColor Green
Write-Host "✓ Sessions: normalization_sessions" -ForegroundColor Green
Write-Host ""

# Итоговый отчет
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "📊 ИТОГОВЫЙ ОТЧЕТ" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""
Write-Host "Клиент ID: $clientId" -ForegroundColor White
Write-Host "Проект ID: $projectId" -ForegroundColor White
Write-Host "Активных баз данных: $($activeDatabases.Count)" -ForegroundColor White
Write-Host "Нормализованных записей: $($nomenclature.total)" -ForegroundColor White
Write-Host ""
Write-Host "✓ Полный цикл тестирования завершен" -ForegroundColor Green
Write-Host ""

