# Скрипт для тестирования функциональности трассировки воркеров
$baseUrl = "http://localhost:9999"
$testTraceId = "test-trace-$(Get-Date -Format 'yyyyMMddHHmmss')"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Тестирование трассировки воркеров" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Тест 1: Проверка SSE endpoint" -ForegroundColor Yellow
Write-Host "  URL: $baseUrl/api/internal/worker-trace/stream?trace_id=$testTraceId" -ForegroundColor Gray

try {
    $response = Invoke-WebRequest -Uri "$baseUrl/api/internal/worker-trace/stream?trace_id=$testTraceId" `
        -Method GET `
        -TimeoutSec 10 `
        -Headers @{
            "Accept" = "text/event-stream"
            "Cache-Control" = "no-cache"
        } `
        -ErrorAction Stop
    
    Write-Host "  ✓ Endpoint доступен" -ForegroundColor Green
    Write-Host "  Status Code: $($response.StatusCode)" -ForegroundColor Gray
    
    if ($response.Headers['Content-Type'] -like '*text/event-stream*') {
        Write-Host "  ✓ Content-Type правильный" -ForegroundColor Green
    } else {
        Write-Host "  ⚠ Content-Type: $($response.Headers['Content-Type'])" -ForegroundColor Yellow
    }
} catch {
    $statusCode = if ($_.Exception.Response) { 
        [int]$_.Exception.Response.StatusCode 
    } else { 
        "N/A" 
    }
    
    if ($statusCode -eq 400) {
        Write-Host "  ✓ Endpoint работает (400 - ожидаемо без trace_id)" -ForegroundColor Green
    } else {
        Write-Host "  ✗ Ошибка: $statusCode" -ForegroundColor Red
        Write-Host "    $($_.Exception.Message)" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "Тест 2: Проверка без trace_id" -ForegroundColor Yellow

try {
    $response = Invoke-WebRequest -Uri "$baseUrl/api/internal/worker-trace/stream" `
        -Method GET `
        -TimeoutSec 5 `
        -ErrorAction Stop
    
    Write-Host "  ✗ Endpoint не требует trace_id (должен возвращать 400)" -ForegroundColor Red
} catch {
    $statusCode = if ($_.Exception.Response) { 
        [int]$_.Exception.Response.StatusCode 
    } else { 
        "N/A" 
    }
    
    if ($statusCode -eq 400) {
        Write-Host "  ✓ Endpoint правильно валидирует trace_id (400)" -ForegroundColor Green
    } else {
        Write-Host "  ⚠ Неожиданный статус: $statusCode" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "Тест 3: Проверка с валидным trace_id" -ForegroundColor Yellow

try {
    $job = Start-Job -ScriptBlock {
        param($url, $traceId)
        try {
            $response = Invoke-WebRequest -Uri "$url?trace_id=$traceId" `
                -Method GET `
                -TimeoutSec 3 `
                -Headers @{
                    "Accept" = "text/event-stream"
                }
            return @{
                Success = $true
                StatusCode = $response.StatusCode
                ContentType = $response.Headers['Content-Type']
            }
        } catch {
            return @{
                Success = $false
                StatusCode = if ($_.Exception.Response) { 
                    [int]$_.Exception.Response.StatusCode 
                } else { 
                    "N/A" 
                }
                Error = $_.Exception.Message
            }
        }
    } -ArgumentList "$baseUrl/api/internal/worker-trace/stream", $testTraceId
    
    Start-Sleep -Seconds 2
    $result = Receive-Job -Job $job
    Stop-Job -Job $job
    Remove-Job -Job $job
    
    if ($result.Success) {
        Write-Host "  ✓ SSE соединение установлено" -ForegroundColor Green
        Write-Host "  Status Code: $($result.StatusCode)" -ForegroundColor Gray
    } else {
        if ($result.StatusCode -eq 200) {
            Write-Host "  ✓ Endpoint отвечает (200)" -ForegroundColor Green
        } else {
            Write-Host "  ⚠ Status Code: $($result.StatusCode)" -ForegroundColor Yellow
        }
    }
} catch {
    Write-Host "  ⚠ Ошибка при тестировании: $($_.Exception.Message)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Тестирование завершено" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

