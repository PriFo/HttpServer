# Скрипт для автоматического резервного копирования баз данных (PowerShell версия для Windows)
# Можно запускать через Task Scheduler для регулярных бэкапов

param(
    [string]$BaseUrl = "http://localhost:9999",
    [string]$BackupDir = ".\data\backups",
    [int]$RetentionDays = 30,
    [string]$LogFile = ".\logs\backup.log"
)

$ErrorActionPreference = "Stop"

# Создаем директории
if (-not (Test-Path $BackupDir)) {
    New-Item -ItemType Directory -Path $BackupDir -Force | Out-Null
}

$logDir = Split-Path -Parent $LogFile
if (-not (Test-Path $logDir)) {
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null
}

# Функция для логирования
function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] $Message"
    Write-Host $logMessage
    Add-Content -Path $LogFile -Value $logMessage
}

# Функция для очистки старых бэкапов
function Remove-OldBackups {
    Write-Log "Очистка старых резервных копий (старше $RetentionDays дней)..."
    
    $cutoffDate = (Get-Date).AddDays(-$RetentionDays)
    
    # Удаляем старые ZIP файлы
    Get-ChildItem -Path $BackupDir -Filter "backup_*.zip" | 
        Where-Object { $_.LastWriteTime -lt $cutoffDate } | 
        Remove-Item -Force
    
    # Удаляем старые директории
    Get-ChildItem -Path $BackupDir -Directory -Filter "backup_*" | 
        Where-Object { $_.LastWriteTime -lt $cutoffDate } | 
        Remove-Item -Recurse -Force
    
    Write-Log "Очистка завершена"
}

# Функция для создания резервной копии через API
function New-Backup {
    param(
        [bool]$IncludeMain = $true,
        [bool]$IncludeUploads = $true,
        [bool]$IncludeService = $false,
        [string]$Format = "both"
    )
    
    Write-Log "Создание резервной копии..."
    Write-Log "  Include main: $IncludeMain"
    Write-Log "  Include uploads: $IncludeUploads"
    Write-Log "  Include service: $IncludeService"
    Write-Log "  Format: $Format"
    
    # Формируем JSON запрос
    $body = @{
        include_main = $IncludeMain
        include_uploads = $IncludeUploads
        include_service = $IncludeService
        format = $Format
    } | ConvertTo-Json
    
    try {
        $response = Invoke-RestMethod -Uri "$BaseUrl/api/databases/backup" `
            -Method Post `
            -ContentType "application/json" `
            -Body $body `
            -ErrorAction Stop
        
        if ($response.success) {
            Write-Log "Резервная копия успешно создана"
            Write-Log "Response: $($response | ConvertTo-Json -Compress)"
            return $true
        } else {
            Write-Log "ОШИБКА: Резервная копия не создана"
            Write-Log "Response: $($response | ConvertTo-Json -Compress)"
            return $false
        }
    } catch {
        Write-Log "ОШИБКА: Не удалось создать резервную копию"
        Write-Log "Error: $($_.Exception.Message)"
        return $false
    }
}

# Функция для проверки целостности резервной копии
function Test-BackupIntegrity {
    param([string]$BackupFile)
    
    if (-not (Test-Path $BackupFile)) {
        Write-Log "ОШИБКА: Файл резервной копии не найден: $BackupFile"
        return $false
    }
    
    # Проверяем, что файл не пустой
    $fileInfo = Get-Item $BackupFile
    if ($fileInfo.Length -eq 0) {
        Write-Log "ОШИБКА: Файл резервной копии пуст: $BackupFile"
        return $false
    }
    
    # Проверяем, что это валидный ZIP (если это ZIP)
    if ($BackupFile -match '\.zip$') {
        try {
            Add-Type -AssemblyName System.IO.Compression.FileSystem
            $zip = [System.IO.Compression.ZipFile]::OpenRead($BackupFile)
            $zip.Dispose()
        } catch {
            Write-Log "ОШИБКА: ZIP архив поврежден: $BackupFile"
            return $false
        }
    }
    
    Write-Log "Резервная копия проверена: $BackupFile (размер: $($fileInfo.Length) байт)"
    return $true
}

# Основная функция
function Main {
    Write-Log "=== Начало резервного копирования ==="
    
    # Проверяем доступность сервера
    try {
        $healthResponse = Invoke-WebRequest -Uri "$BaseUrl/health" -UseBasicParsing -TimeoutSec 5
        if ($healthResponse.StatusCode -ne 200) {
            Write-Log "ОШИБКА: Сервер недоступен на $BaseUrl"
            exit 1
        }
    } catch {
        Write-Log "ОШИБКА: Сервер недоступен на $BaseUrl"
        Write-Log "Error: $($_.Exception.Message)"
        exit 1
    }
    
    # Создаем резервную копию
    if (-not (New-Backup -IncludeMain $true -IncludeUploads $true -IncludeService $false -Format "both")) {
        Write-Log "ОШИБКА: Не удалось создать резервную копию"
        exit 1
    }
    
    # Находим последний созданный бэкап
    $latestBackup = Get-ChildItem -Path $BackupDir -Filter "backup_*.zip" | 
        Sort-Object LastWriteTime -Descending | 
        Select-Object -First 1
    
    if ($latestBackup) {
        Test-BackupIntegrity -BackupFile $latestBackup.FullName
    }
    
    # Очищаем старые бэкапы
    Remove-OldBackups
    
    Write-Log "=== Резервное копирование завершено ==="
}

# Запуск
Main

