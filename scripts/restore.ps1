# Скрипт для восстановления из резервной копии (PowerShell версия для Windows)

param(
    [string]$BackupFile,
    [string]$BackupDir = ".\data\backups",
    [string]$RestoreDir = ".\data\restored",
    [string]$LogFile = ".\logs\restore.log"
)

$ErrorActionPreference = "Stop"

# Создаем директории
if (-not (Test-Path $RestoreDir)) {
    New-Item -ItemType Directory -Path $RestoreDir -Force | Out-Null
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

# Функция для выбора резервной копии
function Select-Backup {
    param([string]$BackupFile)
    
    if ($BackupFile) {
        if (-not (Test-Path $BackupFile)) {
            Write-Log "ОШИБКА: Файл не найден: $BackupFile"
            exit 1
        }
        return $BackupFile
    }
    
    # Ищем последний бэкап
    $backups = Get-ChildItem -Path $BackupDir -Filter "backup_*.zip" | 
        Sort-Object LastWriteTime -Descending
    
    if ($backups.Count -eq 0) {
        Write-Log "ОШИБКА: Резервные копии не найдены в $BackupDir"
        exit 1
    }
    
    return $backups[0].FullName
}

# Функция для восстановления из ZIP
function Restore-FromZip {
    param(
        [string]$BackupFile,
        [string]$RestorePath
    )
    
    Write-Log "Восстановление из: $BackupFile"
    Write-Log "Восстановление в: $RestorePath"
    
    # Проверяем целостность архива
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        $zip = [System.IO.Compression.ZipFile]::OpenRead($BackupFile)
        $zip.Dispose()
    } catch {
        Write-Log "ОШИБКА: ZIP архив поврежден: $BackupFile"
        return $false
    }
    
    # Создаем директорию для восстановления
    if (-not (Test-Path $RestorePath)) {
        New-Item -ItemType Directory -Path $RestorePath -Force | Out-Null
    }
    
    # Распаковываем архив
    Write-Log "Распаковка архива..."
    try {
        Expand-Archive -Path $BackupFile -DestinationPath $RestorePath -Force
        Write-Log "Архив успешно распакован"
        return $true
    } catch {
        Write-Log "ОШИБКА: Не удалось распаковать архив"
        Write-Log "Error: $($_.Exception.Message)"
        return $false
    }
}

# Функция для восстановления из директории
function Restore-FromDirectory {
    param(
        [string]$BackupDir,
        [string]$RestorePath
    )
    
    Write-Log "Восстановление из директории: $BackupDir"
    Write-Log "Восстановление в: $RestorePath"
    
    if (-not (Test-Path $BackupDir)) {
        Write-Log "ОШИБКА: Директория не найдена: $BackupDir"
        return $false
    }
    
    # Создаем директорию для восстановления
    if (-not (Test-Path $RestorePath)) {
        New-Item -ItemType Directory -Path $RestorePath -Force | Out-Null
    }
    
    # Копируем файлы
    Write-Log "Копирование файлов..."
    try {
        Copy-Item -Path "$BackupDir\*" -Destination $RestorePath -Recurse -Force
        Write-Log "Файлы успешно скопированы"
        return $true
    } catch {
        Write-Log "ОШИБКА: Не удалось скопировать файлы"
        Write-Log "Error: $($_.Exception.Message)"
        return $false
    }
}

# Основная функция
function Main {
    param([string]$BackupFileOrDir)
    
    Write-Log "=== Начало восстановления ==="
    
    # Выбираем резервную копию
    $backupPath = if ($BackupFileOrDir) {
        $BackupFileOrDir
    } else {
        Select-Backup -BackupFile $BackupFile
    }
    
    Write-Log "Выбрана резервная копия: $backupPath"
    
    # Определяем тип бэкапа и восстанавливаем
    if ($backupPath -match '\.zip$') {
        if (-not (Restore-FromZip -BackupFile $backupPath -RestorePath $RestoreDir)) {
            Write-Log "ОШИБКА: Восстановление не удалось"
            exit 1
        }
    } elseif (Test-Path $backupPath -PathType Container) {
        if (-not (Restore-FromDirectory -BackupDir $backupPath -RestorePath $RestoreDir)) {
            Write-Log "ОШИБКА: Восстановление не удалось"
            exit 1
        }
    } else {
        Write-Log "ОШИБКА: Неизвестный тип резервной копии: $backupPath"
        exit 1
    }
    
    Write-Log "=== Восстановление завершено ==="
    Write-Log "Восстановленные файлы находятся в: $RestoreDir"
    Write-Log ""
    Write-Log "ВАЖНО: Проверьте восстановленные файлы перед использованием!"
    Write-Log "ВАЖНО: Убедитесь, что сервер остановлен перед заменой файлов!"
}

# Обработка справки
if ($args -contains "--help" -or $args -contains "-h") {
    Write-Host "Использование: .\restore.ps1 [путь_к_резервной_копии]"
    Write-Host ""
    Write-Host "Примеры:"
    Write-Host "  .\restore.ps1                                    # Восстановить из последнего бэкапа"
    Write-Host "  .\restore.ps1 .\data\backups\backup_20231123.zip # Восстановить из указанного файла"
    exit 0
}

# Запуск
Main -BackupFileOrDir $BackupFile

