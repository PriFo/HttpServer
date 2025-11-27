# Скрипт для проверки данных в базе после тестов
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Проверка данных в базе после тестов" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$baseUrl = "http://localhost:9999"

# Функция для получения списка баз данных проекта
function Get-ProjectDatabases {
    param(
        [int]$ClientID,
        [int]$ProjectID
    )
    
    try {
        $url = "$baseUrl/api/clients/$ClientID/projects/$ProjectID/databases"
        $response = Invoke-RestMethod -Uri $url -Method GET -ErrorAction Stop
        return $response
    } catch {
        Write-Host "  ERROR Failed to get databases: $($_.Exception.Message)" -ForegroundColor Red
        return $null
    }
}

# Функция для получения информации о базе данных
function Get-DatabaseInfo {
    param(
        [int]$ClientID,
        [int]$ProjectID,
        [int]$DatabaseID
    )
    
    try {
        $url = "$baseUrl/api/clients/$ClientID/projects/$ProjectID/databases/$DatabaseID"
        $response = Invoke-RestMethod -Uri $url -Method GET -ErrorAction Stop
        return $response
    } catch {
        Write-Host "  ERROR Failed to get database info: $($_.Exception.Message)" -ForegroundColor Red
        return $null
    }
}

# Функция для проверки существования файла
function Test-DatabaseFile {
    param(
        [string]$FilePath
    )
    
    if (Test-Path $FilePath) {
        $fileInfo = Get-Item $FilePath
        return @{
            Exists = $true
            Size = $fileInfo.Length
            LastModified = $fileInfo.LastWriteTime
        }
    } else {
        return @{
            Exists = $false
            Size = 0
            LastModified = $null
        }
    }
}

Write-Host "[Проверка] Тестирование подключения к серверу..." -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "$baseUrl/api/health" -Method GET -ErrorAction Stop
    Write-Host "  OK Сервер доступен" -ForegroundColor Green
} catch {
    Write-Host "  ERROR Сервер недоступен: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "  Запустите сервер перед проверкой" -ForegroundColor Yellow
    exit 1
}

Write-Host ""
Write-Host "[Проверка] Для проверки данных в БД необходимо:" -ForegroundColor Yellow
Write-Host "  1. Запустить интеграционные тесты (test_multiple_database_upload.ps1)" -ForegroundColor Gray
Write-Host "  2. Указать ClientID и ProjectID из результатов тестов" -ForegroundColor Gray
Write-Host ""

# Если переданы параметры, проверяем конкретный проект
param(
    [int]$ClientID = 0,
    [int]$ProjectID = 0
)

if ($ClientID -gt 0 -and $ProjectID -gt 0) {
    Write-Host "[Проверка] Проверка данных для ClientID=$ClientID, ProjectID=$ProjectID..." -ForegroundColor Yellow
    
    $databases = Get-ProjectDatabases -ClientID $ClientID -ProjectID $ProjectID
    
    if ($databases -and $databases.databases) {
        Write-Host "  OK Найдено баз данных: $($databases.databases.Count)" -ForegroundColor Green
        Write-Host ""
        
        foreach ($db in $databases.databases) {
            Write-Host "  База данных ID: $($db.id)" -ForegroundColor Cyan
            Write-Host "    Имя: $($db.name)" -ForegroundColor Gray
            Write-Host "    Путь: $($db.path)" -ForegroundColor Gray
            Write-Host "    Размер: $($db.size) байт" -ForegroundColor Gray
            Write-Host "    Создана: $($db.created_at)" -ForegroundColor Gray
            Write-Host "    Статус: $($db.status)" -ForegroundColor Gray
            
            # Проверяем существование файла
            if ($db.path) {
                $fileCheck = Test-DatabaseFile -FilePath $db.path
                if ($fileCheck.Exists) {
                    Write-Host "    Файл существует: ДА" -ForegroundColor Green
                    Write-Host "    Размер файла: $($fileCheck.Size) байт" -ForegroundColor Gray
                    Write-Host "    Изменен: $($fileCheck.LastModified)" -ForegroundColor Gray
                } else {
                    Write-Host "    Файл существует: НЕТ" -ForegroundColor Red
                }
            }
            Write-Host ""
        }
    } else {
        Write-Host "  WARNING Базы данных не найдены" -ForegroundColor Yellow
    }
} else {
    Write-Host "[Информация] Для проверки конкретного проекта запустите:" -ForegroundColor Yellow
    Write-Host "  .\verify_database_after_tests.ps1 -ClientID <id> -ProjectID <id>" -ForegroundColor Gray
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Проверка завершена" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

