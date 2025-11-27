# Test multiple database upload to service
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Multiple Database Upload Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$baseUrl = "http://localhost:9999"
$testResults = @()

# Функция для создания тестового SQLite файла
function Create-TestDatabaseFile {
    param(
        [string]$FilePath,
        [int]$Size = 16
    )
    
    # SQLite файлы начинаются с "SQLite format 3\000"
    $header = [System.Text.Encoding]::UTF8.GetBytes("SQLite format 3`0")
    $content = New-Object byte[] $Size
    
    # Копируем заголовок
    for ($i = 0; $i -lt [Math]::Min($header.Length, $Size); $i++) {
        $content[$i] = $header[$i]
    }
    
    # Заполняем остальное нулями
    for ($i = $header.Length; $i -lt $Size; $i++) {
        $content[$i] = 0
    }
    
    [System.IO.File]::WriteAllBytes($FilePath, $content)
    return $FilePath
}

# Функция для загрузки файла
function Upload-DatabaseFile {
    param(
        [string]$FilePath,
        [int]$ClientID,
        [int]$ProjectID,
        [bool]$AutoCreate = $false
    )
    
    $url = "$baseUrl/api/clients/$ClientID/projects/$ProjectID/databases"
    $fileName = Split-Path -Leaf $FilePath
    
    try {
        $form = @{
            file = Get-Item $FilePath
            auto_create = if ($AutoCreate) { "true" } else { "false" }
        }
        
        $startTime = Get-Date
        $response = Invoke-RestMethod -Uri $url -Method POST -Form $form -ErrorAction Stop
        $duration = (Get-Date) - $startTime
        
        return @{
            Success = $true
            StatusCode = 200
            Response = $response
            Duration = $duration
            FileName = $fileName
        }
    } catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        $errorBody = ""
        if ($_.Exception.Response) {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            $errorBody = $reader.ReadToEnd()
        }
        
        return @{
            Success = $false
            StatusCode = $statusCode
            Error = $_.Exception.Message
            ErrorBody = $errorBody
            Duration = (Get-Date) - $startTime
            FileName = $fileName
        }
    }
}

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

# Функция для создания клиента и проекта
function Create-TestClientAndProject {
    try {
        # Создаем клиента
        $clientBody = @{
            name = "Test Client Multiple Upload $(Get-Date -Format 'yyyyMMddHHmmss')"
            legal_name = "Test Legal Name"
            description = "Test client for multiple upload testing"
        } | ConvertTo-Json
        
        $client = Invoke-RestMethod -Uri "$baseUrl/api/clients" -Method POST -Body $clientBody -ContentType "application/json" -ErrorAction Stop
        
        # Создаем проект
        $projectBody = @{
            name = "Test Project Multiple Upload"
            project_type = "nomenclature"
            description = "Test project for multiple upload testing"
        } | ConvertTo-Json
        
        $project = Invoke-RestMethod -Uri "$baseUrl/api/clients/$($client.id)/projects" -Method POST -Body $projectBody -ContentType "application/json" -ErrorAction Stop
        
        return @{
            ClientID = $client.id
            ProjectID = $project.id
        }
    } catch {
        Write-Host "  ERROR Failed to create client/project: $($_.Exception.Message)" -ForegroundColor Red
        return $null
    }
}

# Создаем временную директорию для тестовых файлов
$testDir = Join-Path $env:TEMP "multiple_upload_test_$(Get-Date -Format 'yyyyMMddHHmmss')"
New-Item -ItemType Directory -Path $testDir -Force | Out-Null
Write-Host "Created test directory: $testDir" -ForegroundColor Gray
Write-Host ""

# Test 1: Create client and project
Write-Host "[Test 1] Creating test client and project..." -ForegroundColor Yellow
$testSetup = Create-TestClientAndProject
if (-not $testSetup) {
    Write-Host "  ERROR Failed to create test setup" -ForegroundColor Red
    exit 1
}
$clientID = $testSetup.ClientID
$projectID = $testSetup.ProjectID
Write-Host "  OK Client ID: $clientID, Project ID: $projectID" -ForegroundColor Green
Write-Host ""

# Test 2: Sequential upload of 3 valid files
Write-Host "[Test 2] Sequential upload of 3 valid files..." -ForegroundColor Yellow
$testFiles = @()
for ($i = 1; $i -le 3; $i++) {
    $filePath = Join-Path $testDir "test_db_$i.db"
    Create-TestDatabaseFile -FilePath $filePath -Size 1024
    $testFiles += $filePath
    Write-Host "  Created test file: $(Split-Path -Leaf $filePath)" -ForegroundColor Gray
}

$uploadResults = @()
foreach ($filePath in $testFiles) {
    $result = Upload-DatabaseFile -FilePath $filePath -ClientID $clientID -ProjectID $projectID -AutoCreate $true
    $uploadResults += $result
    
    if ($result.Success) {
        Write-Host "  OK Uploaded: $($result.FileName) (Duration: $($result.Duration.TotalMilliseconds) ms)" -ForegroundColor Green
    } else {
        Write-Host "  ERROR Failed: $($result.FileName) - Status: $($result.StatusCode)" -ForegroundColor Red
        Write-Host "    Error: $($result.Error)" -ForegroundColor Red
    }
}

