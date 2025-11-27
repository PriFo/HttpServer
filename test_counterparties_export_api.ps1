# Тестовый скрипт для API экспорта контрагентов
# Использование: .\test_counterparties_export_api.ps1 [client_id] [base_url]

param(
    [int]$ClientID = 1,
    [string]$BaseUrl = "http://localhost:9999"
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Тестирование API /api/counterparties/all/export" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$passed = 0
$failed = 0
$testDir = "test_exports"
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"

# Создаем директорию для тестовых экспортов
if (-not (Test-Path $testDir)) {
    New-Item -ItemType Directory -Path $testDir | Out-Null
}

function Test-Export {
    param(
        [string]$Name,
        [string]$Url,
        [string]$Format,
        [int]$ExpectedStatus = 200
    )
    
    Write-Host "Тест: $Name" -ForegroundColor Yellow
    Write-Host "URL: $Url" -ForegroundColor Gray
    Write-Host "Формат: $Format" -ForegroundColor Gray
    
    try {
        $outputFile = "$testDir\counterparties_export_${timestamp}_${Format}.$Format"
        $response = Invoke-WebRequest -Uri $Url -Method GET -UseBasicParsing -TimeoutSec 30 -ErrorAction Stop
        $status = $response.StatusCode
        
        if ($status -eq $ExpectedStatus) {
            # Сохраняем файл
            [System.IO.File]::WriteAllBytes($outputFile, $response.Content)
            $fileSize = (Get-Item $outputFile).Length
            
            Write-Host "✓ PASS: Status $status" -ForegroundColor Green
            Write-Host "  Файл сохранен: $outputFile" -ForegroundColor Gray
            Write-Host "  Размер файла: $fileSize байт" -ForegroundColor Gray
            
            # Проверяем содержимое
            if ($Format -eq "json") {
                try {
                    $content = Get-Content $outputFile -Raw | ConvertFrom-Json
                    Write-Host "  JSON валиден" -ForegroundColor Gray
                    Write-Host "  Total: $($content.total)" -ForegroundColor Gray
                    Write-Host "  Counterparties: $($content.counterparties.Count)" -ForegroundColor Gray
                } catch {
                    Write-Host "  ⚠ JSON невалиден: $($_.Exception.Message)" -ForegroundColor Yellow
                }
            } elseif ($Format -eq "csv") {
                $lines = Get-Content $outputFile
                Write-Host "  CSV строк: $($lines.Count)" -ForegroundColor Gray
                if ($lines.Count -gt 0) {
                    Write-Host "  Заголовки: $($lines[0])" -ForegroundColor Gray
                }
            }
            
            $script:passed++
            return $true
        } else {
            Write-Host "✗ FAIL: Expected $ExpectedStatus, got $status" -ForegroundColor Red
            $script:failed++
            return $false
        }
    } catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        Write-Host "✗ FAIL: $statusCode - $($_.Exception.Message)" -ForegroundColor Red
        $script:failed++
        return $false
    }
    Write-Host ""
}

# Тест 1: Экспорт в CSV
Write-Host "1. Экспорт всех контрагентов в CSV" -ForegroundColor Magenta
Test-Export -Name "Export all counterparties to CSV" `
    -Url "$BaseUrl/api/counterparties/all/export?client_id=$ClientID&format=csv" `
    -Format "csv" `
    -ExpectedStatus 200
Write-Host ""

# Тест 2: Экспорт в JSON
Write-Host "2. Экспорт всех контрагентов в JSON" -ForegroundColor Magenta
Test-Export -Name "Export all counterparties to JSON" `
    -Url "$BaseUrl/api/counterparties/all/export?client_id=$ClientID&format=json" `
    -Format "json" `
    -ExpectedStatus 200
Write-Host ""

# Тест 3: Экспорт с фильтром по источнику (database)
Write-Host "3. Экспорт контрагентов из баз данных в CSV" -ForegroundColor Magenta
Test-Export -Name "Export database counterparties to CSV" `
    -Url "$BaseUrl/api/counterparties/all/export?client_id=$ClientID&format=csv&source=database" `
    -Format "csv" `
    -ExpectedStatus 200
Write-Host ""

# Тест 4: Экспорт с фильтром по источнику (normalized)
Write-Host "4. Экспорт нормализованных контрагентов в JSON" -ForegroundColor Magenta
Test-Export -Name "Export normalized counterparties to JSON" `
    -Url "$BaseUrl/api/counterparties/all/export?client_id=$ClientID&format=json&source=normalized" `
    -Format "json" `
    -ExpectedStatus 200
Write-Host ""

# Тест 5: Экспорт с поиском
Write-Host "5. Экспорт с поиском в CSV" -ForegroundColor Magenta
Test-Export -Name "Export with search to CSV" `
    -Url "$BaseUrl/api/counterparties/all/export?client_id=$ClientID&format=csv&search=ООО" `
    -Format "csv" `
    -ExpectedStatus 200
Write-Host ""

# Тест 6: Ошибка - отсутствует client_id
Write-Host "6. Ошибка валидации - отсутствует client_id" -ForegroundColor Magenta
try {
    $response = Invoke-WebRequest -Uri "$BaseUrl/api/counterparties/all/export?format=csv" -Method GET -UseBasicParsing -TimeoutSec 7 -ErrorAction Stop
    Write-Host "✗ FAIL: Expected 400, got $($response.StatusCode)" -ForegroundColor Red
    $script:failed++
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    if ($statusCode -eq 400) {
        Write-Host "✓ PASS: Status 400 (Bad Request)" -ForegroundColor Green
        $script:passed++
    } else {
        Write-Host "✗ FAIL: Expected 400, got $statusCode" -ForegroundColor Red
        $script:failed++
    }
}
Write-Host ""

# Тест 7: Ошибка - неверный формат
Write-Host "7. Ошибка валидации - неверный формат" -ForegroundColor Magenta
try {
    $response = Invoke-WebRequest -Uri "$BaseUrl/api/counterparties/all/export?client_id=$ClientID&format=xml" -Method GET -UseBasicParsing -TimeoutSec 7 -ErrorAction Stop
    Write-Host "✗ FAIL: Expected 400, got $($response.StatusCode)" -ForegroundColor Red
    $script:failed++
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    if ($statusCode -eq 400) {
        Write-Host "✓ PASS: Status 400 (Bad Request)" -ForegroundColor Green
        $script:passed++
    } else {
        Write-Host "✗ FAIL: Expected 400, got $statusCode" -ForegroundColor Red
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
Write-Host "Тестовые файлы сохранены в: $testDir" -ForegroundColor Cyan
Write-Host ""

if ($failed -eq 0) {
    Write-Host "✓ Все тесты пройдены успешно!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "✗ Некоторые тесты провалены" -ForegroundColor Red
    exit 1
}

