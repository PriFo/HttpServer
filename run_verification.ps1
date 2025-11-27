# Упрощенный скрипт проверки API
$backendUrl = "http://127.0.0.1:9999"
$timeout = 5

Write-Host "Проверка доступности backend..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$backendUrl/health" -TimeoutSec $timeout -UseBasicParsing -ErrorAction Stop
    Write-Host "[OK] Backend доступен" -ForegroundColor Green
    $backendReady = $true
} catch {
    Write-Host "[ERROR] Backend недоступен: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Запустите backend: go run main_no_gui.go" -ForegroundColor Yellow
    $backendReady = $false
    exit 1
}

if (-not $backendReady) { exit 1 }

Write-Host "`nПолучение списка клиентов..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$backendUrl/api/clients" -TimeoutSec $timeout -UseBasicParsing
    $clients = $response.Content | ConvertFrom-Json
    
    if ($clients -and $clients.Count -gt 0) {
        $clientId = $clients[0].id
        Write-Host "[OK] Найден клиент с ID: $clientId" -ForegroundColor Green
        
        Write-Host "`nПроверка API /api/counterparties/normalized?client_id=$clientId..." -ForegroundColor Yellow
        $uri = [System.UriBuilder]::new("$backendUrl/api/counterparties/normalized")
        $uri.Query = "client_id=$clientId"
        
        $response = Invoke-WebRequest -Uri $uri.ToString() -TimeoutSec $timeout -UseBasicParsing
        $data = $response.Content | ConvertFrom-Json
        
        Write-Host "[OK] Запрос выполнен успешно" -ForegroundColor Green
        Write-Host "  - Контрагентов: $($data.counterparties.Count)" -ForegroundColor Gray
        Write-Host "  - Всего: $($data.total)" -ForegroundColor Gray
        Write-Host "  - Проектов: $($data.projects.Count)" -ForegroundColor Gray
        
        if ($data.counterparties.Count -gt 0) {
            Write-Host "`n[OK] Контрагенты найдены!" -ForegroundColor Green
            $first = $data.counterparties[0]
            Write-Host "  Пример: $($first.name) (ID: $($first.id))" -ForegroundColor Gray
        }
    } else {
        Write-Host "[WARNING] Клиенты не найдены" -ForegroundColor Yellow
    }
} catch {
    Write-Host "[ERROR] Ошибка: $($_.Exception.Message)" -ForegroundColor Red
}

