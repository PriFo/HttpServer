# Скрипт для тестирования нормализации контрагентов на реальных базах данных
# Использование: .\scripts\test_counterparty_normalization.ps1

param(
    [switch]$Verbose,
    [string]$DatabasePath = ""
)

Write-Host "=== Тестирование нормализации контрагентов на реальных базах данных ===" -ForegroundColor Green

# Находим все базы данных с контрагентами
$dbPaths = @()

if ($DatabasePath -ne "") {
    if (Test-Path $DatabasePath) {
        $dbPaths += (Resolve-Path $DatabasePath).Path
    } else {
        Write-Host "Указанный путь не найден: $DatabasePath" -ForegroundColor Red
        exit 1
    }
} else {
    # Директории для поиска
    $searchDirs = @(".", "data", "data\uploads")

    foreach ($dir in $searchDirs) {
        if (Test-Path $dir) {
            $files = Get-ChildItem -Path $dir -Filter "*.db" -Recurse -ErrorAction SilentlyContinue
            foreach ($file in $files) {
                $baseName = $file.Name
                if ($baseName -ne "service.db" -and $baseName -ne "test.db" -and $baseName -ne "normalized_data.db") {
                    $dbPaths += $file.FullName
                }
            }
        }
    }
}

Write-Host "Найдено баз данных: $($dbPaths.Count)" -ForegroundColor Yellow

if ($dbPaths.Count -eq 0) {
    Write-Host "Базы данных не найдены!" -ForegroundColor Red
    exit 1
}

# Выводим список найденных баз
Write-Host "`nСписок баз данных для тестирования:" -ForegroundColor Cyan
foreach ($dbPath in $dbPaths) {
    $fileInfo = Get-Item $dbPath
    Write-Host "  - $dbPath ($([math]::Round($fileInfo.Length / 1MB, 2)) MB)" -ForegroundColor Gray
}

# Проверяем наличие контрагентов в каждой базе
Write-Host "`nПроверка наличия контрагентов..." -ForegroundColor Green
$databasesWithCounterparties = @()

foreach ($dbPath in $dbPaths) {
    try {
        # Используем sqlite3 для проверки
        $result = & sqlite3 $dbPath "SELECT COUNT(*) FROM catalog_items LIMIT 1;" 2>&1
        if ($LASTEXITCODE -eq 0 -and $result -match '^\d+$' -and [int]$result -gt 0) {
            $databasesWithCounterparties += $dbPath
            if ($Verbose) {
                Write-Host "  ✓ $dbPath - найдено контрагентов: $result" -ForegroundColor Green
            }
        } else {
            # Пробуем другие таблицы
            $result2 = & sqlite3 $dbPath "SELECT COUNT(*) FROM counterparties LIMIT 1;" 2>&1
            if ($LASTEXITCODE -eq 0 -and $result2 -match '^\d+$' -and [int]$result2 -gt 0) {
                $databasesWithCounterparties += $dbPath
                if ($Verbose) {
                    Write-Host "  ✓ $dbPath - найдено контрагентов: $result2" -ForegroundColor Green
                }
            } else {
                if ($Verbose) {
                    Write-Host "  ✗ $dbPath - контрагенты не найдены" -ForegroundColor Yellow
                }
            }
        }
    } catch {
        if ($Verbose) {
            Write-Host "  ✗ $dbPath - ошибка проверки: $_" -ForegroundColor Red
        }
    }
}

Write-Host "`nБаз данных с контрагентами: $($databasesWithCounterparties.Count)" -ForegroundColor Cyan

if ($databasesWithCounterparties.Count -eq 0) {
    Write-Host "Базы данных с контрагентами не найдены!" -ForegroundColor Red
    exit 1
}

# Создаем временный тестовый файл для запуска тестов
$testFile = "normalization/counterparty_normalization_integration_test.go"
if (-not (Test-Path $testFile)) {
    Write-Host "Тестовый файл не найден: $testFile" -ForegroundColor Red
    exit 1
}

# Запускаем тесты
Write-Host "`nЗапуск интеграционных тестов..." -ForegroundColor Green
Write-Host "Примечание: Тесты будут использовать моковый AI нормализатор" -ForegroundColor Yellow

# Устанавливаем переменную окружения для указания баз данных
$env:TEST_DATABASE_PATHS = ($databasesWithCounterparties -join ";")

# Запускаем тест с таймаутом
$testResult = go test ./normalization -v -run "^TestCounterpartyNormalization_AllDatabases$" -timeout 10m -short=false 2>&1

# Выводим результаты
Write-Host "`n=== Результаты тестирования ===" -ForegroundColor Cyan
$testResult | ForEach-Object {
    if ($_ -match "PASS|FAIL|RUN") {
        Write-Host $_ -ForegroundColor $(if ($_ -match "PASS") { "Green" } elseif ($_ -match "FAIL") { "Red" } else { "Yellow" })
    } elseif ($Verbose) {
        Write-Host $_ -ForegroundColor Gray
    }
}

# Проверяем результат
if ($LASTEXITCODE -eq 0) {
    Write-Host "`n=== Все тесты пройдены успешно! ===" -ForegroundColor Green
    exit 0
} else {
    Write-Host "`n=== Некоторые тесты провалились ===" -ForegroundColor Red
    exit 1
}

