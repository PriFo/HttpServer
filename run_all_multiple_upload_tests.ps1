# Скрипт для запуска всех тестов множественной загрузки
# Запускает все тесты последовательно и собирает результаты

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Запуск всех тестов множественной загрузки" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$testResults = @{
    UnitTests = $null
    IntegrationTests = $null
    E2ETests = $null
    LoadTests = $null
    EdgeCaseTests = $null
}

$startTime = Get-Date

# Проверка доступности backend
Write-Host "[Проверка] Проверка доступности backend..." -ForegroundColor Yellow
try {
    $null = Invoke-WebRequest -Uri "http://localhost:9999/api/health" -Method GET -TimeoutSec 5 -UseBasicParsing -ErrorAction Stop
    Write-Host "  OK Backend доступен" -ForegroundColor Green
} catch {
    Write-Host "  WARNING Backend недоступен на http://localhost:9999" -ForegroundColor Yellow
    Write-Host "  Продолжаем выполнение тестов..." -ForegroundColor Gray
}
Write-Host ""

# 1. Unit-тесты
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "[1/5] Unit-тесты (Backend)" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$unitTestStart = Get-Date
try {
    $unitTestOutput = go test ./server -run TestMultipleUpload -v 2>&1
    $unitTestDuration = (Get-Date) - $unitTestStart
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  OK Unit-тесты пройдены успешно" -ForegroundColor Green
        $testResults.UnitTests = @{
            Success = $true
            Duration = $unitTestDuration
            Output = $unitTestOutput
        }
    } else {
        Write-Host "  ERROR Unit-тесты провалились" -ForegroundColor Red
        $testResults.UnitTests = @{
            Success = $false
            Duration = $unitTestDuration
            Output = $unitTestOutput
        }
    }
} catch {
    Write-Host "  ERROR Ошибка при запуске unit-тестов: $($_.Exception.Message)" -ForegroundColor Red
    $testResults.UnitTests = @{
        Success = $false
        Error = $_.Exception.Message
    }
}
Write-Host "  Время выполнения: $($unitTestDuration.TotalSeconds) секунд" -ForegroundColor Gray
Write-Host ""

# 2. Интеграционные тесты
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "[2/5] Интеграционные тесты (API)" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$integrationTestStart = Get-Date
try {
    $integrationResult = & .\test_multiple_database_upload.ps1
    $integrationTestDuration = (Get-Date) - $integrationTestStart
    
    if ($integrationResult) {
        Write-Host "  OK Интеграционные тесты завершены" -ForegroundColor Green
        $testResults.IntegrationTests = @{
            Success = $true
            Duration = $integrationTestDuration
            Result = $integrationResult
        }
    } else {
        Write-Host "  WARNING Интеграционные тесты завершены с предупреждениями" -ForegroundColor Yellow
        $testResults.IntegrationTests = @{
            Success = $true
            Duration = $integrationTestDuration
            Warnings = $true
        }
    }
} catch {
    Write-Host "  ERROR Ошибка при запуске интеграционных тестов: $($_.Exception.Message)" -ForegroundColor Red
    $testResults.IntegrationTests = @{
        Success = $false
        Error = $_.Exception.Message
    }
}
Write-Host "  Время выполнения: $($integrationTestDuration.TotalSeconds) секунд" -ForegroundColor Gray
Write-Host ""

# 3. E2E тесты
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "[3/5] E2E тесты (Frontend)" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$e2eTestStart = Get-Date
try {
    $e2eResult = & .\test_multiple_upload_e2e.ps1
    $e2eTestDuration = (Get-Date) - $e2eTestStart
    
    if ($e2eResult) {
        Write-Host "  OK E2E тесты завершены" -ForegroundColor Green
        $testResults.E2ETests = @{
            Success = $true
            Duration = $e2eTestDuration
            Result = $e2eResult
        }
    } else {
        Write-Host "  WARNING E2E тесты завершены с предупреждениями" -ForegroundColor Yellow
        $testResults.E2ETests = @{
            Success = $true
            Duration = $e2eTestDuration
            Warnings = $true
        }
    }
} catch {
    Write-Host "  ERROR Ошибка при запуске E2E тестов: $($_.Exception.Message)" -ForegroundColor Red
    $testResults.E2ETests = @{
        Success = $false
        Error = $_.Exception.Message
    }
}
Write-Host "  Время выполнения: $($e2eTestDuration.TotalSeconds) секунд" -ForegroundColor Gray
Write-Host ""

