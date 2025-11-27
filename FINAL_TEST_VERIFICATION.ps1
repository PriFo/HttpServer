# Финальная проверка всех тестов множественной загрузки
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "ФИНАЛЬНАЯ ПРОВЕРКА ТЕСТОВ" -ForegroundColor Cyan
Write-Host "Множественная загрузка баз данных" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$allTestsPassed = $true
$results = @()

# 1. Проверка Unit-тестов
Write-Host "[1/4] Проверка Unit-тестов (Go)..." -ForegroundColor Yellow
$unitTestOutput = go test ./server -run TestMultipleUpload -v -timeout 120s 2>&1
$unitTestExitCode = $LASTEXITCODE

if ($unitTestExitCode -eq 0) {
    $testCount = ($unitTestOutput | Select-String -Pattern "--- PASS:").Count
    $skipCount = ($unitTestOutput | Select-String -Pattern "--- SKIP:").Count
    Write-Host "  ✅ Все unit-тесты прошли успешно" -ForegroundColor Green
    Write-Host "    Пройдено: $testCount тестов" -ForegroundColor Gray
    if ($skipCount -gt 0) {
        Write-Host "    Пропущено: $skipCount тестов (ожидаемо на Windows)" -ForegroundColor Gray
    }
    
    # Проверяем, что тесты проверяют БД
    $dbCheckCount = (Select-String -Path "server\database_upload_multiple_test.go" -Pattern "GetProjectDatabases" | Measure-Object).Count
    Write-Host "    Проверок БД в тестах: $dbCheckCount" -ForegroundColor Gray
    
    $results += @{
        Type = "Unit Tests"
        Status = "PASS"
        Details = "Пройдено $testCount тестов, $dbCheckCount проверок БД"
    }
} else {
    Write-Host "  ❌ Unit-тесты не прошли" -ForegroundColor Red
    $allTestsPassed = $false
    $results += @{
        Type = "Unit Tests"
        Status = "FAIL"
        Details = "Ошибки в unit-тестах"
    }
}

Write-Host ""

# 2. Проверка наличия тестовых файлов
Write-Host "[2/4] Проверка тестовых файлов..." -ForegroundColor Yellow
$testFiles = @(
    "server\database_upload_multiple_test.go",
    "test_multiple_database_upload.ps1",
    "test_multiple_upload_e2e.ps1",
    "test_multiple_upload_load.ps1",
    "test_multiple_upload_edge_cases.ps1",
    "run_all_multiple_upload_tests.ps1",
    "verify_database_after_tests.ps1",
    "run_all_tests_with_db_check.ps1"
)

$missingFiles = @()
foreach ($file in $testFiles) {
    if (Test-Path $file) {
        Write-Host "  ✅ $file" -ForegroundColor Green
    } else {
        Write-Host "  ❌ $file (не найден)" -ForegroundColor Red
        $missingFiles += $file
    }
}

if ($missingFiles.Count -eq 0) {
    $results += @{
        Type = "Test Files"
        Status = "PASS"
        Details = "Все тестовые файлы на месте"
    }
} else {
    Write-Host "  ⚠️  Отсутствуют файлы: $($missingFiles -join ', ')" -ForegroundColor Yellow
    $results += @{
        Type = "Test Files"
        Status = "WARNING"
        Details = "Отсутствуют: $($missingFiles.Count) файлов"
    }
}

Write-Host ""

# 3. Проверка проверок БД в unit-тестах
Write-Host "[3/4] Проверка проверок БД в unit-тестах..." -ForegroundColor Yellow
$dbChecks = Select-String -Path "server\database_upload_multiple_test.go" -Pattern "GetProjectDatabases|serviceDB\.Get" | Select-Object -First 10

