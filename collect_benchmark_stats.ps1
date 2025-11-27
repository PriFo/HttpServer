# Скрипт для сбора статистики из результатов бенчмарка

$benchmarkUrl = "http://localhost:9999/api/models/benchmark"

Write-Host "Получение результатов бенчмарка..." -ForegroundColor Green

try {
    # Получаем последние результаты
    $response = Invoke-RestMethod -Uri $benchmarkUrl -Method GET -ContentType "application/json"
    
    if ($response.message) {
        Write-Host "Сообщение: $($response.message)" -ForegroundColor Yellow
    }
    
    if (-not $response.models -or $response.models.Count -eq 0) {
        Write-Host "Нет данных о моделях. Запустите бенчмарк через POST запрос." -ForegroundColor Red
        exit 1
    }
    
    Write-Host "`n=== РЕЗУЛЬТАТЫ БЕНЧМАРКА ===" -ForegroundColor Cyan
    Write-Host "Тестов выполнено: $($response.test_count)" -ForegroundColor White
    Write-Host "Всего моделей: $($response.total)" -ForegroundColor White
    if ($response.timestamp) {
        Write-Host "Время выполнения: $($response.timestamp)" -ForegroundColor White
    }
    
    # Сохраняем полные результаты в файл
    $response | ConvertTo-Json -Depth 10 | Out-File -FilePath "benchmark_results.json" -Encoding UTF8
    Write-Host "`nПолные результаты сохранены в benchmark_results.json" -ForegroundColor Green
    
    $models = $response.models
    
    # Топ-10 по скорости
    Write-Host "`n=== ТОП-10 МОДЕЛЕЙ ПО СКОРОСТИ (speed) ===" -ForegroundColor Yellow
    $topSpeed = $models | Sort-Object -Property speed -Descending | Select-Object -First 10
    $i = 1
    foreach ($model in $topSpeed) {
        Write-Host "$i. $($model.model)" -ForegroundColor Cyan
        Write-Host "   Speed: $([math]::Round($model.speed, 4)) | Priority: $($model.priority) | Success: $([math]::Round($model.success_rate, 1))% | Avg Time: $($model.avg_response_time_ms)ms" -ForegroundColor White
        $i++
    }
    
    # Топ-10 по успешности
    Write-Host "`n=== ТОП-10 МОДЕЛЕЙ ПО УСПЕШНОСТИ (success_rate) ===" -ForegroundColor Yellow
    $topSuccess = $models | Sort-Object -Property success_rate -Descending | Select-Object -First 10
    $i = 1
    foreach ($model in $topSuccess) {
        Write-Host "$i. $($model.model)" -ForegroundColor Cyan
        Write-Host "   Success Rate: $([math]::Round($model.success_rate, 1))% | Speed: $([math]::Round($model.speed, 4)) | Avg Time: $($model.avg_response_time_ms)ms" -ForegroundColor White
        $i++
    }
    
    # Топ-10 по скорости ответа
    Write-Host "`n=== ТОП-10 МОДЕЛЕЙ ПО СКОРОСТИ ОТВЕТА (avg_response_time_ms) ===" -ForegroundColor Yellow
    $topFast = $models | Where-Object { $_.avg_response_time_ms -gt 0 } | Sort-Object -Property avg_response_time_ms | Select-Object -First 10
    $i = 1
    foreach ($model in $topFast) {
        Write-Host "$i. $($model.model)" -ForegroundColor Cyan
        Write-Host "   Avg Time: $($model.avg_response_time_ms)ms | Success: $([math]::Round($model.success_rate, 1))% | Speed: $([math]::Round($model.speed, 4))" -ForegroundColor White
        $i++
    }
    
    # Статистика по статусам
    Write-Host "`n=== СТАТИСТИКА ПО СТАТУСАМ ===" -ForegroundColor Cyan
    $statusGroups = $models | Group-Object -Property status
    foreach ($group in $statusGroups) {
        Write-Host "  $($group.Name): $($group.Count)" -ForegroundColor White
    }
    
    # Общая статистика
    Write-Host "`n=== ОБЩАЯ СТАТИСТИКА ===" -ForegroundColor Cyan
    $totalRequests = ($models | Measure-Object -Property total_requests -Sum).Sum
    $totalSuccess = ($models | Measure-Object -Property success_count -Sum).Sum
    $totalErrors = ($models | Measure-Object -Property error_count -Sum).Sum
    $avgSpeed = ($models | Measure-Object -Property speed -Average).Average
    $avgResponseTime = ($models | Where-Object { $_.avg_response_time_ms -gt 0 } | Measure-Object -Property avg_response_time_ms -Average).Average
    
    Write-Host "  Всего запросов: $totalRequests" -ForegroundColor White
    Write-Host "  Успешных: $totalSuccess ($([math]::Round(($totalSuccess/$totalRequests)*100, 1))%)" -ForegroundColor Green
    Write-Host "  Ошибок: $totalErrors ($([math]::Round(($totalErrors/$totalRequests)*100, 1))%)" -ForegroundColor Red
    Write-Host "  Средняя скорость: $([math]::Round($avgSpeed, 4))" -ForegroundColor White
    Write-Host "  Среднее время ответа: $([math]::Round($avgResponseTime, 0))ms" -ForegroundColor White
    
    # Статистика по приоритетам
    Write-Host "`n=== СТАТИСТИКА ПО ПРИОРИТЕТАМ ===" -ForegroundColor Cyan
    $priorityGroups = $models | Group-Object -Property priority | Sort-Object -Property Name
    Write-Host "  Диапазон приоритетов: $($priorityGroups[0].Name) - $($priorityGroups[-1].Name)" -ForegroundColor White
    Write-Host "  Моделей с приоритетом 1-10: $(($models | Where-Object { $_.priority -ge 1 -and $_.priority -le 10 }).Count)" -ForegroundColor White
    Write-Host "  Моделей с приоритетом 11-50: $(($models | Where-Object { $_.priority -ge 11 -and $_.priority -le 50 }).Count)" -ForegroundColor White
    Write-Host "  Моделей с приоритетом 51-100: $(($models | Where-Object { $_.priority -ge 51 -and $_.priority -le 100 }).Count)" -ForegroundColor White
    Write-Host "  Моделей с приоритетом >100: $(($models | Where-Object { $_.priority -gt 100 }).Count)" -ForegroundColor White
    
    Write-Host "`nСтатистика собрана успешно!" -ForegroundColor Green
    
} catch {
    Write-Host "Ошибка при получении результатов: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        try {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            $responseBody = $reader.ReadToEnd()
            Write-Host "Ответ сервера: $responseBody" -ForegroundColor Red
        } catch {
            Write-Host "Не удалось прочитать ответ сервера" -ForegroundColor Red
        }
    }
    exit 1
}

