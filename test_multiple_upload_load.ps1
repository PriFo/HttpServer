# Load Test for Multiple Database Upload
# Tests performance with large batches and large files

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Load Test: Multiple Database Upload" -ForegroundColor Cyan
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
    
    $header = [System.Text.Encoding]::UTF8.GetBytes("SQLite format 3`0")
    $content = New-Object byte[] $Size
    
    for ($i = 0; $i -lt [Math]::Min($header.Length, $Size); $i++) {
        $content[$i] = $header[$i]
    }
    
    for ($i = $header.Length; $i -lt $Size; $i++) {
        $content[$i] = 0
    }
    
    [System.IO.File]::WriteAllBytes($FilePath, $content)
    return $FilePath
}

# Функция для загрузки файла с измерением времени
function Upload-DatabaseFileWithMetrics {
    param(
        [string]$FilePath,
        [int]$ClientID,
        [int]$ProjectID
    )
    
    $url = "$baseUrl/api/clients/$ClientID/projects/$ProjectID/databases"
    
    try {
        $form = @{
            file = Get-Item $FilePath
            auto_create = "true"
        }
        
        $startTime = Get-Date
        $response = Invoke-RestMethod -Uri $url -Method POST -Form $form -ErrorAction Stop
        $duration = (Get-Date) - $startTime
        
        $fileSize = (Get-Item $FilePath).Length
        $speedMBps = if ($duration.TotalSeconds -gt 0) {
            ($fileSize / 1MB) / $duration.TotalSeconds
        } else { 0 }
        
        return @{
            Success = $true
            Duration = $duration
            FileSize = $fileSize
            SpeedMBps = $speedMBps
            Response = $response
        }
    } catch {
        return @{
            Success = $false
            Error = $_.Exception.Message
            Duration = (Get-Date) - $startTime
        }
    }
}