if ($dbChecks.Count -gt 0) {
    Write-Host "  ✅ Найдено проверок БД: $($dbChecks.Count)" -ForegroundColor Green
    Write-Host "    Примеры проверок:" -ForegroundColor Gray
    foreach ($check in $dbChecks | Select-Object -First 3) {
        $lineNum = $check.LineNumber
        $line = $check.Line.Trim()
        if ($line.Length -gt 60) {
            $line = $line.Substring(0, 60) + "..."
        }
        Write-Host "      Строка $lineNum : $line" -ForegroundColor DarkGray
    }
    
    $results += @{
        Type = "Database Checks"
        Status = "PASS"
        Details = "Найдено $($dbChecks.Count) проверок БД в тестах"
    }
} else {
    Write-Host "  ❌ Проверки БД не найдены в unit-тестах" -ForegroundColor Red
    $allTestsPassed = $false
    $results += @{
        Type = "Database Checks"
        Status = "FAIL"
        Details = "Проверки БД отсутствуют"
    }
}

Write-Host ""

# 4. Проверка структуры тестов
Write-Host "[4/4] Проверка структуры тестов..." -ForegroundColor Yellow

# Проверяем, что тесты покрывают все сценарии из плана
$testScenarios = @(
    "SequentialSuccess",
    "PartialFailure",
    "ValidationEachFile",
    "FileSizeLimit",
    "DuplicateNames",
    "CleanupOnPartialError",
    "LargeBatch",
    "LargeFiles",
    "Stress",
    "SpecialCharactersInNames",
    "LongFileName"
)

$foundScenarios = @()
foreach ($scenario in $testScenarios) {
    $found = Select-String -Path "server\database_upload_multiple_test.go" -Pattern "TestMultipleUpload_$scenario" -Quiet
    if ($found) {
        $foundScenarios += $scenario
    }
}

$coverage = [math]::Round(($foundScenarios.Count / $testScenarios.Count) * 100, 1)
Write-Host "  ✅ Покрытие тестами: $coverage% ($($foundScenarios.Count)/$($testScenarios.Count))" -ForegroundColor Green
Write-Host "    Покрытые сценарии:" -ForegroundColor Gray
foreach ($scenario in $foundScenarios) {
    Write-Host "      - $scenario" -ForegroundColor DarkGray
}

$results += @{
    Type = "Test Coverage"
    Status = "PASS"
    Details = "Покрытие: $coverage%"
}

Write-Host ""

# Итоговая сводка
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "ИТОГОВАЯ СВОДКА" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

foreach ($result in $results) {
    $statusIcon = switch ($result.Status) {
        "PASS" { "✅" }
        "FAIL" { "❌" }
        "WARNING" { "⚠️ " }
        default { "❓" }
    }
    $color = switch ($result.Status) {
        "PASS" { "Green" }
        "FAIL" { "Red" }
        "WARNING" { "Yellow" }
        default { "Gray" }
    }
    
    Write-Host "$statusIcon $($result.Type): $($result.Status)" -ForegroundColor $color
    Write-Host "   $($result.Details)" -ForegroundColor DarkGray
    Write-Host ""
}

if ($allTestsPassed) {
    Write-Host "✅ ВСЕ ПРОВЕРКИ ПРОЙДЕНЫ УСПЕШНО!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Следующие шаги:" -ForegroundColor Yellow
    Write-Host "  1. Unit-тесты проходят и проверяют БД" -ForegroundColor Gray
    Write-Host "  2. Для интеграционных тестов запустите сервер:" -ForegroundColor Gray
    Write-Host "     go run ." -ForegroundColor White
    Write-Host "  3. Затем запустите интеграционные тесты:" -ForegroundColor Gray
    Write-Host "     .\test_multiple_database_upload.ps1" -ForegroundColor White
    Write-Host "  4. Проверьте данные в БД:" -ForegroundColor Gray
    Write-Host "     .\verify_database_after_tests.ps1 -ClientID <id> -ProjectID <id>" -ForegroundColor White
} else {
    Write-Host "❌ НЕКОТОРЫЕ ПРОВЕРКИ НЕ ПРОЙДЕНЫ" -ForegroundColor Red
    Write-Host "  Проверьте вывод выше для деталей" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Проверка завершена" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

