# Скрипт для мониторинга прогресса нормализации по всем проектам

$baseUrl = "http://localhost:8080"
$timeout = 7
$checkInterval = 10 # секунд

Write-Host "📊 МОНИТОРИНГ ПРОГРЕССА НОРМАЛИЗАЦИИ" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

# Функция для выполнения HTTP запроса
function Invoke-ApiRequest {
    param(
        [string]$Method,
        [string]$Url,
        [int]$Timeout = 7
    )
    
    try {
        $response = Invoke-WebRequest -Method $Method -Uri $Url -TimeoutSec $Timeout -ErrorAction Stop
        return @{
            Success = $true
            StatusCode = $response.StatusCode
            Content = $response.Content | ConvertFrom-Json
        }
    }
    catch {
        return @{
            Success = $false
            StatusCode = $_.Exception.Response.StatusCode.value__
            Error = $_.Exception.Message
            Content = $null
        }
    }
}

# Получаем всех клиентов
$clientsResponse = Invoke-ApiRequest -Method "GET" -Url "$baseUrl/api/clients" -Timeout $timeout

if (-not $clientsResponse.Success) {
    Write-Host "❌ Ошибка получения клиентов: $($clientsResponse.Error)" -ForegroundColor Red
    exit 1
}

$clients = $clientsResponse.Content
$allProjects = @()

# Собираем все проекты
foreach ($client in $clients) {
    $clientId = $client.id
    $projectsResponse = Invoke-ApiRequest -Method "GET" -Url "$baseUrl/api/clients/$clientId/projects" -Timeout $timeout
    
    if ($projectsResponse.Success) {
        $projects = $projectsResponse.Content
        foreach ($project in $projects) {
            $allProjects += @{
                ClientID = $clientId
                ClientName = $client.name
                ProjectID = $project.id
                ProjectName = $project.name
                ProjectType = $project.project_type
            }
        }
    }
}

Write-Host "Найдено проектов для мониторинга: $($allProjects.Count)" -ForegroundColor Green
Write-Host "Интервал проверки: $checkInterval секунд" -ForegroundColor Yellow
Write-Host "Нажмите Ctrl+C для остановки" -ForegroundColor Yellow
Write-Host ""

$iteration = 0

while ($true) {
    $iteration++
    $timestamp = Get-Date -Format "HH:mm:ss"
    
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
    Write-Host "[$timestamp] Проверка #$iteration" -ForegroundColor Cyan
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
    
    $runningCount = 0
    $completedCount = 0
    $failedCount = 0
    $idleCount = 0
    
    foreach ($project in $allProjects) {
        $statusResponse = Invoke-ApiRequest -Method "GET" `
            -Url "$baseUrl/api/clients/$($project.ClientID)/projects/$($project.ProjectID)/normalization/status" `
            -Timeout $timeout
        
        if ($statusResponse.Success) {
            $status = $statusResponse.Content
            $statusText = $status.status
            
            switch ($statusText) {
                "running" {
                    $runningCount++
                    $processed = if ($status.processed) { $status.processed } else { 0 }
                    $total = if ($status.total) { $status.total } else { 0 }
                    $percent = if ($total -gt 0) { [math]::Round(($processed / $total) * 100, 1) } else { 0 }
                    Write-Host "  🟢 $($project.ClientName) / $($project.ProjectName) ($($project.ProjectType)): $processed/$total ($percent%)" -ForegroundColor Green
                }
                "completed" {
                    $completedCount++
                    Write-Host "  ✅ $($project.ClientName) / $($project.ProjectName): Завершено" -ForegroundColor Cyan
                }
                "failed" {
                    $failedCount++
                    Write-Host "  ❌ $($project.ClientName) / $($project.ProjectName): Ошибка" -ForegroundColor Red
                }
                default {
                    $idleCount++
                    Write-Host "  ⚪ $($project.ClientName) / $($project.ProjectName): Не запущено" -ForegroundColor Gray
                }
            }
        }
        else {
            Write-Host "  ⚠️  $($project.ClientName) / $($project.ProjectName): Ошибка получения статуса" -ForegroundColor Yellow
        }
    }
    
    Write-Host ""
    Write-Host "Статистика:" -ForegroundColor White
    Write-Host "  🟢 Запущено: $runningCount" -ForegroundColor Green
    Write-Host "  ✅ Завершено: $completedCount" -ForegroundColor Cyan
    Write-Host "  ❌ Ошибки: $failedCount" -ForegroundColor Red
    Write-Host "  ⚪ Не запущено: $idleCount" -ForegroundColor Gray
    Write-Host ""
    
    Start-Sleep -Seconds $checkInterval
}


