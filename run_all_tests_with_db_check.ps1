# Скрипт для запуска всех тестов с проверкой данных в БД
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Полная проверка тестов множественной загрузки" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$baseUrl = "http://localhost:9999"
$testResults = @()

# Проверка доступности сервера
Write-Host "[Шаг 1] Проверка доступности сервера..." -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "$baseUrl/api/health" -Method GET -TimeoutSec 3 -ErrorAction Stop
    Write-Host "  OK Сервер доступен" -ForegroundColor Green
} catch {
    Write-Host "  ERROR Сервер недоступен: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "  Запустите сервер командой: go run ." -ForegroundColor Yellow
    Write-Host "  Или в отдельном окне PowerShell: Start-Process powershell -ArgumentList '-NoExit', '-Command', 'cd E:\HttpServer; go run .'" -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# Шаг 2: Запуск Unit-тестов
Write-Host "[Шаг 2] Запуск Unit-тестов..." -ForegroundColor Yellow
$unitTestOutput = go test ./server -run TestMultipleUpload -v -timeout 120s 2>&1
$unitTestSuccess = $LASTEXITCODE -eq 0

if ($unitTestSuccess) {
    Write-Host "  OK Все unit-тесты прошли успешно" -ForegroundColor Green
    
    # Подсчитываем количество тестов
    $testCount = ($unitTestOutput | Select-String -Pattern "--- PASS:").Count
    $skipCount = ($unitTestOutput | Select-String -Pattern "--- SKIP:").Count
    Write-Host "    Пройдено тестов: $testCount" -ForegroundColor Gray
    if ($skipCount -gt 0) {
        Write-Host "    Пропущено тестов: $skipCount" -ForegroundColor Gray
    }
} else {
    Write-Host "  ERROR Некоторые unit-тесты не прошли" -ForegroundColor Red
    Write-Host "    Вывод последних ошибок:" -ForegroundColor Yellow
    $unitTestOutput | Select-Object -Last 20 | ForEach-Object { Write-Host "    $_" -ForegroundColor Gray }
}

$testResults += @{
    Type = "Unit Tests"
    Success = $unitTestSuccess
    Output = $unitTestOutput
}

Write-Host ""

# Шаг 3: Запуск интеграционных тестов
Write-Host "[Шаг 3] Запуск интеграционных тестов..." -ForegroundColor Yellow
if (Test-Path "test_multiple_database_upload.ps1") {
    try {
        $integrationResults = & .\test_multiple_database_upload.ps1
        
        if ($integrationResults -and $integrationResults.Successful -eq $integrationResults.TotalTests) {
            Write-Host "  OK Все интеграционные тесты прошли успешно" -ForegroundColor Green
            Write-Host "    Всего загрузок: $($integrationResults.TotalTests)" -ForegroundColor Gray
            Write-Host "    Успешных: $($integrationResults.Successful)" -ForegroundColor Gray
            Write-Host "    Баз данных в проекте: $($integrationResults.FinalDatabaseCount)" -ForegroundColor Gray
            
            # Сохраняем ClientID и ProjectID для проверки БД
            if ($integrationResults.ClientID -and $integrationResults.ProjectID) {
                $global:testClientID = $integrationResults.ClientID
                $global:testProjectID = $integrationResults.ProjectID
                Write-Host "    ClientID: $($integrationResults.ClientID), ProjectID: $($integrationResults.ProjectID)" -ForegroundColor Gray
            }
            
            $testResults += @{
                Type = "Integration Tests"
                Success = $true
                Results = $integrationResults
            }
        } else {
            Write-Host "  WARNING Некоторые интеграционные тесты не прошли" -ForegroundColor Yellow
            if ($integrationResults) {
                Write-Host "    Успешных: $($integrationResults.Successful)/$($integrationResults.TotalTests)" -ForegroundColor Gray
            }
            
            $testResults += @{
                Type = "Integration Tests"
                Success = $false
                Results = $integrationResults
            }
        }
    } catch {
        Write-Host "  ERROR Ошибка при запуске интеграционных тестов: $($_.Exception.Message)" -ForegroundColor Red
        $testResults += @{
            Type = "Integration Tests"
            Success = $false
            Error = $_.Exception.Message
        }
    }
} else {
    Write-Host "  WARNING Файл test_multiple_database_upload.ps1 не найден" -ForegroundColor Yellow
}

Write-Host ""

# Шаг 4: Проверка данных в БД
if ($global:testClientID -and $global:testProjectID) {
    Write-Host "[Шаг 4] Проверка данных в базе данных..." -ForegroundColor Yellow
    Write-Host "  ClientID: $global:testClientID, ProjectID: $global:testProjectID" -ForegroundColor Gray
    
    try {
        $dbCheck = & .\verify_database_after_tests.ps1 -ClientID $global:testClientID -ProjectID $global:testProjectID
        Write-Host "  OK Проверка БД завершена" -ForegroundColor Green
    } catch {
        Write-Host "  ERROR Ошибка при проверке БД: $($_.Exception.Message)" -ForegroundColor Red
    }
} else {
    Write-Host "[Шаг 4] Пропуск проверки БД (нет данных о ClientID/ProjectID)" -ForegroundColor Yellow
    Write-Host "  Для ручной проверки запустите:" -ForegroundColor Gray
    Write-Host "    .\verify_database_after_tests.ps1 -ClientID <id> -ProjectID <id>" -ForegroundColor Gray
}

Write-Host ""

# Итоговая сводка
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Итоговая сводка" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

$allSuccess = $true
foreach ($result in $testResults) {
    $status = if ($result.Success) { "OK" } else { "FAIL" }
    $color = if ($result.Success) { "Green" } else { "Red" }
    Write-Host "$($result.Type): $status" -ForegroundColor $color
    
    if (-not $result.Success) {
        $allSuccess = $false
    }
}

Write-Host ""
if ($allSuccess) {
    Write-Host "Все тесты прошли успешно!" -ForegroundColor Green
} else {
    Write-Host "Некоторые тесты не прошли. Проверьте вывод выше." -ForegroundColor Yellow
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Проверка завершена" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

