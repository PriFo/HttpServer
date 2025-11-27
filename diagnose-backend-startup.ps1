# Диагностический скрипт для выявления проблем запуска бэкенда
# Запускает сервер с полным логированием всех ошибок

param(
    [switch]$UseGoRun = $false,  # Использовать go run вместо exe
    [switch]$KeepRunning = $false  # Не останавливать после диагностики
)

$ErrorActionPreference = "Continue"
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptPath

$logFile = "backend-startup-diagnostic-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
$env:ARLIAI_API_KEY = "597dbe7e-16ca-4803-ab17-5fa084909f37"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Диагностика запуска бэкенда" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Лог файл: $logFile" -ForegroundColor Yellow
Write-Host ""

# Функция для логирования
function Write-Log {
    param([string]$Message, [string]$Color = "White")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] $Message"
    Write-Host $logMessage -ForegroundColor $Color
    Add-Content -Path $logFile -Value $logMessage
}

# Проверка окружения
Write-Log "=== Проверка окружения ===" "Cyan"

# Проверка Go (если используется go run)
if ($UseGoRun) {
    $goVersion = & go version 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Log "✓ Go установлен: $goVersion" "Green"
    } else {
        Write-Log "✗ Go не найден в PATH" "Red"
        exit 1
    }
}

# Проверка файлов БД
Write-Log "=== Проверка файлов баз данных ===" "Cyan"
$dbFiles = @("1c_data.db", "data.db", "normalized_data.db", "service.db")
foreach ($dbFile in $dbFiles) {
    if (Test-Path $dbFile) {
        $size = (Get-Item $dbFile).Length
        Write-Log "✓ $dbFile существует ($([math]::Round($size/1KB, 2)) KB)" "Green"
    } else {
        Write-Log "⚠ $dbFile не найден (будет создан при запуске)" "Yellow"
    }
}

# Проверка папки data
if (-not (Test-Path "data")) {
    Write-Log "⚠ Папка 'data' не существует" "Yellow"
    try {
        New-Item -ItemType Directory -Path "data" -Force | Out-Null
        Write-Log "✓ Папка 'data' создана" "Green"
    } catch {
        Write-Log "✗ Не удалось создать папку 'data': $_" "Red"
    }
}

# Проверка порта 9999
Write-Log "=== Проверка порта 9999 ===" "Cyan"
$portCheck = Get-NetTCPConnection -LocalPort 9999 -ErrorAction SilentlyContinue
if ($portCheck) {
    $processId = $portCheck.OwningProcess
    $processName = (Get-Process -Id $processId -ErrorAction SilentlyContinue).ProcessName
    Write-Log "⚠ Порт 9999 уже занят процессом: $processName (PID: $processId)" "Yellow"
    Write-Log "  Рекомендуется остановить процесс перед запуском" "Yellow"
} else {
    Write-Log "✓ Порт 9999 свободен" "Green"
}

# Проверка исполняемого файла
if (-not $UseGoRun) {
    Write-Log "=== Проверка исполняемого файла ===" "Cyan"
    if (Test-Path "httpserver_no_gui.exe") {
        $exeSize = (Get-Item "httpserver_no_gui.exe").Length
        $exeDate = (Get-Item "httpserver_no_gui.exe").LastWriteTime
        Write-Log "✓ httpserver_no_gui.exe найден ($([math]::Round($exeSize/1MB, 2)) MB, изменен: $exeDate)" "Green"
    } else {
        Write-Log "✗ httpserver_no_gui.exe не найден, переключение на go run" "Yellow"
        $UseGoRun = $true
    }
}

# Переменные окружения
Write-Log "=== Переменные окружения ===" "Cyan"
Write-Log "  ARLIAI_API_KEY: $([string]::new('*', [Math]::Min($env:ARLIAI_API_KEY.Length, 10)))..." "Gray"
Write-Log "  SERVER_PORT: $($env:SERVER_PORT)" "Gray"
Write-Log "  DATABASE_PATH: $($env:DATABASE_PATH)" "Gray"

Write-Log ""
Write-Log "=== Запуск бэкенда ===" "Cyan"
Write-Log "Режим: $(if ($UseGoRun) { 'go run' } else { 'exe' })" "Yellow"

