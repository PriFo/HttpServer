# Скрипт для тестирования нормализации контрагентов на всех базах данных
# Использование: .\scripts\test_all_databases.ps1

Write-Host "=== Тестирование нормализации контрагентов на всех базах данных ===" -ForegroundColor Green

# Находим все базы данных с контрагентами
$dbPaths = @()

# Директории для поиска
$searchDirs = @(".", "data", "data\uploads")

foreach ($dir in $searchDirs) {
    if (Test-Path $dir) {
        $files = Get-ChildItem -Path $dir -Filter "*.db" -Recurse -ErrorAction SilentlyContinue
        foreach ($file in $files) {
            $baseName = $file.Name
            if ($baseName -ne "service.db" -and $baseName -ne "test.db") {
                $dbPaths += $file.FullName
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
    Write-Host "  - $dbPath" -ForegroundColor Gray
}

# Запускаем тесты
Write-Host "`nЗапуск интеграционных тестов..." -ForegroundColor Green
$env:TEST_ALL_DATABASES = "true"

# Запускаем тест с коротким таймаутом для каждой базы
go test ./normalization -v -run "^TestCounterpartyNormalization_AllDatabases$" -timeout 5m -short=false

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n=== Все тесты пройдены успешно! ===" -ForegroundColor Green
} else {
    Write-Host "`n=== Некоторые тесты провалились ===" -ForegroundColor Red
    exit 1
}

