# Скрипт для тестирования полного цикла нормализации данных
# Автоматизированная проверка: создание сущностей → запуск нормализации → мониторинг → проверка результатов
# Использование: .\test_normalization_full_cycle.ps1

param(
    [string]$BaseUrl = "http://localhost:9999",
    [int]$PollInterval = 10,  # Интервал опроса статуса в секундах
    [int]$MaxWaitTime = 3600, # Максимальное время ожидания завершения в секундах (1 час)
    [string]$ServiceDbPath = "data/service.db"  # Путь к service database
)

$ErrorActionPreference = "Stop"
$global:TestResults = @{
    ClientID = $null
    ProjectID = $null
    DatabaseID = $null
    DatabasePath = $null
    SessionID = $null
    StartTime = Get-Date
    Steps = @()
    Errors = @()
    FinalStatus = "Unknown"
}

# Цветной вывод
function Write-Step {
    param([string]$Message, [string]$Color = "Yellow")
    Write-Host "$(Get-Date -Format 'HH:mm:ss') [$Message]" -ForegroundColor $Color
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Error-Custom {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
    $global:TestResults.Errors += $Message
}

function Write-Info {
    param([string]$Message)
    Write-Host "  → $Message" -ForegroundColor Gray
}

# HTTP запросы
function Invoke-ApiRequest {
    param(
        [string]$Method,
        [string]$Uri,
        [object]$Body = $null,
        [int]$Timeout = 30
    )
    
    try {
        $headers = @{
            "Content-Type" = "application/json"
        }
        
        $params = @{
            Method = $Method
            Uri = $Uri
            Headers = $headers
            TimeoutSec = $Timeout
            ErrorAction = "Stop"
        }
        
        if ($Body) {
            $params.Body = ($Body | ConvertTo-Json -Depth 10 -Compress)
        }
        
        $response = Invoke-RestMethod @params
        return @{
            Success = $true
            StatusCode = 200
            Data = $response
        }
    }
    catch {
        $statusCode = 0
        if ($_.Exception.Response) {
            $statusCode = [int]$_.Exception.Response.StatusCode.value__
        }
        
        $errorMessage = $_.Exception.Message
        if ($_.Exception.Response) {
            try {
                $stream = $_.Exception.Response.GetResponseStream()
                $reader = New-Object System.IO.StreamReader($stream)
                $responseBody = $reader.ReadToEnd()
                if ($responseBody) {
                    $errorMessage += " | Response: $responseBody"
                }
            } catch {}
        }
        
        return @{
            Success = $false
            StatusCode = $statusCode
            Error = $errorMessage
            Data = $null
        }
    }
}

# Получение session_id из service DB через SQLite
function Get-NormalizationSessionID {
    param([int]$DatabaseID)
    
    # Пробуем несколько возможных путей к service.db
    $possiblePaths = @(
        $ServiceDbPath,
        "service.db",
        "data\service.db",
        ".\data\service.db"
    )
    
    $serviceDbFile = $null
    foreach ($path in $possiblePaths) {
        if (Test-Path $path) {
            $serviceDbFile = (Resolve-Path $path).Path
            Write-Info "Найден service.db: $serviceDbFile"
            break
        }
    }
    
    if (-not $serviceDbFile) {
        Write-Info "Service DB не найден. Попробуем получить session_id через API или мониторинг."
        return $null
    }
    
    # Проверяем наличие sqlite3
    $sqlite3Path = $null
    $sqlite3Paths = @(
        "sqlite3",
        "sqlite3.exe",
        "C:\Program Files\SQLite\sqlite3.exe",
        "C:\sqlite3\sqlite3.exe"
    )
    
    foreach ($path in $sqlite3Paths) {
        try {
            $null = Get-Command $path -ErrorAction SilentlyContinue
            $sqlite3Path = $path
            break
        } catch {
            continue
        }
    }
    
    if (-not $sqlite3Path) {
        Write-Info "sqlite3 не найден. Попробуем получить session_id через API."
        return $null
    }
    
    try {
        # Выполняем SQL запрос для получения последней сессии для database_id
        $query = "SELECT id FROM normalization_sessions WHERE project_database_id = $DatabaseID ORDER BY started_at DESC LIMIT 1;"
        $result = & $sqlite3Path $serviceDbFile $query 2>&1
        
        if ($LASTEXITCODE -eq 0 -and $result -match '^\d+$') {
            $sessionID = [int]$result.Trim()
            Write-Success "Получен session_id из БД: $sessionID"
            return $sessionID
        }
    }
    catch {
        Write-Info "Не удалось выполнить SQL запрос: $_"
    }
    
    return $null
}

# Шаг 1: Создание клиента
function Step-CreateClient {
    Write-Step "Шаг 1: Создание клиента" "Cyan"
    
    $timestamp = Get-Date -Format 'yyyyMMddHHmmss'
    $clientData = @{
        name = "Test Client Normalization $timestamp"
        legal_name = "ООО Тестовый клиент для нормализации"
        description = "Автоматически созданный клиент для тестирования полного цикла нормализации"
        contact_email = "test-normalization@example.com"
        contact_phone = "+7 (999) 123-45-67"
        tax_id = "1234567890"
        country = "RU"
    }
    
    $response = Invoke-ApiRequest -Method "POST" -Uri "$BaseUrl/api/clients" -Body $clientData
    
    if (-not $response.Success) {
        Write-Error-Custom "Не удалось создать клиента: $($response.Error)"
        throw "Failed to create client"
    }
    
    $clientID = $response.Data.id
    $global:TestResults.ClientID = $clientID
    Write-Success "Клиент создан успешно. ID: $clientID"
    $global:TestResults.Steps += @{
        Step = "CreateClient"
        Status = "Success"
        Data = $response.Data
        Timestamp = Get-Date
    }
    
    return $clientID
}

# Шаг 2: Создание проекта
function Step-CreateProject {
    param([int]$ClientID)
    
    Write-Step "Шаг 2: Создание проекта для клиента $ClientID" "Cyan"
    
    $timestamp = Get-Date -Format 'yyyyMMddHHmmss'
    $projectData = @{
        name = "Test Project Normalization $timestamp"
        project_type = "normalization"
        description = "Тестовый проект для проверки полного цикла нормализации"
        source_system = "1C"
        target_quality_score = 0.9
    }
    
    $response = Invoke-ApiRequest -Method "POST" -Uri "$BaseUrl/api/clients/$ClientID/projects" -Body $projectData
    
    if (-not $response.Success) {
        Write-Error-Custom "Не удалось создать проект: $($response.Error)"
        throw "Failed to create project"
    }
    
    $projectID = $response.Data.id
    $global:TestResults.ProjectID = $projectID
    Write-Success "Проект создан успешно. ID: $projectID"
    $global:TestResults.Steps += @{
        Step = "CreateProject"
        Status = "Success"
        Data = $response.Data
        Timestamp = Get-Date
    }
    
    return $projectID
}

# Шаг 3: Создание базы данных
function Step-CreateDatabase {
    param([int]$ClientID, [int]$ProjectID)
    
    Write-Step "Шаг 3: Создание базы данных для проекта $ProjectID" "Cyan"
    
    $timestamp = Get-Date -Format 'yyyyMMddHHmmss'
    $dbName = "test_normalization_$timestamp"
    
    # Пробуем использовать существующую тестовую БД или создаем путь для новой
    $testDbPaths = @(
        "1c_data.db",
        "data\1c_data.db",
        ".\data\1c_data.db"
    )
    
    $dbPath = $null
    foreach ($path in $testDbPaths) {
        if (Test-Path $path) {
            $dbPath = (Resolve-Path $path).Path
            Write-Info "Используем существующую БД: $dbPath"
            break
        }
    }
    
    if (-not $dbPath) {
        # Если БД нет, создаем путь для новой (путь будет пустым, БД создастся при загрузке)
        $dbPath = ""
        Write-Info "Тестовая БД не найдена. Будет создана новая."
    }
    
    $databaseData = @{
        name = $dbName
        file_path = $dbPath
        description = "Тестовая база данных для проверки нормализации"
    }
    
    $response = Invoke-ApiRequest -Method "POST" -Uri "$BaseUrl/api/clients/$ClientID/projects/$ProjectID/databases" -Body $databaseData
    
    if (-not $response.Success) {
        Write-Error-Custom "Не удалось создать базу данных: $($response.Error)"
        throw "Failed to create database"
    }
    
    $databaseID = $response.Data.id
    $actualDbPath = if ($response.Data.file_path) { $response.Data.file_path } else { $dbPath }
    
    $global:TestResults.DatabaseID = $databaseID
    $global:TestResults.DatabasePath = $actualDbPath
    Write-Success "База данных создана успешно. ID: $databaseID, Path: $actualDbPath"
    $global:TestResults.Steps += @{
        Step = "CreateDatabase"
        Status = "Success"
        Data = $response.Data
        Timestamp = Get-Date
    }
    
    return $databaseID
}

# Шаг 4: Запуск нормализации
function Step-StartNormalization {
    param([int]$ClientID, [int]$ProjectID)
    
    Write-Step "Шаг 4: Запуск нормализации для проекта $ProjectID" "Cyan"
    
    $options = @{}  # Можно добавить опции здесь
    
    $response = Invoke-ApiRequest -Method "POST" -Uri "$BaseUrl/api/clients/$ClientID/projects/$ProjectID/normalization/start" -Body $options -Timeout 60
    
    if (-not $response.Success) {
        Write-Error-Custom "Не удалось запустить нормализацию: $($response.Error)"
        throw "Failed to start normalization"
    }
    
    Write-Success "Нормализация запущена"
    Write-Info "Ответ сервера: $($response.Data | ConvertTo-Json -Depth 3)"
    $global:TestResults.Steps += @{
        Step = "StartNormalization"
        Status = "Success"
        Data = $response.Data
        Timestamp = Get-Date
    }
    
    # Ждем немного, чтобы сессия успела создать
    Write-Info "Ожидание создания сессии нормализации (5 секунд)..."
    Start-Sleep -Seconds 5
}

# Получение session_id через API
function Get-SessionIDFromAPI {
    param([int]$ClientID, [int]$ProjectID, [int]$DatabaseID)
    
    try {
        $response = Invoke-ApiRequest -Method "GET" -Uri "$BaseUrl/api/clients/$ClientID/projects/$ProjectID/normalization/sessions" -Timeout 30
        
        if ($response.Success -and $response.Data) {
            # Ищем сессию по database_id в списке сессий
            $sessions = $response.Data.sessions
            if ($sessions -and $sessions.Count -gt 0) {
                $targetSession = $sessions | Where-Object { $_.project_database_id -eq $DatabaseID } | Sort-Object -Property started_at -Descending | Select-Object -First 1
                
                if ($targetSession -and $targetSession.id) {
                    Write-Info "Найдена сессия через API: ID=$($targetSession.id), Status=$($targetSession.status)"
                    return $targetSession.id
                }
            }
        }
    }
    catch {
        Write-Info "Не удалось получить сессии через API: $_"
    }
    
    return $null
}

# Шаг 5: Получение session_id
function Step-GetSessionID {
    param([int]$ClientID, [int]$ProjectID, [int]$DatabaseID)
    
    Write-Step "Шаг 5: Получение session_id для database_id $DatabaseID" "Cyan"
    
    # Способ 1: Пробуем получить через SQL
    Write-Info "Попытка 1: Получение session_id через SQL запрос к service DB..."
    $sessionID = Get-NormalizationSessionID -DatabaseID $DatabaseID
    
    if ($sessionID) {
        $global:TestResults.SessionID = $sessionID
        Write-Success "session_id получен через SQL: $sessionID"
        return $sessionID
    }
    
    # Способ 2: Пробуем через API (список сессий проекта)
    Write-Info "Попытка 2: Получение session_id через API (список сессий проекта)..."
    $sessionID = Get-SessionIDFromAPI -ClientID $ClientID -ProjectID $ProjectID -DatabaseID $DatabaseID
    
    if ($sessionID) {
        $global:TestResults.SessionID = $sessionID
        Write-Success "session_id получен через API: $sessionID"
        return $sessionID
    }
    
    # Если не получилось, ждем и пробуем еще раз
    Write-Info "Ожидание создания сессии (10 секунд)..."
    Start-Sleep -Seconds 10
    
    # Повторная попытка через SQL
    Write-Info "Попытка 3: Повторное получение session_id через SQL..."
    $sessionID = Get-NormalizationSessionID -DatabaseID $DatabaseID
    
    if ($sessionID) {
        $global:TestResults.SessionID = $sessionID
        Write-Success "session_id получен через SQL (повторная попытка): $sessionID"
        return $sessionID
    }
    
    # Повторная попытка через API
    Write-Info "Попытка 4: Повторное получение session_id через API..."
    $sessionID = Get-SessionIDFromAPI -ClientID $ClientID -ProjectID $ProjectID -DatabaseID $DatabaseID
    
    if ($sessionID) {
        $global:TestResults.SessionID = $sessionID
        Write-Success "session_id получен через API (повторная попытка): $sessionID"
        return $sessionID
    }
    
    Write-Info "Не удалось получить session_id ни через SQL, ни через API. Продолжим мониторинг через статус проекта."
    $global:TestResults.Steps += @{
        Step = "GetSessionID"
        Status = "Warning"
        Message = "Session ID не получен, будет использован мониторинг через статус проекта"
        Timestamp = Get-Date
    }
    
    return $null
}

# Шаг 6: Мониторинг прогресса
function Step-MonitorProgress {
    param([int]$ClientID, [int]$ProjectID, [int]$SessionID = 0)
    
    Write-Step "Шаг 6: Мониторинг прогресса нормализации" "Cyan"
    
    $startTime = Get-Date
    $lastStatus = $null
    $statusHistory = @()
    
    Write-Info "Интервал опроса: $PollInterval секунд"
    Write-Info "Максимальное время ожидания: $MaxWaitTime секунд"
    
    while ($true) {
        $elapsed = (Get-Date) - $startTime
        
        if ($elapsed.TotalSeconds -gt $MaxWaitTime) {
            Write-Error-Custom "Превышено максимальное время ожидания ($MaxWaitTime секунд)"
            break
        }
        
        $response = Invoke-ApiRequest -Method "GET" -Uri "$BaseUrl/api/clients/$ClientID/projects/$ProjectID/normalization/status" -Timeout 30
        
        if (-not $response.Success) {
            Write-Error-Custom "Ошибка при получении статуса: $($response.Error)"
            Start-Sleep -Seconds $PollInterval
            continue
        }
        
        $status = $response.Data
        $statusHistory += @{
            Timestamp = Get-Date
            Status = $status
        }
        
        $isRunning = if ($status.is_running -ne $null) { $status.is_running } else { $false }
        $processed = if ($status.processed -ne $null) { $status.processed } else { 0 }
        $total = if ($status.total -ne $null) { $status.total } else { 0 }
        $success = if ($status.success -ne $null) { $status.success } else { 0 }
        $errors = if ($status.errors -ne $null) { $status.errors } else { 0 }
        $progress = if ($status.progress -ne $null) { $status.progress } else { 0 }
        
        $timeStr = "{0:mm\:ss}" -f $elapsed
        
        if ($total -gt 0) {
            Write-Host "  [$timeStr] Статус: $(if ($isRunning) { 'Running' } else { 'Stopped' }) | Обработано: $processed/$total | Успешно: $success | Ошибок: $errors | Прогресс: $([math]::Round($progress, 1))%" -ForegroundColor $(if ($isRunning) { "Yellow" } else { "Green" })
        } else {
            Write-Host "  [$timeStr] Статус: $(if ($isRunning) { 'Running' } else { 'Stopped' }) | Обработано: $processed | Успешно: $success | Ошибок: $errors" -ForegroundColor $(if ($isRunning) { "Yellow" } else { "Green" })
        }
        
        # Сохраняем последний статус
        $lastStatus = $status
        
        # Проверяем, завершилась ли нормализация
        if (-not $isRunning) {
            if ($processed -gt 0 -or $success -gt 0) {
                Write-Success "Нормализация завершена"
                $global:TestResults.FinalStatus = "Completed"
            } elseif ($errors -gt 0) {
                Write-Error-Custom "Нормализация завершилась с ошибками"
                $global:TestResults.FinalStatus = "Failed"
            } else {
                Write-Info "Нормализация не запущена или не обработала данные"
                $global:TestResults.FinalStatus = "NoData"
            }
            break
        }
        
        Start-Sleep -Seconds $PollInterval
    }
    
    $global:TestResults.Steps += @{
        Step = "MonitorProgress"
        Status = "Success"
        StatusHistory = $statusHistory
        LastStatus = $lastStatus
        Duration = (Get-Date) - $startTime
        Timestamp = Get-Date
    }
    
    return $lastStatus
}

# Шаг 7: Получение истории сессии (если есть session_id)
function Step-GetSessionHistory {
    param([int]$SessionID)
    
    if (-not $SessionID -or $SessionID -eq 0) {
        Write-Info "session_id не доступен, пропускаем получение истории"
        return
    }
    
    Write-Step "Шаг 7: Получение истории сессии $SessionID" "Cyan"
    
    $response = Invoke-ApiRequest -Method "GET" -Uri "$BaseUrl/api/normalization/history?session_id=$SessionID" -Timeout 30
    
    if (-not $response.Success) {
        Write-Error-Custom "Не удалось получить историю сессии: $($response.Error)"
        return
    }
    
    Write-Success "История сессии получена"
    Write-Info "Количество стадий: $(if ($response.Data.stages) { $response.Data.stages.Count } else { 0 })"
    $global:TestResults.Steps += @{
        Step = "GetSessionHistory"
        Status = "Success"
        Data = $response.Data
        Timestamp = Get-Date
    }
}

# Шаг 8: Экспорт нормализованных данных
function Step-ExportData {
    param([string]$DatabasePath)
    
    Write-Step "Шаг 8: Экспорт нормализованных данных" "Cyan"
    
    # Если путь к БД не указан, пытаемся получить его через API используя db_id
    if (-not $DatabasePath) {
        Write-Info "Путь к БД не указан, пробуем получить из информации о базе данных..."
        
        # Пробуем получить информацию о базе данных через API
        if ($global:TestResults.DatabaseID -and $global:TestResults.ProjectID -and $global:TestResults.ClientID) {
            $dbResponse = Invoke-ApiRequest -Method "GET" -Uri "$BaseUrl/api/clients/$($global:TestResults.ClientID)/projects/$($global:TestResults.ProjectID)/databases" -Timeout 30
            if ($dbResponse.Success -and $dbResponse.Data.databases) {
                $targetDb = $dbResponse.Data.databases | Where-Object { $_.id -eq $global:TestResults.DatabaseID } | Select-Object -First 1
                if ($targetDb -and $targetDb.file_path) {
                    $DatabasePath = $targetDb.file_path
                    Write-Info "Получен путь к БД из API: $DatabasePath"
                }
            }
        }
    }
    
    if (-not $DatabasePath) {
        Write-Info "Путь к БД не доступен, пропускаем экспорт"
        return
    }
    
    # Кодируем путь для URL и выполняем экспорт
    $encodedPath = [uri]::EscapeDataString($DatabasePath)
    $exportUri = "$BaseUrl/api/normalization/export?database=$encodedPath"
    Write-Info "Выполняем экспорт через: $exportUri"
    $response = Invoke-ApiRequest -Method "GET" -Uri $exportUri -Timeout 60
    
    if (-not $response.Success) {
        Write-Error-Custom "Не удалось экспортировать данные: $($response.Error)"
        return
    }
    
    Write-Success "Экспорт данных выполнен успешно"
    Write-Info "Размер экспорта: $(if ($response.Data -is [Array]) { "$($response.Data.Count) записей" } else { "N/A" })"
    $global:TestResults.Steps += @{
        Step = "ExportData"
        Status = "Success"
        RecordCount = if ($response.Data -is [Array]) { $response.Data.Count } else { 0 }
        Timestamp = Get-Date
    }
}

# Шаг 9: Проверка логов маппинга контрагентов
function Step-CheckCounterpartyMapping {
    Write-Step "Шаг 9: Проверка логов маппинга контрагентов" "Cyan"
    
    # Проверяем логи на наличие записей о маппинге контрагентов
    # В реальной системе логи могут быть в файлах или через API
    $logFiles = @(
        "backend.log",
        "logs\backend.log",
        ".\logs\backend.log",
        "server.log",
        "logs\server.log"
    )
    
    $foundLogs = $false
    $mappingErrors = @()
    $mappingWarnings = @()
    $mappingSuccess = $false
    $mappingInfo = @()
    
    foreach ($logFile in $logFiles) {
        if (Test-Path $logFile) {
            Write-Info "Проверка лог-файла: $logFile"
            $logContent = Get-Content $logFile -Tail 2000 -ErrorAction SilentlyContinue
            
            if ($logContent) {
                $foundLogs = $true
                
                # Ищем записи о counterparty_mapper и маппинге контрагентов
                $mapperEntries = $logContent | Select-String -Pattern "counterparty_mapper|auto-map counterparties|map counterparties|counterparty mapping" -CaseSensitive:$false
                
                if ($mapperEntries) {
                    Write-Success "Найдены записи о маппинге контрагентов ($($mapperEntries.Count) записей)"
                    
                    # Проверяем на критическую ошибку "database is closed"
                    $closedErrors = $logContent | Select-String -Pattern "database is closed|database.*closed" -CaseSensitive:$false
                    if ($closedErrors) {
                        Write-Error-Custom "Обнаружена критическая ошибка 'database is closed' в логах"
                        $mappingErrors += "database is closed error found"
                        $closedErrors | Select-Object -First 3 | ForEach-Object {
                            Write-Info "  ERROR: $_"
                        }
                    }
                    
                    # Проверяем на предупреждение "Failed to auto-map counterparties"
                    $failedMapping = $logContent | Select-String -Pattern "Failed to auto-map counterparties" -CaseSensitive:$false
                    if ($failedMapping) {
                        Write-Info "Обнаружено предупреждение о неудачном маппинге:"
                        $failedMapping | Select-Object -First 3 | ForEach-Object {
                            Write-Info "  WARN: $_"
                            $mappingWarnings += $_.Line
                        }
                    }
                    
                    # Проверяем успешное начало маппинга
                    $startEntries = $logContent | Select-String -Pattern "Starting counterparty mapping|Successfully auto-mapped counterparties|Mapped counterparties from database" -CaseSensitive:$false
                    if ($startEntries) {
                        Write-Success "Найдены записи о начале/успешном выполнении маппинга:"
                        $startEntries | Select-Object -First 3 | ForEach-Object {
                            Write-Info "  INFO: $_"
                            $mappingInfo += $_.Line
                        }
                    }
                    
                    # Проверяем завершение маппинга
                    $completedEntries = $logContent | Select-String -Pattern "Completed counterparty mapping|Successfully.*auto-mapped|mapping.*completed" -CaseSensitive:$false
                    if ($completedEntries) {
                        Write-Success "Маппинг контрагентов завершен успешно"
                        $mappingSuccess = $true
                        $completedEntries | Select-Object -First 3 | ForEach-Object {
                            Write-Info "  SUCCESS: $_"
                        }
                    }
                    
                    # Проверяем на наличие catalog items
                    $catalogItems = $logContent | Select-String -Pattern "Found catalog items|No catalog items found" -CaseSensitive:$false
                    if ($catalogItems) {
                        Write-Info "Информация о catalog items:"
                        $catalogItems | Select-Object -First 2 | ForEach-Object {
                            Write-Info "  INFO: $_"
                        }
                    }
                    
                    # Показываем последние записи о маппинге
                    Write-Info "Последние записи о маппинге контрагентов (топ 5):"
                    $mapperEntries | Select-Object -Last 5 | ForEach-Object {
                        Write-Info "  $_"
                    }
                } else {
                    Write-Info "Записи о маппинге контрагентов (counterparty_mapper) не найдены в логах"
                }
            }
            break
        }
    }
    
    if (-not $foundLogs) {
        Write-Info "Лог-файлы не найдены. Проверка логов пропущена."
    }
    
    # Финальный вывод результатов проверки
    Write-Host ""
    if ($mappingErrors.Count -gt 0) {
        Write-Error-Custom "Обнаружены ошибки при проверке маппинга контрагентов ($($mappingErrors.Count) ошибок)"
    } elseif ($mappingWarnings.Count -gt 0) {
        Write-Info "Обнаружены предупреждения при проверке маппинга контрагентов ($($mappingWarnings.Count) предупреждений)"
    } elseif ($mappingSuccess) {
        Write-Success "Маппинг контрагентов выполнен успешно"
    } else {
        Write-Info "Информация о маппинге контрагентов не найдена (возможно, маппинг не выполнялся или логи недоступны)"
    }
    
    $global:TestResults.Steps += @{
        Step = "CheckCounterpartyMapping"
        Status = if ($mappingErrors.Count -eq 0 -and $mappingSuccess) { "Success" } elseif ($mappingErrors.Count -gt 0) { "Error" } else { "Warning" }
        MappingSuccess = $mappingSuccess
        MappingErrors = $mappingErrors
        MappingWarnings = $mappingWarnings
        MappingInfo = $mappingInfo
        FoundLogs = $foundLogs
        Timestamp = Get-Date
    }
}

# Генерация отчета
function Generate-Report {
    Write-Step "Генерация финального отчета" "Cyan"
    
    $endTime = Get-Date
    $duration = $endTime - $global:TestResults.StartTime
    
    Write-Host ""
    Write-Host "════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
    Write-Host "                    ФИНАЛЬНЫЙ ОТЧЕТ" -ForegroundColor Cyan
    Write-Host "════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
    Write-Host ""
    
    Write-Host "Созданные сущности:" -ForegroundColor Yellow
    Write-Host "  Client ID:     $($global:TestResults.ClientID)" -ForegroundColor White
    Write-Host "  Project ID:    $($global:TestResults.ProjectID)" -ForegroundColor White
    Write-Host "  Database ID:   $($global:TestResults.DatabaseID)" -ForegroundColor White
    Write-Host "  Database Path: $($global:TestResults.DatabasePath)" -ForegroundColor White
    Write-Host ""
    
    if ($global:TestResults.SessionID) {
        Write-Host "Session ID:     $($global:TestResults.SessionID)" -ForegroundColor White
        Write-Host ""
    }
    
    Write-Host "Временные метки:" -ForegroundColor Yellow
    Write-Host "  Начало:  $($global:TestResults.StartTime.ToString('yyyy-MM-dd HH:mm:ss'))" -ForegroundColor White
    Write-Host "  Конец:   $($endTime.ToString('yyyy-MM-dd HH:mm:ss'))" -ForegroundColor White
    Write-Host "  Длительность: $($duration.ToString('hh\:mm\:ss'))" -ForegroundColor White
    Write-Host ""
    
    Write-Host "Статус выполнения:" -ForegroundColor Yellow
    $statusColor = switch ($global:TestResults.FinalStatus) {
        "Completed" { "Green" }
        "Failed" { "Red" }
        "NoData" { "Yellow" }
        default { "Gray" }
    }
    Write-Host "  $($global:TestResults.FinalStatus)" -ForegroundColor $statusColor
    Write-Host ""
    
    Write-Host "Выполненные шаги:" -ForegroundColor Yellow
    foreach ($step in $global:TestResults.Steps) {
        $stepColor = switch ($step.Status) {
            "Success" { "Green" }
            "Error" { "Red" }
            "Warning" { "Yellow" }
            default { "Gray" }
        }
        Write-Host "  [$($step.Step)] $($step.Status)" -ForegroundColor $stepColor
    }
    Write-Host ""
    
    if ($global:TestResults.Errors.Count -gt 0) {
        Write-Host "Ошибки:" -ForegroundColor Red
        foreach ($error in $global:TestResults.Errors) {
            Write-Host "  - $error" -ForegroundColor Red
        }
        Write-Host ""
    }
    
    # Сохранение отчета в JSON файл
    $reportPath = "normalization_test_report_$(Get-Date -Format 'yyyyMMdd_HHmmss').json"
    $reportData = @{
        TestResults = $global:TestResults
        Summary = @{
            TotalSteps = $global:TestResults.Steps.Count
            SuccessfulSteps = ($global:TestResults.Steps | Where-Object { $_.Status -eq "Success" }).Count
            FailedSteps = ($global:TestResults.Steps | Where-Object { $_.Status -eq "Error" }).Count
            WarningSteps = ($global:TestResults.Steps | Where-Object { $_.Status -eq "Warning" }).Count
            Duration = $duration.TotalSeconds
            FinalStatus = $global:TestResults.FinalStatus
        }
    }
    
    try {
        $reportData | ConvertTo-Json -Depth 10 | Out-File -FilePath $reportPath -Encoding UTF8
        Write-Success "Отчет сохранен в файл: $reportPath"
    } catch {
        Write-Error-Custom "Не удалось сохранить отчет: $_"
    }
    
    Write-Host ""
    Write-Host "════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
    Write-Host ""
    
    # Итоговое заключение
    if ($global:TestResults.FinalStatus -eq "Completed" -and $global:TestResults.Errors.Count -eq 0) {
        Write-Host "✓✓✓ ТЕСТ ПРОЙДЕН УСПЕШНО ✓✓✓" -ForegroundColor Green
        return 0
    } elseif ($global:TestResults.FinalStatus -eq "Failed" -or $global:TestResults.Errors.Count -gt 0) {
        Write-Host "✗✗✗ ТЕСТ ЗАВЕРШИЛСЯ С ОШИБКАМИ ✗✗✗" -ForegroundColor Red
        return 1
    } else {
        Write-Host "⚠⚠⚠ ТЕСТ ЗАВЕРШЕН С ПРЕДУПРЕЖДЕНИЯМИ ⚠⚠⚠" -ForegroundColor Yellow
        return 2
    }
}

# Основной цикл выполнения
function Main {
    Write-Host ""
    Write-Host "╔═══════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║   ТЕСТИРОВАНИЕ ПОЛНОГО ЦИКЛА НОРМАЛИЗАЦИИ ДАННЫХ            ║" -ForegroundColor Cyan
    Write-Host "╚═══════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
    Write-Info "Base URL: $BaseUrl"
    Write-Info "Poll Interval: $PollInterval секунд"
    Write-Info "Max Wait Time: $MaxWaitTime секунд"
    Write-Host ""
    
    try {
        # Выполняем все шаги последовательно
        $clientID = Step-CreateClient
        $projectID = Step-CreateProject -ClientID $clientID
        $databaseID = Step-CreateDatabase -ClientID $clientID -ProjectID $projectID
        Step-StartNormalization -ClientID $clientID -ProjectID $projectID
        $sessionID = Step-GetSessionID -ClientID $clientID -ProjectID $projectID -DatabaseID $databaseID
        Step-MonitorProgress -ClientID $clientID -ProjectID $projectID -SessionID $sessionID
        
        # Дополнительные проверки
        Step-GetSessionHistory -SessionID $sessionID
        Step-ExportData -DatabasePath $global:TestResults.DatabasePath
        Step-CheckCounterpartyMapping
        
        # Генерируем отчет
        $exitCode = Generate-Report
        exit $exitCode
    }
    catch {
        Write-Error-Custom "Критическая ошибка: $_"
        $global:TestResults.FinalStatus = "Failed"
        Generate-Report
        exit 1
    }
}

# Запуск
Main