$successCount = ($uploadResults | Where-Object { $_.Success }).Count
Write-Host "  Summary: $successCount/3 files uploaded successfully" -ForegroundColor $(if ($successCount -eq 3) { "Green" } else { "Yellow" })
Write-Host ""

# Проверяем, что все базы данных созданы
$databases = Get-ProjectDatabases -ClientID $clientID -ProjectID $projectID
if ($databases -and $databases.databases.Count -eq 3) {
    Write-Host "  OK All 3 databases created in project" -ForegroundColor Green
} else {
    Write-Host "  WARNING Expected 3 databases, got $($databases.databases.Count)" -ForegroundColor Yellow
}
Write-Host ""

# Test 3: Upload files with different sizes
Write-Host "[Test 3] Upload files with different sizes..." -ForegroundColor Yellow
$sizeTestFiles = @(
    @{ Name = "small.db"; Size = 16 },
    @{ Name = "medium.db"; Size = 1024 * 100 },  # 100 KB
    @{ Name = "large.db"; Size = 1024 * 1024 }  # 1 MB
)

$sizeResults = @()
foreach ($fileSpec in $sizeTestFiles) {
    $filePath = Join-Path $testDir $fileSpec.Name
    Create-TestDatabaseFile -FilePath $filePath -Size $fileSpec.Size
    Write-Host "  Created: $($fileSpec.Name) ($($fileSpec.Size) bytes)" -ForegroundColor Gray
    
    $result = Upload-DatabaseFile -FilePath $filePath -ClientID $clientID -ProjectID $projectID -AutoCreate $true
    $sizeResults += $result
    
    if ($result.Success) {
        $sizeMB = [math]::Round($fileSpec.Size / 1MB, 2)
        $speedMBps = if ($result.Duration.TotalSeconds -gt 0) {
            [math]::Round($sizeMB / $result.Duration.TotalSeconds, 2)
        } else { 0 }
        Write-Host "  OK Uploaded: $($fileSpec.Name) - Size: $sizeMB MB, Speed: $speedMBps MB/s" -ForegroundColor Green
    } else {
        Write-Host "  ERROR Failed: $($fileSpec.Name) - $($result.Error)" -ForegroundColor Red
    }
}
Write-Host ""

# Test 4: Upload with one invalid file among valid ones
Write-Host "[Test 4] Upload with one invalid file among valid ones..." -ForegroundColor Yellow
$mixedFiles = @(
    @{ Name = "valid1.db"; Valid = $true },
    @{ Name = "invalid.txt"; Valid = $false },
    @{ Name = "valid2.db"; Valid = $true }
)

$mixedResults = @()
foreach ($fileSpec in $mixedFiles) {
    $filePath = Join-Path $testDir $fileSpec.Name
    
    if ($fileSpec.Valid) {
        Create-TestDatabaseFile -FilePath $filePath -Size 512
    } else {
        # Создаем невалидный файл
        [System.IO.File]::WriteAllText($filePath, "This is not a database file")
    }
    
    $result = Upload-DatabaseFile -FilePath $filePath -ClientID $clientID -ProjectID $projectID -AutoCreate $true
    $mixedResults += $result
    
    if ($fileSpec.Valid) {
        if ($result.Success) {
            Write-Host "  OK Valid file uploaded: $($fileSpec.Name)" -ForegroundColor Green
        } else {
            Write-Host "  ERROR Valid file failed: $($fileSpec.Name)" -ForegroundColor Red
        }
    } else {
        if (-not $result.Success) {
            Write-Host "  OK Invalid file rejected: $($fileSpec.Name) (Status: $($result.StatusCode))" -ForegroundColor Green
        } else {
            Write-Host "  ERROR Invalid file was accepted: $($fileSpec.Name)" -ForegroundColor Red
        }
    }
}

$validUploaded = ($mixedResults | Where-Object { $_.Success }).Count
$invalidRejected = ($mixedResults | Where-Object { -not $_.Success }).Count
Write-Host "  Summary: $validUploaded valid files uploaded, $invalidRejected invalid files rejected" -ForegroundColor $(if ($validUploaded -eq 2 -and $invalidRejected -eq 1) { "Green" } else { "Yellow" })
Write-Host ""

# Test 5: Upload files with duplicate names
Write-Host "[Test 5] Upload files with duplicate names..." -ForegroundColor Yellow
$duplicateName = "duplicate.db"
$duplicateFiles = @()
for ($i = 1; $i -le 3; $i++) {
    $filePath = Join-Path $testDir "$duplicateName"
    Create-TestDatabaseFile -FilePath $filePath -Size 256
    $duplicateFiles += $filePath
}

