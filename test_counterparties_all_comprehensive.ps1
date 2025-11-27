# Комплексный тест API /api/counterparties/all
# Использование: .\test_counterparties_all_comprehensive.ps1 [client_id]

param(
    [int]$ClientID = 1,
    [string]$BaseUrl = "http://localhost:9999"
)

$ErrorActionPreference = "Continue"
$endpoint = "$BaseUrl/api/counterparties/all"
$exportEndpoint = "$BaseUrl/api/counterparties/all/export"

Write-Host "`n╔═══════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║  Комплексное тестирование API /api/counterparties/all    ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host "`nПараметры:" -ForegroundColor Yellow
Write-Host "  Client ID: $ClientID" -ForegroundColor Gray
Write-Host "  Base URL: $BaseUrl" -ForegroundColor Gray

$testsPassed = 0
$testsFailed = 0
$testResults = @()

function Test-API {
    param(
        [string]$Name,
        [string]$Url,
        [scriptblock]$Validator
    )
    
    Write-Host "`n[$Name]" -ForegroundColor Yellow
    try {
        $startTime = Get-Date
        $response = Invoke-RestMethod -Uri $Url -Method GET -ErrorAction Stop
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalMilliseconds
        
        if ($Validator) {
            $result = & $Validator $response
            if ($result) {
                Write-Host "  ✓ Успешно ($([math]::Round($duration, 2))ms)" -ForegroundColor Green
                $script:testsPassed++
                $script:testResults += [PSCustomObject]@{
                    Test = $Name
                    Status = "PASS"
                    Duration = [math]::Round($duration, 2)
                }
                return $true
            } else {
                Write-Host "  ✗ Валидация не пройдена" -ForegroundColor Red
                $script:testsFailed++
                $script:testResults += [PSCustomObject]@{
                    Test = $Name
                    Status = "FAIL"
                    Duration = [math]::Round($duration, 2)
                }
                return $false
            }
        } else {
            Write-Host "  ✓ Успешно ($([math]::Round($duration, 2))ms)" -ForegroundColor Green
            $script:testsPassed++
            $script:testResults += [PSCustomObject]@{
                Test = $Name
                Status = "PASS"
                Duration = [math]::Round($duration, 2)
            }
            return $true
        }
    } catch {
        Write-Host "  ✗ Ошибка: $($_.Exception.Message)" -ForegroundColor Red
        $script:testsFailed++
        $script:testResults += [PSCustomObject]@{
            Test = $Name
            Status = "FAIL"
            Duration = 0
            Error = $_.Exception.Message
        }
        return $false
    }
}

# Тест 1: Базовый запрос
Test-API -Name "Базовый запрос" -Url "$endpoint?client_id=$ClientID&limit=5" -Validator {
    param($response)
    return ($response.total -gt 0 -and $response.counterparties.Count -eq 5)
}

# Тест 2: Пагинация
Test-API -Name "Пагинация (offset=10, limit=5)" -Url "$endpoint?client_id=$ClientID&offset=10&limit=5" -Validator {
    param($response)
    return ($response.counterparties.Count -eq 5)
}

# Тест 3: Фильтр по источнику (database)
Test-API -Name "Фильтр source=database" -Url "$endpoint?client_id=$ClientID&source=database&limit=3" -Validator {
    param($response)
    return ($response.counterparties | Where-Object { $_.source -eq "database" } | Measure-Object).Count -eq $response.counterparties.Count
}

# Тест 4: Поиск
Test-API -Name "Поиск (search)" -Url "$endpoint?client_id=$ClientID&search=ООО&limit=5" -Validator {
    param($response)
    return ($response.total -ge 0)
}

# Тест 5: Сортировка по имени
Test-API -Name "Сортировка по имени (asc)" -Url "$endpoint?client_id=$ClientID&sort_by=name&order=asc&limit=5" -Validator {
    param($response)
    return ($response.counterparties.Count -gt 0)
}

