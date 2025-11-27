# Скрипт для нагрузочного тестирования API (PowerShell версия для Windows)
# Использует Invoke-WebRequest для отправки множественных запросов

param(
    [string]$BaseUrl = "http://localhost:9999",
    [int]$ConcurrentRequests = 10,
    [int]$TotalRequests = 100,
    [int]$TestDuration = 60
)

$ErrorActionPreference = "Stop"

Write-Host "=== Нагрузочное тестирование API ===" -ForegroundColor Green
Write-Host "Base URL: $BaseUrl"
Write-Host "Concurrent requests: $ConcurrentRequests"
Write-Host "Total requests: $TotalRequests"
Write-Host "Duration: ${TestDuration}s"
Write-Host ""

# Проверка доступности сервера
Write-Host "Проверка доступности сервера..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$BaseUrl/health" -Method Get -UseBasicParsing -TimeoutSec 5
    Write-Host "Сервер доступен" -ForegroundColor Green
} catch {
    Write-Host "Ошибка: сервер недоступен на $BaseUrl" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Функция для тестирования endpoint
function Test-Endpoint {
    param(
        [string]$Endpoint,
        [string]$Method = "GET",
        [string]$Data = ""
    )
    
    Write-Host "Тестирование: $Method $Endpoint" -ForegroundColor Yellow
    
    $success = 0
    $failed = 0
    $totalTime = 0
    $minTime = [double]::MaxValue
    $maxTime = 0
    $times = @()
    
    $startTime = Get-Date
    
    for ($i = 1; $i -le $TotalRequests; $i++) {
        $requestStart = Get-Date
        
        try {
            $params = @{
                Uri = "$BaseUrl$Endpoint"
                Method = $Method
                UseBasicParsing = $true
                TimeoutSec = 30
            }
            
            if ($Method -eq "POST" -and $Data) {
                $params.Body = $Data
                $params.ContentType = "application/json"
            }
            
            $response = Invoke-WebRequest @params
            $requestEnd = Get-Date
            $requestTime = ($requestEnd - $requestStart).TotalMilliseconds / 1000
            
            if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 300) {
                $success++
            } else {
                $failed++
                Write-Host "  Запрос $i : HTTP $($response.StatusCode)" -ForegroundColor Red
            }
            
            $totalTime += $requestTime
            $times += $requestTime
            
            if ($requestTime -lt $minTime) {
                $minTime = $requestTime
            }
            
            if ($requestTime -gt $maxTime) {
                $maxTime = $requestTime
            }
            
            # Прогресс
            if ($i % 10 -eq 0) {
                Write-Host "." -NoNewline
            }
        } catch {
            $failed++
            Write-Host "  Запрос $i : Ошибка - $($_.Exception.Message)" -ForegroundColor Red
        }
    }
    
    $endTime = Get-Date
    $duration = ($endTime - $startTime).TotalSeconds
    
    $avgTime = if ($success -gt 0) { $totalTime / $success } else { 0 }
    $successRate = if ($TotalRequests -gt 0) { ($success * 100) / $TotalRequests } else { 0 }
    $rps = if ($duration -gt 0) { $TotalRequests / $duration } else { 0 }
    
    # Вычисляем перцентили
    $sortedTimes = $times | Sort-Object
    $p95 = if ($sortedTimes.Count -gt 0) { $sortedTimes[[Math]::Floor($sortedTimes.Count * 0.95)] } else { 0 }
    $p99 = if ($sortedTimes.Count -gt 0) { $sortedTimes[[Math]::Floor($sortedTimes.Count * 0.99)] } else { 0 }
    
    Write-Host ""
    Write-Host "Результаты для $Endpoint :" -ForegroundColor Green
    Write-Host "  Успешных: $success / $TotalRequests ($([Math]::Round($successRate, 2))%)"
    Write-Host "  Неудачных: $failed"
    Write-Host "  Среднее время ответа: $([Math]::Round($avgTime, 3))s"
    Write-Host "  Минимальное время: $([Math]::Round($minTime, 3))s"
    Write-Host "  Максимальное время: $([Math]::Round($maxTime, 3))s"
    Write-Host "  P95: $([Math]::Round($p95, 3))s"
    Write-Host "  P99: $([Math]::Round($p99, 3))s"
    Write-Host "  Запросов в секунду: $([Math]::Round($rps, 2))"
    Write-Host "  Общее время: $([Math]::Round($duration, 2))s"
    Write-Host ""
    
    return @{
        Endpoint = $Endpoint
        Method = $Method
        Success = $success
        Failed = $failed
        AvgTime = $avgTime
        MinTime = $minTime
        MaxTime = $maxTime
        P95 = $p95
        P99 = $p99
        RPS = $rps
    }
}

# Создаем массив результатов
$results = @()

# Тестируем основные endpoints
Write-Host "=== Тест 1: Health Check ===" -ForegroundColor Green
$results += Test-Endpoint -Endpoint "/health" -Method "GET"

Write-Host "=== Тест 2: System Summary ===" -ForegroundColor Green
$results += Test-Endpoint -Endpoint "/api/system/summary" -Method "GET"

Write-Host "=== Тест 3: Monitoring Metrics ===" -ForegroundColor Green
$results += Test-Endpoint -Endpoint "/api/monitoring/metrics" -Method "GET"

Write-Host "=== Тест 4: Performance Metrics ===" -ForegroundColor Green
$results += Test-Endpoint -Endpoint "/api/monitoring/performance" -Method "GET"

Write-Host "=== Тест 5: Databases List ===" -ForegroundColor Green
$results += Test-Endpoint -Endpoint "/api/databases/list" -Method "GET"

# Генерируем отчет
Write-Host "=== Генерация отчета ===" -ForegroundColor Green
$reportFile = "load_test_report_$(Get-Date -Format 'yyyyMMdd_HHmmss').txt"

$report = @"
Отчет о нагрузочном тестировании
Дата: $(Get-Date)
Base URL: $BaseUrl
Конкурентных запросов: $ConcurrentRequests
Всего запросов на endpoint: $TotalRequests

Результаты:
---
"@

foreach ($result in $results) {
    $report += "`n$($result.Endpoint) ($($result.Method)):`n"
    $report += "  Успешных: $($result.Success) / $TotalRequests`n"
    $report += "  Неудачных: $($result.Failed)`n"
    $report += "  Среднее время: $([Math]::Round($result.AvgTime, 3))s`n"
    $report += "  P95: $([Math]::Round($result.P95, 3))s`n"
    $report += "  P99: $([Math]::Round($result.P99, 3))s`n"
    $report += "  RPS: $([Math]::Round($result.RPS, 2))`n"
}

$report | Out-File -FilePath $reportFile -Encoding UTF8

Write-Host "Отчет сохранен в: $reportFile" -ForegroundColor Green
Write-Host ""
Write-Host "Нагрузочное тестирование завершено" -ForegroundColor Green

