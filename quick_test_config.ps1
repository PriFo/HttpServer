# Быстрый тест API конфигурации
# Убедитесь, что сервер запущен на http://localhost:9999

$BASE_URL = "http://localhost:9999"
$errors = 0

Write-Host "=== Тестирование API конфигурации ===" -ForegroundColor Cyan
Write-Host ""

# Тест 1: GET /api/config
Write-Host "1. Тест GET /api/config (безопасная версия)..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/config" -Method GET -ErrorAction Stop
    if ($response.log_level) {
        Write-Host "   ✅ PASS - log_level присутствует: $($response.log_level)" -ForegroundColor Green
    } else {
        Write-Host "   ❌ FAIL - log_level отсутствует" -ForegroundColor Red
        $errors++
    }
    if ($response.has_arliai_api_key -ne $null) {
        Write-Host "   ✅ PASS - has_arliai_api_key присутствует" -ForegroundColor Green
    } else {
        Write-Host "   ⚠️  WARN - has_arliai_api_key отсутствует" -ForegroundColor Yellow
    }
    if ($response.arliai_api_key) {
        Write-Host "   ❌ FAIL - arliai_api_key не должен быть в безопасной версии" -ForegroundColor Red
        $errors++
    } else {
        Write-Host "   ✅ PASS - arliai_api_key отсутствует (безопасно)" -ForegroundColor Green
    }
} catch {
    Write-Host "   ❌ FAIL - Ошибка: $($_.Exception.Message)" -ForegroundColor Red
    $errors++
}
Write-Host ""

# Тест 2: GET /api/config/full
Write-Host "2. Тест GET /api/config/full (полная версия)..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/config/full" -Method GET -ErrorAction Stop
    if ($response.log_level) {
        Write-Host "   ✅ PASS - log_level присутствует: $($response.log_level)" -ForegroundColor Green
        $currentLogLevel = $response.log_level
    } else {
        Write-Host "   ❌ FAIL - log_level отсутствует" -ForegroundColor Red
        $errors++
    }
    Write-Host "   ✅ PASS - Полная конфигурация получена" -ForegroundColor Green
    $currentConfig = $response
} catch {
    Write-Host "   ❌ FAIL - Ошибка: $($_.Exception.Message)" -ForegroundColor Red
    $errors++
    exit 1
}
Write-Host ""

# Тест 3: PUT /api/config
Write-Host "3. Тест PUT /api/config (обновление log_level)..." -ForegroundColor Yellow
try {
    $newLogLevel = if ($currentLogLevel -eq "INFO") { "DEBUG" } else { "INFO" }
    $currentConfig.log_level = $newLogLevel
    $jsonBody = $currentConfig | ConvertTo-Json -Depth 10
    
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/config?reason=QA%20test%20update" -Method PUT -Body $jsonBody -ContentType "application/json" -ErrorAction Stop
    if ($response.log_level -eq $newLogLevel) {
        Write-Host "   ✅ PASS - log_level успешно обновлен: $($response.log_level)" -ForegroundColor Green
    } else {
        Write-Host "   ❌ FAIL - log_level не обновлен (ожидалось: $newLogLevel, получено: $($response.log_level))" -ForegroundColor Red
        $errors++
    }
} catch {
    Write-Host "   ❌ FAIL - Ошибка: $($_.Exception.Message)" -ForegroundColor Red
    $errors++
}
Write-Host ""

# Тест 4: GET /api/config/history
Write-Host "4. Тест GET /api/config/history..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/config/history" -Method GET -ErrorAction Stop
    if ($response.current_version) {
        Write-Host "   ✅ PASS - current_version: $($response.current_version)" -ForegroundColor Green
    } else {
        Write-Host "   ❌ FAIL - current_version отсутствует" -ForegroundColor Red
        $errors++
    }
    if ($response.history -and $response.history.Count -gt 0) {
        $lastEntry = $response.history[0]
        Write-Host "   ✅ PASS - История содержит $($response.history.Count) записей" -ForegroundColor Green
        if ($lastEntry.version) {
            Write-Host "   ✅ PASS - Версия последней записи: $($lastEntry.version)" -ForegroundColor Green
        }
        if ($lastEntry.changed_by) {
            Write-Host "   ✅ PASS - changed_by: $($lastEntry.changed_by)" -ForegroundColor Green
        }
        if ($lastEntry.change_reason -eq "QA test update") {
            Write-Host "   ✅ PASS - change_reason корректно сохранен: $($lastEntry.change_reason)" -ForegroundColor Green
        } elseif ($lastEntry.change_reason) {
            Write-Host "   ⚠️  WARN - change_reason: $($lastEntry.change_reason) (ожидалось: QA test update)" -ForegroundColor Yellow
        }
        if ($lastEntry.created_at) {
            Write-Host "   ✅ PASS - created_at: $($lastEntry.created_at)" -ForegroundColor Green
        }
    } else {
        Write-Host "   ⚠️  WARN - История пуста или отсутствует" -ForegroundColor Yellow
    }
} catch {
    Write-Host "   ❌ FAIL - Ошибка: $($_.Exception.Message)" -ForegroundColor Red
    $errors++
}
Write-Host ""

# Тест 5: Невалидный log_level
Write-Host "5. Тест PUT /api/config (невалидный log_level)..." -ForegroundColor Yellow
try {
    $currentConfig.log_level = "INVALID"
    $jsonBody = $currentConfig | ConvertTo-Json -Depth 10
    
    try {
        $response = Invoke-RestMethod -Uri "$BASE_URL/api/config" -Method PUT -Body $jsonBody -ContentType "application/json" -ErrorAction Stop
        Write-Host "   ❌ FAIL - Должна была быть ошибка 400" -ForegroundColor Red
        $errors++
    } catch {
        if ($_.Exception.Response.StatusCode.value__ -eq 400) {
            Write-Host "   ✅ PASS - Получена ожидаемая ошибка 400" -ForegroundColor Green
        } else {
            Write-Host "   ⚠️  WARN - Получена ошибка $($_.Exception.Response.StatusCode.value__), ожидалась 400" -ForegroundColor Yellow
        }
    }
} catch {
    Write-Host "   ⚠️  WARN - Не удалось выполнить тест" -ForegroundColor Yellow
}
Write-Host ""

# Резюме
Write-Host "=== Резюме ===" -ForegroundColor Cyan
if ($errors -eq 0) {
    Write-Host "✅ Все тесты пройдены успешно!" -ForegroundColor Green
} else {
    Write-Host "❌ Обнаружено ошибок: $errors" -ForegroundColor Red
}
Write-Host ""
Write-Host "Примечание: Проверьте логи сервера на наличие записей с префиксом [Config]" -ForegroundColor Gray

