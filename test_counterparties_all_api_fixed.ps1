# Тестовый скрипт для API получения всех контрагентов клиента
# Использование: .\test_counterparties_all_api_fixed.ps1 [client_id] [base_url]

param(
    [int]$ClientID = 1,
    [string]$BaseUrl = "http://localhost:9999"
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Тестирование API /api/counterparties/all" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$passed = 0
$failed = 0

function Test-Endpoint {
    param(
        [string]$Name,
        [string]$Url,
        [int]$ExpectedStatus = 200
    )
    
    Write-Host "Тест: $Name" -ForegroundColor Yellow
    Write-Host "URL: $Url" -ForegroundColor Gray
    
    try {
        $response = Invoke-WebRequest -Uri $Url -Method GET -UseBasicParsing -TimeoutSec 7 -ErrorAction Stop
        $status = $response.StatusCode
        $body = $response.Content | ConvertFrom-Json
        
        if ($status -eq $ExpectedStatus) {
            Write-Host "✓ PASS: Status $status" -ForegroundColor Green
            Write-Host "  Total: $($body.total)" -ForegroundColor Gray
            Write-Host "  Counterparties: $($body.counterparties.Count)" -ForegroundColor Gray
            Write-Host "  Projects: $($body.projects.Count)" -ForegroundColor Gray
            if ($body.counterparties.Count -gt 0) {
                $first = $body.counterparties[0]
                Write-Host "  First item: ID=$($first.id), Name=$($first.name), Source=$($first.source)" -ForegroundColor Gray
            }
            $script:passed++
            return $true
        } else {
            Write-Host "✗ FAIL: Expected $ExpectedStatus, got $status" -ForegroundColor Red
            $script:failed++
            return $false
        }
    } catch {
        if ($null -ne $_.Exception.Response) {
            $statusCode = $_.Exception.Response.StatusCode.value__
            Write-Host "✗ FAIL: $statusCode - $($_.Exception.Message)" -ForegroundColor Red
        } else {
            Write-Host "✗ FAIL: $($_.Exception.Message)" -ForegroundColor Red
        }
        $script:failed++
        return $false
    }
    Write-Host ""
}

# Тест 1: Получение всех контрагентов клиента
Write-Host "1. Получение всех контрагентов клиента" -ForegroundColor Magenta
Test-Endpoint -Name "Get all counterparties by client" `
    -Url "$BaseUrl/api/counterparties/all?client_id=$ClientID" `
    -ExpectedStatus 200
Write-Host ""

# Тест 2: Получение с пагинацией
Write-Host "2. Получение с пагинацией" -ForegroundColor Magenta
Test-Endpoint -Name "Get counterparties with pagination" `
    -Url "$BaseUrl/api/counterparties/all?client_id=$ClientID&offset=0&limit=10" `
    -ExpectedStatus 200
Write-Host ""

# Тест 3: Фильтр по источнику - только из баз данных
Write-Host "3. Фильтр по источнику - только из баз данных" -ForegroundColor Magenta
Test-Endpoint -Name "Get counterparties from databases only" `
    -Url "$BaseUrl/api/counterparties/all?client_id=$ClientID&source=database" `
    -ExpectedStatus 200
Write-Host ""

# Тест 4: Фильтр по источнику - только нормализованные
Write-Host "4. Фильтр по источнику - только нормализованные" -ForegroundColor Magenta
Test-Endpoint -Name "Get normalized counterparties only" `
    -Url "$BaseUrl/api/counterparties/all?client_id=$ClientID&source=normalized" `
    -ExpectedStatus 200
Write-Host ""

# Тест 5: Поиск по имени
Write-Host "5. Поиск по имени" -ForegroundColor Magenta
Test-Endpoint -Name "Search counterparties by name" `
    -Url "$BaseUrl/api/counterparties/all?client_id=$ClientID&search=ООО" `
    -ExpectedStatus 200
Write-Host ""

# Тест 6: Фильтр по проекту
Write-Host "6. Фильтр по проекту" -ForegroundColor Magenta
try {
    $projectsResponse = Invoke-WebRequest -Uri "$BaseUrl/api/clients/$ClientID/projects" -Method GET -UseBasicParsing -TimeoutSec 7 -ErrorAction Stop
    $projects = $projectsResponse.Content | ConvertFrom-Json
    if ($projects.Count -gt 0) {
        $projectID = $projects[0].id
        Test-Endpoint -Name "Get counterparties by project" `
            -Url "$BaseUrl/api/counterparties/all?client_id=$ClientID&project_id=$projectID" `
            -ExpectedStatus 200
    } else {
        Write-Host "  Пропущено: нет проектов у клиента" -ForegroundColor Yellow
    }
} catch {
    Write-Host "  Пропущено: не удалось получить проекты" -ForegroundColor Yellow
}
Write-Host ""

# Тест 7: Ошибка - отсутствует client_id
Write-Host "7. Ошибка валидации - отсутствует client_id" -ForegroundColor Magenta
try {
    $response = Invoke-WebRequest -Uri "$BaseUrl/api/counterparties/all" -Method GET -UseBasicParsing -TimeoutSec 7 -ErrorAction Stop
    Write-Host "✗ FAIL: Expected 400, got $($response.StatusCode)" -ForegroundColor Red
    $script:failed++
} catch {
    if ($null -ne $_.Exception.Response) {
        $statusCode = $_.Exception.Response.StatusCode.value__
        if ($statusCode -eq 400) {
            Write-Host "✓ PASS: Status 400 (Bad Request)" -ForegroundColor Green
            $script:passed++
        } else {
            Write-Host "✗ FAIL: Expected 400, got $statusCode" -ForegroundColor Red
            $script:failed++
        }
    } else {
        Write-Host "✗ FAIL: $($_.Exception.Message)" -ForegroundColor Red
        $script:failed++
    }
}
Write-Host ""

# Итоги
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Итоги тестирования" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
$total = $passed + $failed
Write-Host "Всего тестов: $total" -ForegroundColor White
Write-Host "Пройдено: $passed" -ForegroundColor Green
Write-Host "Провалено: $failed" -ForegroundColor Red
Write-Host ""

if ($failed -eq 0) {
    Write-Host "✓ Все тесты пройдены успешно!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "✗ Некоторые тесты провалены" -ForegroundColor Red
    exit 1
}