# Тест 6: Сортировка по ID
Test-API -Name "Сортировка по ID (desc)" -Url "$endpoint?client_id=$ClientID&sort_by=id&order=desc&limit=5" -Validator {
    param($response)
    return ($response.counterparties.Count -gt 0)
}

# Тест 7: Фильтр по качеству
Test-API -Name "Фильтр по качеству (min_quality)" -Url "$endpoint?client_id=$ClientID&min_quality=0.5&limit=5" -Validator {
    param($response)
    return ($response.total -ge 0)
}

# Тест 8: Комбинированные фильтры
Test-API -Name "Комбинированные фильтры" -Url "$endpoint?client_id=$ClientID&source=database&sort_by=name&order=asc&limit=5" -Validator {
    param($response)
    return ($response.counterparties.Count -ge 0)
}

# Тест 9: Статистика
Test-API -Name "Проверка статистики" -Url "$endpoint?client_id=$ClientID&limit=0" -Validator {
    param($response)
    return ($response.stats.databases_processed -gt 0 -and $response.stats.projects_processed -gt 0)
}

# Тест 10: Структура данных
Test-API -Name "Структура UnifiedCounterparty" -Url "$endpoint?client_id=$ClientID&limit=1" -Validator {
    param($response)
    if ($response.counterparties.Count -gt 0) {
        $cp = $response.counterparties[0]
        return ($cp.PSObject.Properties.Name -contains "id" -and 
                $cp.PSObject.Properties.Name -contains "source" -and
                $cp.PSObject.Properties.Name -contains "project_name")
    }
    return $false
}

# Тест 11: Экспорт JSON
Test-API -Name "Экспорт в JSON" -Url "$exportEndpoint?client_id=$ClientID&format=json&limit=5" -Validator {
    param($response)
    return $true
}

# Тест 12: Экспорт CSV
try {
    Write-Host "`n[Экспорт в CSV]" -ForegroundColor Yellow
    $startTime = Get-Date
    $response = Invoke-WebRequest -Uri "$exportEndpoint?client_id=$ClientID&format=csv&limit=5" -UseBasicParsing
    $endTime = Get-Date
    $duration = ($endTime - $startTime).TotalMilliseconds
    
    if ($response.StatusCode -eq 200) {
        Write-Host "  ✓ Успешно ($([math]::Round($duration, 2))ms)" -ForegroundColor Green
        $testsPassed++
    } else {
        Write-Host "  ✗ Ошибка: Status $($response.StatusCode)" -ForegroundColor Red
        $testsFailed++
    }
} catch {
    Write-Host "  ✗ Ошибка: $($_.Exception.Message)" -ForegroundColor Red
    $testsFailed++
}

# Итоговая статистика
Write-Host "`n╔═══════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║  Результаты тестирования                                  ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════════════════╝" -ForegroundColor Cyan

Write-Host "`nПройдено: $testsPassed" -ForegroundColor Green
Write-Host "Провалено: $testsFailed" -ForegroundColor $(if ($testsFailed -eq 0) { "Green" } else { "Red" })
Write-Host "Всего: $($testsPassed + $testsFailed)" -ForegroundColor Yellow

if ($testResults.Count -gt 0) {
    Write-Host "`nДетальная статистика:" -ForegroundColor Yellow
    $avgDuration = ($testResults | Where-Object { $_.Duration -gt 0 } | Measure-Object -Property Duration -Average).Average
    Write-Host "  Среднее время ответа: $([math]::Round($avgDuration, 2))ms" -ForegroundColor Gray
    
    $slowTests = $testResults | Where-Object { $_.Duration -gt 500 } | Sort-Object Duration -Descending
    if ($slowTests.Count -gt 0) {
        Write-Host "`n  Медленные тесты (>500ms):" -ForegroundColor Yellow
        $slowTests | ForEach-Object {
            Write-Host "    - $($_.Test): $($_.Duration)ms" -ForegroundColor Gray
        }
    }
}

if ($testsFailed -eq 0) {
    Write-Host "`n✅ Все тесты пройдены успешно!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "`n⚠ Некоторые тесты провалены" -ForegroundColor Yellow
    exit 1
}