# Создаем тестовую среду
Write-Host "[Setup] Creating test client and project..." -ForegroundColor Yellow
try {
    $clientBody = @{
        name = "Load Test Client $(Get-Date -Format 'yyyyMMddHHmmss')"
        legal_name = "Load Test Legal Name"
        description = "Load test client"
    } | ConvertTo-Json
    
    $client = Invoke-RestMethod -Uri "$baseUrl/api/clients" -Method POST -Body $clientBody -ContentType "application/json" -ErrorAction Stop
    
    $projectBody = @{
        name = "Load Test Project"
        project_type = "nomenclature"
        description = "Load test project"
    } | ConvertTo-Json
    
    $project = Invoke-RestMethod -Uri "$baseUrl/api/clients/$($client.id)/projects" -Method POST -Body $projectBody -ContentType "application/json" -ErrorAction Stop
    
    $clientID = $client.id
    $projectID = $project.id
    
    Write-Host "  OK Client ID: $clientID, Project ID: $projectID" -ForegroundColor Green
} catch {
    Write-Host "  ERROR Failed to create test setup: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
Write-Host ""

$testDir = Join-Path $env:TEMP "load_test_$(Get-Date -Format 'yyyyMMddHHmmss')"
New-Item -ItemType Directory -Path $testDir -Force | Out-Null

# Test 1: Large batch upload (10+ files)
Write-Host "[Load Test 1] Large batch upload (15 files)..." -ForegroundColor Yellow

$batchFiles = @()
for ($i = 1; $i -le 15; $i++) {
    $filePath = Join-Path $testDir "batch_$i.db"
    Create-TestDatabaseFile -FilePath $filePath -Size (512 * 1024) # 512 KB
    $batchFiles += $filePath
}

$batchStartTime = Get-Date
$batchResults = @()
foreach ($filePath in $batchFiles) {
    $result = Upload-DatabaseFileWithMetrics -FilePath $filePath -ClientID $clientID -ProjectID $projectID
    $batchResults += $result
    
    if ($result.Success) {
        Write-Host "  OK $(Split-Path -Leaf $filePath): $([math]::Round($result.Duration.TotalMilliseconds)) ms, $([math]::Round($result.SpeedMBps, 2)) MB/s" -ForegroundColor Green
    } else {
        Write-Host "  ERROR $(Split-Path -Leaf $filePath): $($result.Error)" -ForegroundColor Red
    }
}
$batchDuration = (Get-Date) - $batchStartTime

$batchSuccess = ($batchResults | Where-Object { $_.Success }).Count
$avgSpeed = ($batchResults | Where-Object { $_.Success } | ForEach-Object { $_.SpeedMBps } | Measure-Object -Average).Average

Write-Host "  Summary: $batchSuccess/15 files uploaded" -ForegroundColor $(if ($batchSuccess -eq 15) { "Green" } else { "Yellow" })
Write-Host "  Total time: $([math]::Round($batchDuration.TotalSeconds, 2)) seconds" -ForegroundColor Gray
Write-Host "  Average speed: $([math]::Round($avgSpeed, 2)) MB/s" -ForegroundColor Gray
Write-Host "  Average time per file: $([math]::Round($batchDuration.TotalSeconds / 15, 2)) seconds" -ForegroundColor Gray
Write-Host ""

# Test 2: Large files upload (close to 500MB limit)
Write-Host "[Load Test 2] Large files upload (near 500MB limit)..." -ForegroundColor Yellow

$largeFileSizes = @(
    @{ Name = "large_50mb.db"; Size = 50 * 1024 * 1024 },   # 50 MB
    @{ Name = "large_100mb.db"; Size = 100 * 1024 * 1024 }, # 100 MB
    @{ Name = "large_200mb.db"; Size = 200 * 1024 * 1024 }  # 200 MB
)

$largeResults = @()
foreach ($fileSpec in $largeFileSizes) {
    $filePath = Join-Path $testDir $fileSpec.Name
    Write-Host "  Creating $($fileSpec.Name) ($([math]::Round($fileSpec.Size/1MB, 0)) MB)..." -ForegroundColor Gray
    
    $createStart = Get-Date
    Create-TestDatabaseFile -FilePath $filePath -Size $fileSpec.Size
    $createDuration = (Get-Date) - $createStart
    Write-Host "    File created in $([math]::Round($createDuration.TotalSeconds, 2)) seconds" -ForegroundColor Gray
    
    $result = Upload-DatabaseFileWithMetrics -FilePath $filePath -ClientID $clientID -ProjectID $projectID
    $largeResults += $result
    
    if ($result.Success) {
        Write-Host "  OK $($fileSpec.Name): $([math]::Round($result.Duration.TotalSeconds, 2))s, $([math]::Round($result.SpeedMBps, 2)) MB/s" -ForegroundColor Green
    } else {
        Write-Host "  ERROR $($fileSpec.Name): $($result.Error)" -ForegroundColor Red
    }
}

$largeSuccess = ($largeResults | Where-Object { $_.Success }).Count
Write-Host "  Summary: $largeSuccess/3 large files uploaded" -ForegroundColor $(if ($largeSuccess -eq 3) { "Green" } else { "Yellow" })
Write-Host ""

# Test 3: Mixed batch (different sizes)
Write-Host "[Load Test 3] Mixed batch upload (different sizes)..." -ForegroundColor Yellow

$mixedSizes = @(
    @{ Name = "mixed_small.db"; Size = 16 },
    @{ Name = "mixed_medium.db"; Size = 1024 * 1024 },      # 1 MB
    @{ Name = "mixed_large.db"; Size = 10 * 1024 * 1024 },   # 10 MB
    @{ Name = "mixed_small2.db"; Size = 512 },
    @{ Name = "mixed_medium2.db"; Size = 5 * 1024 * 1024 }  # 5 MB
)

$mixedResults = @()
$mixedStartTime = Get-Date
foreach ($fileSpec in $mixedSizes) {
    $filePath = Join-Path $testDir $fileSpec.Name
    Create-TestDatabaseFile -FilePath $filePath -Size $fileSpec.Size
    
    $result = Upload-DatabaseFileWithMetrics -FilePath $filePath -ClientID $clientID -ProjectID $projectID
    $mixedResults += $result
}
$mixedDuration = (Get-Date) - $mixedStartTime

$mixedSuccess = ($mixedResults | Where-Object { $_.Success }).Count
$totalSize = ($mixedResults | Where-Object { $_.Success } | ForEach-Object { $_.FileSize } | Measure-Object -Sum).Sum

Write-Host "  Summary: $mixedSuccess/5 files uploaded" -ForegroundColor $(if ($mixedSuccess -eq 5) { "Green" } else { "Yellow" })
Write-Host "  Total size: $([math]::Round($totalSize/1MB, 2)) MB" -ForegroundColor Gray
Write-Host "  Total time: $([math]::Round($mixedDuration.TotalSeconds, 2)) seconds" -ForegroundColor Gray
Write-Host "  Average speed: $([math]::Round(($totalSize/1MB) / $mixedDuration.TotalSeconds, 2)) MB/s" -ForegroundColor Gray
Write-Host ""

# Performance summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Load Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

$allResults = $batchResults + $largeResults + $mixedResults
$totalTests = $allResults.Count
$successfulTests = ($allResults | Where-Object { $_.Success }).Count
$totalDuration = ($allResults | Where-Object { $_.Success } | ForEach-Object { $_.Duration.TotalSeconds } | Measure-Object -Sum).Sum
$totalSize = ($allResults | Where-Object { $_.Success } | ForEach-Object { $_.FileSize } | Measure-Object -Sum).Sum
$avgSpeed = if ($totalDuration -gt 0) {
    ($totalSize / 1MB) / $totalDuration
} else { 0 }

Write-Host "Total uploads: $totalTests" -ForegroundColor White
Write-Host "Successful: $successfulTests" -ForegroundColor Green
Write-Host "Failed: $($totalTests - $successfulTests)" -ForegroundColor $(if (($totalTests - $successfulTests) -eq 0) { "Green" } else { "Red" })
Write-Host "Total size uploaded: $([math]::Round($totalSize/1MB, 2)) MB" -ForegroundColor White
Write-Host "Total time: $([math]::Round($totalDuration, 2)) seconds" -ForegroundColor White
Write-Host "Average speed: $([math]::Round($avgSpeed, 2)) MB/s" -ForegroundColor White

# Performance metrics
$minSpeed = ($allResults | Where-Object { $_.Success } | ForEach-Object { $_.SpeedMBps } | Measure-Object -Minimum).Minimum
$maxSpeed = ($allResults | Where-Object { $_.Success } | ForEach-Object { $_.SpeedMBps } | Measure-Object -Maximum).Maximum

Write-Host ""
Write-Host "Performance Metrics:" -ForegroundColor Cyan
Write-Host "  Min speed: $([math]::Round($minSpeed, 2)) MB/s" -ForegroundColor Gray
Write-Host "  Max speed: $([math]::Round($maxSpeed, 2)) MB/s" -ForegroundColor Gray
Write-Host "  Average speed: $([math]::Round($avgSpeed, 2)) MB/s" -ForegroundColor Gray

# Cleanup
Write-Host ""
Write-Host "Cleaning up test files..." -ForegroundColor Yellow
Remove-Item -Path $testDir -Recurse -Force -ErrorAction SilentlyContinue
Write-Host "  OK Test directory removed" -ForegroundColor Green

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Load test completed" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

return @{
    TotalTests = $totalTests
    Successful = $successfulTests
    Failed = $totalTests - $successfulTests
    TotalSize = $totalSize
    TotalDuration = $totalDuration
    AverageSpeed = $avgSpeed
    Results = $allResults
}