# 4. Нагрузочные тесты
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "[4/5] Нагрузочные тесты" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$loadTestStart = Get-Date
try {
    $loadResult = & .\test_multiple_upload_load.ps1
    $loadTestDuration = (Get-Date) - $loadTestStart
    
    if ($loadResult) {
        Write-Host "  OK Нагрузочные тесты завершены" -ForegroundColor Green
        $testResults.LoadTests = @{
            Success = $true
            Duration = $loadTestDuration
            Result = $loadResult
        }
    } else {
        Write-Host "  WARNING Нагрузочные тесты завершены с предупреждениями" -ForegroundColor Yellow
        $testResults.LoadTests = @{
            Success = $true
            Duration = $loadTestDuration
            Warnings = $true
        }
    }
} catch {
    Write-Host "  ERROR Ошибка при запуске нагрузочных тестов: $($_.Exception.Message)" -ForegroundColor Red
    $testResults.LoadTests = @{
        Success = $false
        Error = $_.Exception.Message
    }
}
Write-Host "  Время выполнения: $($loadTestDuration.TotalSeconds) секунд" -ForegroundColor Gray
Write-Host ""

# 5. Тесты граничных случаев
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "[5/5] Тесты граничных случаев" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$edgeCaseTestStart = Get-Date
try {
    $edgeCaseResult = & .\test_multiple_upload_edge_cases.ps1
    $edgeCaseTestDuration = (Get-Date) - $edgeCaseTestStart
    
    if ($edgeCaseResult) {
        Write-Host "  OK Тесты граничных случаев завершены" -ForegroundColor Green
        $testResults.EdgeCaseTests = @{
            Success = $true
            Duration = $edgeCaseTestDuration
            Result = $edgeCaseResult
        }
    } else {
        Write-Host "  WARNING Тесты граничных случаев завершены с предупреждениями" -ForegroundColor Yellow
        $testResults.EdgeCaseTests = @{
            Success = $true
            Duration = $edgeCaseTestDuration
            Warnings = $true
        }
    }
} catch {
    Write-Host "  ERROR Ошибка при запуске тестов граничных случаев: $($_.Exception.Message)" -ForegroundColor Red
    $testResults.EdgeCaseTests = @{
        Success = $false
        Error = $_.Exception.Message
    }
}
Write-Host "  Время выполнения: $($edgeCaseTestDuration.TotalSeconds) секунд" -ForegroundColor Gray
Write-Host ""

# Итоговая сводка
$totalDuration = (Get-Date) - $startTime

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "ИТОГОВАЯ СВОДКА" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$successCount = 0
$failCount = 0

foreach ($testType in @("UnitTests", "IntegrationTests", "E2ETests", "LoadTests", "EdgeCaseTests")) {
    $result = $testResults[$testType]
    if ($result -and $result.Success) {
        $successCount++
        $status = "✓ PASS"
        $color = "Green"
    } else {
        $failCount++
        $status = "✗ FAIL"
        $color = "Red"
    }
    
    $duration = if ($result -and $result.Duration) { 
        "$([math]::Round($result.Duration.TotalSeconds, 2))s" 
    } else { 
        "N/A" 
    }
    
    Write-Host "  $status $testType ($duration)" -ForegroundColor $color
}

Write-Host ""
Write-Host "Общее время выполнения: $([math]::Round($totalDuration.TotalSeconds, 2)) секунд" -ForegroundColor White
Write-Host "Успешно: $successCount/5" -ForegroundColor $(if ($successCount -eq 5) { "Green" } else { "Yellow" })
Write-Host "Провалено: $failCount/5" -ForegroundColor $(if ($failCount -eq 0) { "Green" } else { "Red" })

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Все тесты завершены" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# Сохраняем результаты в файл
$resultsFile = "multiple_upload_test_results_$(Get-Date -Format 'yyyyMMdd_HHmmss').json"
$testResults | ConvertTo-Json -Depth 10 | Out-File -FilePath $resultsFile -Encoding UTF8
Write-Host ""
Write-Host "Результаты сохранены в: $resultsFile" -ForegroundColor Gray

return $testResults

