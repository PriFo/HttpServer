# Скрипт для запуска бенчмарка моделей и сбора статистики

$benchmarkUrl = "http://localhost:9999/api/models/benchmark"

Write-Host "Запуск бенчмарка моделей..." -ForegroundColor Green

# Параметры бенчмарка
$body = @{
    max_retries = 3
    retry_delay_ms = 200
    auto_update_priorities = $true
} | ConvertTo-Json

try {
    # Запускаем бенчмарк
    Write-Host "Отправка запроса на запуск бенчмарка..." -ForegroundColor Yellow
    $response = Invoke-RestMethod -Uri $benchmarkUrl -Method POST -Body $body -ContentType "application/json" -TimeoutSec 600
    
    Write-Host "`n=== РЕЗУЛЬТАТЫ БЕНЧМАРКА ===" -ForegroundColor Cyan
    Write-Host "Тестов выполнено: $($response.test_count)" -ForegroundColor White
    Write-Host "Всего моделей: $($response.total)" -ForegroundColor White
    Write-Host "Время выполнения: $($response.timestamp)" -ForegroundColor White
    
    # Сохраняем полные результаты в файл
    $response | ConvertTo-Json -Depth 10 | Out-File -FilePath "benchmark_results.json" -Encoding UTF8
    Write-Host "`nПолные результаты сохранены в benchmark_results.json" -ForegroundColor Green
    
    # Собираем статистику
    Write-Host "`n=== СТАТИСТИКА ПО МОДЕЛЯМ ===" -ForegroundColor Cyan
    
    $models = $response.models
    
    # Топ-10 по скорости
    Write-Host "`nТОП-10 моделей по скорости (speed):" -ForegroundColor Yellow
    $models | Sort-Object -Property speed -Descending | Select-Object -First 10 | ForEach-Object {
        Write-Host "  $($_.model): $([math]::Round($_.speed, 4)) (priority: $($_.priority), success_rate: $([math]::Round($_.success_rate, 1))%)" -ForegroundColor White
    }
    
    # Топ-10 по успешности
    Write-Host "`nТОП-10 моделей по успешности (success_rate):" -ForegroundColor Yellow
    $models | Sort-Object -Property success_rate -Descending | Select-Object -First 10 | ForEach-Object {
        Write-Host "  $($_.model): $([math]::Round($_.success_rate, 1))% (speed: $([math]::Round($_.speed, 4)), avg_time: $($_.avg_response_time_ms)ms)" -ForegroundColor White
    }
    
    # Топ-10 по скорости ответа
    Write-Host "`nТОП-10 моделей по скорости ответа (avg_response_time_ms):" -ForegroundColor Yellow
    $models | Where-Object { $_.avg_response_time_ms -gt 0 } | Sort-Object -Property avg_response_time_ms | Select-Object -First 10 | ForEach-Object {
        Write-Host "  $($_.model): $($_.avg_response_time_ms)ms (success_rate: $([math]::Round($_.success_rate, 1))%, speed: $([math]::Round($_.speed, 4)))" -ForegroundColor White
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
    Write-Host "  Успешных: $totalSuccess" -ForegroundColor Green
    Write-Host "  Ошибок: $totalErrors" -ForegroundColor Red
    Write-Host "  Средняя скорость: $([math]::Round($avgSpeed, 4))" -ForegroundColor White
    Write-Host "  Среднее время ответа: $([math]::Round($avgResponseTime, 0))ms" -ForegroundColor White
    
    Write-Host "`nБенчмарк завершен успешно!" -ForegroundColor Green
    
} catch {
    Write-Host "Ошибка при выполнении бенчмарка: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "Ответ сервера: $responseBody" -ForegroundColor Red
    }
    exit 1
}