# Запуск с перехватом вывода
$processInfo = New-Object System.Diagnostics.ProcessStartInfo
if ($UseGoRun) {
    $processInfo.FileName = "go"
    $processInfo.Arguments = "run -tags no_gui main_no_gui.go"
} else {
    $processInfo.FileName = "$scriptPath\httpserver_no_gui.exe"
    $processInfo.Arguments = ""
}
$processInfo.WorkingDirectory = $scriptPath
$processInfo.UseShellExecute = $false
$processInfo.RedirectStandardOutput = $true
$processInfo.RedirectStandardError = $true
$processInfo.CreateNoWindow = $false  # Показываем окно, чтобы видеть ошибки

$process = New-Object System.Diagnostics.Process
$process.StartInfo = $processInfo

# Буферы для вывода
$outputBuilder = New-Object System.Text.StringBuilder
$errorBuilder = New-Object System.Text.StringBuilder

$outputEvent = Register-ObjectEvent -InputObject $process -EventName OutputDataReceived -Action {
    if ($EventArgs.Data) {
        [void]$Event.MessageData.AppendLine($EventArgs.Data)
        Write-Host $EventArgs.Data
        Add-Content -Path $Event.MessageData.LogFile -Value $EventArgs.Data
    }
}.MessageData = @{LogFile = $logFile}

$errorEvent = Register-ObjectEvent -InputObject $process -EventName ErrorDataReceived -Action {
    if ($EventArgs.Data) {
        [void]$Event.MessageData.AppendLine($EventArgs.Data)
        Write-Host $EventArgs.Data -ForegroundColor Red
        Add-Content -Path $Event.MessageData.LogFile -Value "[ERROR] $($EventArgs.Data)"
    }
}.MessageData = @{LogFile = $logFile}

try {
    Write-Log "Запуск процесса..." "Yellow"
    $process.Start() | Out-Null
    $process.BeginOutputReadLine()
    $process.BeginErrorReadLine()
    
    # Ждем несколько секунд и проверяем статус
    Start-Sleep -Seconds 3
    
    if ($process.HasExited) {
        Write-Log "✗ Процесс завершился с кодом: $($process.ExitCode)" "Red"
        Write-Log "Проверьте лог файл для деталей: $logFile" "Yellow"
        
        # Пытаемся прочитать последние строки вывода
        Start-Sleep -Seconds 1  # Даем время на завершение событий
        
        exit $process.ExitCode
    } else {
        Write-Log "✓ Процесс запущен (PID: $($process.Id))" "Green"
        
        # Проверяем, слушает ли порт
        Start-Sleep -Seconds 5
        
        $portCheck = Get-NetTCPConnection -LocalPort 9999 -ErrorAction SilentlyContinue
        if ($portCheck) {
            Write-Log "✓ Сервер слушает на порту 9999" "Green"
            
            # Проверяем health endpoint
            try {
                $response = Invoke-WebRequest -Uri "http://localhost:9999/health" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
                Write-Log "✓ Health check успешен: $($response.StatusCode)" "Green"
                Write-Log "  Ответ: $($response.Content)" "Gray"
            } catch {
                Write-Log "⚠ Health check не прошел: $_" "Yellow"
            }
        } else {
            Write-Log "⚠ Порт 9999 еще не занят (сервер может еще запускаться)" "Yellow"
        }
        
        if ($KeepRunning) {
            Write-Log ""
            Write-Log "=== Сервер запущен ===" "Green"
            Write-Log "Нажмите Ctrl+C для остановки" "Yellow"
            Write-Log "Логи сохраняются в: $logFile" "Gray"
            $process.WaitForExit()
        } else {
            Write-Log ""
            Write-Log "=== Диагностика завершена ===" "Cyan"
            Write-Log "Процесс продолжает работать. Проверьте логи в файле: $logFile" "Yellow"
            Write-Log "Для остановки процесса используйте: Stop-Process -Id $($process.Id)" "Gray"
        }
    }
} catch {
    Write-Log "✗ Ошибка при запуске: $_" "Red"
    Write-Log "  Stack trace: $($_.ScriptStackTrace)" "Red"
    exit 1
} finally {
    if ($outputEvent) { Unregister-Event -SourceIdentifier $outputEvent.Name }
    if ($errorEvent) { Unregister-Event -SourceIdentifier $errorEvent.Name }
}

