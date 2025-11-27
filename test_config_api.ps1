# Тестирование API конфигурации
# Сервер должен быть запущен на http://localhost:9999

$BASE_URL = "http://localhost:9999"
$REPORT_FILE = "config_api_test_report.md"

# Очищаем файл отчета
"# Отчет о тестировании API конфигурации" | Out-File -FilePath $REPORT_FILE -Encoding UTF8
"" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
"Дата: $(Get-Date)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
"" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8

# Функция для выполнения запроса и сохранения результата
function Test-Request {
    param(
        [string]$TestName,
        [string]$Method,
        [string]$Url,
        [string]$Data = $null,
        [int]$ExpectedStatus
    )
    
    "## Тест: $TestName" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    "" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    "**Запрос:**" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    "- Метод: $Method" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    "- URL: $Url" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    if ($Data) {
        "- Body: ``$Data``" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    }
    "" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    
    try {
        if ($Method -eq "GET") {
            $response = Invoke-WebRequest -Uri $Url -Method GET -UseBasicParsing -ErrorAction Stop
        } elseif ($Method -eq "PUT") {
            $response = Invoke-WebRequest -Uri $Url -Method PUT -Body $Data -ContentType "application/json" -UseBasicParsing -ErrorAction Stop
        }
        
        $httpCode = $response.StatusCode
        $body = $response.Content
        
        "**Ответ:**" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        "- Статус: $httpCode" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        "- Body:" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        "``````json" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        
        # Пытаемся форматировать JSON
        try {
            $jsonBody = $body | ConvertFrom-Json | ConvertTo-Json -Depth 10
            $jsonBody | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        } catch {
            $body | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        }
        
        "``````" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        "" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        
        if ($httpCode -eq $ExpectedStatus) {
            "**Результат:** ✅ PASS" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        } else {
            "**Результат:** ❌ FAIL (ожидался статус $ExpectedStatus, получен $httpCode)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        }
    } catch {
        $httpCode = $_.Exception.Response.StatusCode.value__
        $body = $_.Exception.Response | ConvertTo-Json -Depth 5
        "**Ответ:**" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        "- Статус: $httpCode" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        "- Ошибка: $($_.Exception.Message)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        "" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        
        if ($httpCode -eq $ExpectedStatus) {
            "**Результат:** ✅ PASS" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        } else {
            "**Результат:** ❌ FAIL (ожидался статус $ExpectedStatus, получен $httpCode)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        }
    }
    
    "" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    "---" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    "" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
}

# Тест 1: GET /api/config (безопасная версия)
Write-Host "Выполняю тест 1: GET /api/config..."
Test-Request -TestName "GET /api/config (безопасная версия)" `
    -Method "GET" `
    -Url "$BASE_URL/api/config" `
    -ExpectedStatus 200

# Проверка отсутствия секретных полей
try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/config" -Method GET
    if ($response.arliai_api_key) {
        "⚠️  ВНИМАНИЕ: В безопасной версии обнаружено поле arliai_api_key" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    } else {
        "✅ Поле arliai_api_key отсутствует в безопасной версии" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    }
    
    if ($response.has_arliai_api_key -ne $null) {
        "✅ Поле has_arliai_api_key присутствует: $($response.has_arliai_api_key)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    } else {
        "⚠️  Поле has_arliai_api_key отсутствует" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    }
    
    if ($response.log_level) {
        "✅ Поле log_level присутствует: $($response.log_level)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    } else {
        "❌ Поле log_level отсутствует" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    }
} catch {
    "❌ Ошибка при проверке безопасной версии: $($_.Exception.Message)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
}

"" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8

# Тест 2: GET /api/config/full (полная версия)
Write-Host "Выполняю тест 2: GET /api/config/full..."
Test-Request -TestName "GET /api/config/full (полная версия)" `
    -Method "GET" `
    -Url "$BASE_URL/api/config/full" `
    -ExpectedStatus 200

# Сохраняем текущую конфигурацию
try {
    $currentConfig = Invoke-RestMethod -Uri "$BASE_URL/api/config/full" -Method GET
    $currentConfigJson = $currentConfig | ConvertTo-Json -Depth 10
    $currentConfigJson | Out-File -FilePath "current_config.json" -Encoding UTF8
} catch {
    Write-Host "Ошибка при получении текущей конфигурации: $($_.Exception.Message)"
}

# Тест 3: PUT /api/config (обновление log_level)
Write-Host "Выполняю тест 3: PUT /api/config..."
try {
    $currentConfig = Invoke-RestMethod -Uri "$BASE_URL/api/config/full" -Method GET
    $currentConfig.log_level = "DEBUG"
    $updatedConfigJson = $currentConfig | ConvertTo-Json -Depth 10
    
    Test-Request -TestName "PUT /api/config (обновление log_level)" `
        -Method "PUT" `
        -Url "$BASE_URL/api/config?reason=QA test update" `
        -Data $updatedConfigJson `
        -ExpectedStatus 200
    
    # Проверяем, что log_level изменился
    Start-Sleep -Seconds 1
    $newConfig = Invoke-RestMethod -Uri "$BASE_URL/api/config/full" -Method GET
    if ($newConfig.log_level -eq "DEBUG") {
        "✅ log_level успешно обновлен на DEBUG" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    } else {
        "❌ log_level не обновлен (текущее значение: $($newConfig.log_level))" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    }
} catch {
    "❌ Ошибка при обновлении конфигурации: $($_.Exception.Message)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
}

"" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8

# Тест 4: GET /api/config/history
Write-Host "Выполняю тест 4: GET /api/config/history..."
Test-Request -TestName "GET /api/config/history" `
    -Method "GET" `
    -Url "$BASE_URL/api/config/history" `
    -ExpectedStatus 200

# Проверка структуры истории
try {
    $historyResponse = Invoke-RestMethod -Uri "$BASE_URL/api/config/history" -Method GET
    if ($historyResponse.current_version) {
        "✅ current_version: $($historyResponse.current_version)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    } else {
        "❌ Поле current_version отсутствует" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    }
    
    if ($historyResponse.history) {
        $historyCount = $historyResponse.history.Count
        "✅ Количество записей в истории: $historyCount" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
        
        if ($historyCount -gt 0) {
            $lastEntry = $historyResponse.history[0]
            if ($lastEntry.version) {
                "✅ Версия последней записи: $($lastEntry.version)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
            }
            
            if ($lastEntry.changed_by) {
                "✅ changed_by: $($lastEntry.changed_by)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
            }
            
            if ($lastEntry.change_reason) {
                if ($lastEntry.change_reason -eq "QA test update") {
                    "✅ change_reason корректно сохранен: $($lastEntry.change_reason)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
                } else {
                    "⚠️  change_reason: $($lastEntry.change_reason) (ожидалось: QA test update)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
                }
            }
            
            if ($lastEntry.created_at) {
                "✅ created_at: $($lastEntry.created_at)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
            }
        }
    } else {
        "❌ Поле history отсутствует" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    }
} catch {
    "❌ Ошибка при проверке истории: $($_.Exception.Message)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
}

"" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8

# Тест 5: Невалидный JSON
Write-Host "Выполняю тест 5: PUT /api/config с невалидным JSON..."
Test-Request -TestName "PUT /api/config (невалидный JSON)" `
    -Method "PUT" `
    -Url "$BASE_URL/api/config" `
    -Data '{"log_level":}' `
    -ExpectedStatus 400

# Тест 6: Невалидное значение log_level
Write-Host "Выполняю тест 6: PUT /api/config с невалидным log_level..."
try {
    $currentConfig = Invoke-RestMethod -Uri "$BASE_URL/api/config/full" -Method GET
    $currentConfig.log_level = "INVALID"
    $invalidConfigJson = $currentConfig | ConvertTo-Json -Depth 10
    
    Test-Request -TestName "PUT /api/config (невалидный log_level)" `
        -Method "PUT" `
        -Url "$BASE_URL/api/config" `
        -Data $invalidConfigJson `
        -ExpectedStatus 400
} catch {
    # Ожидаем ошибку 400
    if ($_.Exception.Response.StatusCode.value__ -eq 400) {
        "✅ Получена ожидаемая ошибка 400" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    }
}

# Тест 7: Некорректный параметр limit
Write-Host "Выполняю тест 7: GET /api/config/history?limit=-1..."
Test-Request -TestName "GET /api/config/history?limit=-1" `
    -Method "GET" `
    -Url "$BASE_URL/api/config/history?limit=-1" `
    -ExpectedStatus 200

# Тест 8: Слишком большое значение limit
Write-Host "Выполняю тест 8: GET /api/config/history?limit=1000..."
Test-Request -TestName "GET /api/config/history?limit=1000" `
    -Method "GET" `
    -Url "$BASE_URL/api/config/history?limit=1000" `
    -ExpectedStatus 200

# Проверяем, что limit ограничен
try {
    $largeLimitResponse = Invoke-RestMethod -Uri "$BASE_URL/api/config/history?limit=1000" -Method GET
    $largeLimitCount = $largeLimitResponse.history.Count
    if ($largeLimitCount -le 100) {
        "✅ limit корректно ограничен максимумом (100): получено $largeLimitCount записей" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    } else {
        "⚠️  limit не ограничен (получено $largeLimitCount записей)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
    }
} catch {
    "❌ Ошибка при проверке limit: $($_.Exception.Message)" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
}

"" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
"## Резюме" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
"" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8
"Тестирование завершено. Результаты сохранены в $REPORT_FILE" | Out-File -FilePath $REPORT_FILE -Append -Encoding UTF8

Write-Host ""
Write-Host "Тестирование завершено. Отчет сохранен в $REPORT_FILE"