$duplicateResults = @()
foreach ($filePath in $duplicateFiles) {
    $result = Upload-DatabaseFile -FilePath $filePath -ClientID $clientID -ProjectID $projectID -AutoCreate $true
    $duplicateResults += $result
    
    if ($result.Success) {
        $savedPath = $result.Response.file_path
        Write-Host "  OK Uploaded: $duplicateName -> $savedPath" -ForegroundColor Green
    } else {
        Write-Host "  ERROR Failed: $duplicateName" -ForegroundColor Red
    }
}

# Проверяем, что файлы переименованы
$uniquePaths = $duplicateResults | Where-Object { $_.Success } | ForEach-Object { $_.Response.file_path } | Select-Object -Unique
if ($uniquePaths.Count -eq $duplicateResults.Count) {
    Write-Host "  OK All files renamed with unique names" -ForegroundColor Green
} else {
    Write-Host "  WARNING Some files may have same name" -ForegroundColor Yellow
}
Write-Host ""

# Test 6: Check upload metrics
Write-Host "[Test 6] Check upload metrics..." -ForegroundColor Yellow
$metricsFile = Join-Path $testDir "metrics_test.db"
Create-TestDatabaseFile -FilePath $metricsFile -Size 2048

$metricsResult = Upload-DatabaseFile -FilePath $metricsFile -ClientID $clientID -ProjectID $projectID -AutoCreate $true
if ($metricsResult.Success -and $metricsResult.Response.upload_metrics) {
    $metrics = $metricsResult.Response.upload_metrics
    Write-Host "  OK Metrics retrieved:" -ForegroundColor Green
    Write-Host "    Duration: $($metrics.duration_sec) seconds" -ForegroundColor Gray
    Write-Host "    File size: $([math]::Round($metrics.file_size_mb, 2)) MB" -ForegroundColor Gray
    Write-Host "    Speed: $([math]::Round($metrics.speed_mbps, 2)) MB/s" -ForegroundColor Gray
} else {
    Write-Host "  WARNING Metrics not available" -ForegroundColor Yellow
}
Write-Host ""

# Test 7: Large batch upload (5 files)
Write-Host "[Test 7] Large batch upload (5 files)..." -ForegroundColor Yellow
$batchFiles = @()
for ($i = 1; $i -le 5; $i++) {
    $filePath = Join-Path $testDir "batch_$i.db"
    Create-TestDatabaseFile -FilePath $filePath -Size 512
    $batchFiles += $filePath
}

$batchStartTime = Get-Date
$batchResults = @()
foreach ($filePath in $batchFiles) {
    $result = Upload-DatabaseFile -FilePath $filePath -ClientID $clientID -ProjectID $projectID -AutoCreate $true
    $batchResults += $result
}
$batchDuration = (Get-Date) - $batchStartTime

$batchSuccess = ($batchResults | Where-Object { $_.Success }).Count
Write-Host "  Uploaded: $batchSuccess/5 files" -ForegroundColor $(if ($batchSuccess -eq 5) { "Green" } else { "Yellow" })
Write-Host "  Total time: $($batchDuration.TotalSeconds) seconds" -ForegroundColor Gray
Write-Host "  Average time per file: $([math]::Round($batchDuration.TotalSeconds / 5, 2)) seconds" -ForegroundColor Gray
Write-Host ""

# Final summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

$allResults = $uploadResults + $sizeResults + $mixedResults + $duplicateResults + $batchResults
$allResults += $metricsResult

$totalTests = $allResults.Count
$successfulTests = ($allResults | Where-Object { $_.Success }).Count
$failedTests = $totalTests - $successfulTests

Write-Host "Total uploads: $totalTests" -ForegroundColor White
Write-Host "Successful: $successfulTests" -ForegroundColor Green
Write-Host "Failed: $failedTests" -ForegroundColor $(if ($failedTests -eq 0) { "Green" } else { "Red" })

# Проверяем финальное состояние баз данных
$finalDatabases = Get-ProjectDatabases -ClientID $clientID -ProjectID $projectID
if ($finalDatabases) {
    Write-Host "Total databases in project: $($finalDatabases.databases.Count)" -ForegroundColor White
}

# Очистка
Write-Host ""
Write-Host "Cleaning up test files..." -ForegroundColor Yellow
Remove-Item -Path $testDir -Recurse -Force -ErrorAction SilentlyContinue
Write-Host "  OK Test directory removed" -ForegroundColor Green

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test completed" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# Сохраняем результаты для отчета
$testResults = @{
    TotalTests = $totalTests
    Successful = $successfulTests
    Failed = $failedTests
    Results = $allResults
    FinalDatabaseCount = if ($finalDatabases) { $finalDatabases.databases.Count } else { 0 }
    ClientID = $clientID
    ProjectID = $projectID
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Для проверки данных в БД выполните:" -ForegroundColor Yellow
Write-Host "  .\verify_database_after_tests.ps1 -ClientID $clientID -ProjectID $projectID" -ForegroundColor Gray
Write-Host "========================================" -ForegroundColor Cyan

return $testResults

