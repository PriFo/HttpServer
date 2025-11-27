# E2E Test for Multiple Database Upload
# Tests the frontend API routes and user experience

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "E2E Test: Multiple Database Upload" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$baseUrl = "http://localhost:9999"
$frontendUrl = "http://localhost:3000"
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

# Функция для проверки доступности сервера
function Test-ServerAvailability {
    param([string]$Url)
    
    try {
        $response = Invoke-WebRequest -Uri $Url -Method GET -TimeoutSec 5 -UseBasicParsing -ErrorAction Stop
        return $true
    } catch {
        return $false
    }
}

# Test 1: Check backend availability
Write-Host "[E2E Test 1] Checking backend availability..." -ForegroundColor Yellow
$backendAvailable = Test-ServerAvailability -Url "$baseUrl/api/health"
if ($backendAvailable) {
    Write-Host "  OK Backend is available" -ForegroundColor Green
} else {
    Write-Host "  ERROR Backend is not available at $baseUrl" -ForegroundColor Red
    Write-Host "  Please start the backend server before running E2E tests" -ForegroundColor Yellow
    exit 1
}
Write-Host ""

# Test 2: Check frontend availability (optional)
Write-Host "[E2E Test 2] Checking frontend availability..." -ForegroundColor Yellow
$frontendAvailable = Test-ServerAvailability -Url $frontendUrl
if ($frontendAvailable) {
    Write-Host "  OK Frontend is available" -ForegroundColor Green
} else {
    Write-Host "  WARNING Frontend is not available at $frontendUrl" -ForegroundColor Yellow
    Write-Host "  Continuing with backend API tests only" -ForegroundColor Gray
}
Write-Host ""

# Test 3: Create test client and project
Write-Host "[E2E Test 3] Creating test client and project..." -ForegroundColor Yellow
try {
    $clientBody = @{
        name = "E2E Test Client $(Get-Date -Format 'yyyyMMddHHmmss')"
        legal_name = "E2E Test Legal Name"
        description = "E2E test client for multiple upload"
    } | ConvertTo-Json
    
    $client = Invoke-RestMethod -Uri "$baseUrl/api/clients" -Method POST -Body $clientBody -ContentType "application/json" -ErrorAction Stop
    
    $projectBody = @{
        name = "E2E Test Project"
        project_type = "nomenclature"
        description = "E2E test project for multiple upload"
    } | ConvertTo-Json
    
    $project = Invoke-RestMethod -Uri "$baseUrl/api/clients/$($client.id)/projects" -Method POST -Body $projectBody -ContentType "application/json" -ErrorAction Stop
    
    $clientID = $client.id
    $projectID = $project.id
    
    Write-Host "  OK Client ID: $clientID, Project ID: $projectID" -ForegroundColor Green
} catch {
    Write-Host "  ERROR Failed to create client/project: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Test 4: Simulate multiple file upload through frontend API route
Write-Host "[E2E Test 4] Simulating multiple file upload through frontend API..." -ForegroundColor Yellow

$testDir = Join-Path $env:TEMP "e2e_multiple_upload_$(Get-Date -Format 'yyyyMMddHHmmss')"
New-Item -ItemType Directory -Path $testDir -Force | Out-Null

# Create test files
$testFiles = @()
for ($i = 1; $i -le 3; $i++) {
    $filePath = Join-Path $testDir "e2e_test_$i.db"
    Create-TestDatabaseFile -FilePath $filePath -Size 512
    $testFiles += $filePath
}

$uploadResults = @()
foreach ($filePath in $testFiles) {
    $fileName = Split-Path -Leaf $filePath
    
    try {
        # Upload through frontend API route (which proxies to backend)
        $form = @{
            file = Get-Item $filePath
            auto_create = "true"
        }
        
        $startTime = Get-Date
        $url = "$frontendUrl/api/clients/$clientID/projects/$projectID/databases"
        
        # Try frontend first, fallback to backend
        try {
            $response = Invoke-RestMethod -Uri $url -Method POST -Form $form -ErrorAction Stop
            $viaFrontend = $true
        } catch {
            # Fallback to backend if frontend is not available
            $url = "$baseUrl/api/clients/$clientID/projects/$projectID/databases"
            $response = Invoke-RestMethod -Uri $url -Method POST -Form $form -ErrorAction Stop
            $viaFrontend = $false
        }
        
        $duration = (Get-Date) - $startTime
        
        $uploadResults += @{
            FileName = $fileName
            Success = $true
            Response = $response
            Duration = $duration
            ViaFrontend = $viaFrontend
        }
        
        Write-Host "  OK Uploaded: $fileName (via $($viaFrontend ? 'Frontend' : 'Backend'), $($duration.TotalMilliseconds) ms)" -ForegroundColor Green
    } catch {
        $uploadResults += @{
            FileName = $fileName
            Success = $false
            Error = $_.Exception.Message
        }
        Write-Host "  ERROR Failed: $fileName - $($_.Exception.Message)" -ForegroundColor Red
    }
}

$successCount = ($uploadResults | Where-Object { $_.Success }).Count
Write-Host "  Summary: $successCount/3 files uploaded successfully" -ForegroundColor $(if ($successCount -eq 3) { "Green" } else { "Yellow" })
Write-Host ""

# Test 5: Check progress tracking (simulate frontend behavior)
Write-Host "[E2E Test 5] Testing progress tracking..." -ForegroundColor Yellow

# Simulate frontend checking upload status
try {
    $databases = Invoke-RestMethod -Uri "$baseUrl/api/clients/$clientID/projects/$projectID/databases" -Method GET -ErrorAction Stop
    
    if ($databases.databases.Count -eq $successCount) {
        Write-Host "  OK All uploaded databases are visible in project ($($databases.databases.Count) databases)" -ForegroundColor Green
        
        foreach ($db in $databases.databases) {
            Write-Host "    - $($db.name) (ID: $($db.id), Size: $([math]::Round($db.file_size/1KB, 2)) KB)" -ForegroundColor Gray
        }
    } else {
        Write-Host "  WARNING Expected $successCount databases, found $($databases.databases.Count)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "  ERROR Failed to get databases: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# Test 6: Test error handling (upload invalid file)
Write-Host "[E2E Test 6] Testing error handling..." -ForegroundColor Yellow

$invalidFile = Join-Path $testDir "invalid.txt"
[System.IO.File]::WriteAllText($invalidFile, "This is not a database file")

try {
    $form = @{
        file = Get-Item $invalidFile
        auto_create = "true"
    }
    
    $url = "$baseUrl/api/clients/$clientID/projects/$projectID/databases"
    $response = Invoke-RestMethod -Uri $url -Method POST -Form $form -ErrorAction Stop
    
    Write-Host "  ERROR Invalid file was accepted (should be rejected)" -ForegroundColor Red
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    if ($statusCode -eq 400) {
        Write-Host "  OK Invalid file correctly rejected (Status: 400)" -ForegroundColor Green
    } else {
        Write-Host "  WARNING Unexpected status code: $statusCode" -ForegroundColor Yellow
    }
}
Write-Host ""

# Test 7: Test mixed file types (valid and invalid)
Write-Host "[E2E Test 7] Testing mixed file types..." -ForegroundColor Yellow

$mixedFiles = @(
    @{ Path = Join-Path $testDir "valid_mixed.db"; Valid = $true },
    @{ Path = Join-Path $testDir "invalid_mixed.txt"; Valid = $false },
    @{ Path = Join-Path $testDir "valid_mixed2.db"; Valid = $true }
)

$mixedResults = @()
foreach ($fileSpec in $mixedFiles) {
    if ($fileSpec.Valid) {
        Create-TestDatabaseFile -FilePath $fileSpec.Path -Size 256
    } else {
        [System.IO.File]::WriteAllText($fileSpec.Path, "not a database")
    }
    
    try {
        $form = @{
            file = Get-Item $fileSpec.Path
            auto_create = "true"
        }
        
        $url = "$baseUrl/api/clients/$clientID/projects/$projectID/databases"
        $response = Invoke-RestMethod -Uri $url -Method POST -Form $form -ErrorAction Stop
        
        if ($fileSpec.Valid) {
            $mixedResults += @{ File = Split-Path -Leaf $fileSpec.Path; Success = $true; Expected = $true }
            Write-Host "  OK Valid file uploaded: $(Split-Path -Leaf $fileSpec.Path)" -ForegroundColor Green
        } else {
            $mixedResults += @{ File = Split-Path -Leaf $fileSpec.Path; Success = $true; Expected = $false }
            Write-Host "  ERROR Invalid file was accepted: $(Split-Path -Leaf $fileSpec.Path)" -ForegroundColor Red
        }
    } catch {
        if ($fileSpec.Valid) {
            $mixedResults += @{ File = Split-Path -Leaf $fileSpec.Path; Success = $false; Expected = $true }
            Write-Host "  ERROR Valid file was rejected: $(Split-Path -Leaf $fileSpec.Path)" -ForegroundColor Red
        } else {
            $mixedResults += @{ File = Split-Path -Leaf $fileSpec.Path; Success = $false; Expected = $false }
            Write-Host "  OK Invalid file correctly rejected: $(Split-Path -Leaf $fileSpec.Path)" -ForegroundColor Green
        }
    }
}

$correctHandling = ($mixedResults | Where-Object { $_.Success -eq $_.Expected }).Count
Write-Host "  Summary: $correctHandling/3 files handled correctly" -ForegroundColor $(if ($correctHandling -eq 3) { "Green" } else { "Yellow" })
Write-Host ""

# Test 8: Test duplicate file names
Write-Host "[E2E Test 8] Testing duplicate file names..." -ForegroundColor Yellow

$duplicateName = "duplicate_e2e.db"
$duplicateFiles = @()
for ($i = 1; $i -le 2; $i++) {
    $filePath = Join-Path $testDir $duplicateName
    Create-TestDatabaseFile -FilePath $filePath -Size 256
    $duplicateFiles += $filePath
}

$duplicateResults = @()
foreach ($filePath in $duplicateFiles) {
    try {
        $form = @{
            file = Get-Item $filePath
            auto_create = "true"
        }
        
        $url = "$baseUrl/api/clients/$clientID/projects/$projectID/databases"
        $response = Invoke-RestMethod -Uri $url -Method POST -Form $form -ErrorAction Stop
        
        $savedPath = $response.file_path
        $duplicateResults += @{ Original = $duplicateName; Saved = $savedPath; Success = $true }
        Write-Host "  OK Uploaded: $duplicateName -> $(Split-Path -Leaf $savedPath)" -ForegroundColor Green
    } catch {
        $duplicateResults += @{ Original = $duplicateName; Success = $false; Error = $_.Exception.Message }
        Write-Host "  ERROR Failed: $duplicateName" -ForegroundColor Red
    }
}

# Check if files were renamed
$uniquePaths = $duplicateResults | Where-Object { $_.Success } | ForEach-Object { $_.Saved } | Select-Object -Unique
if ($uniquePaths.Count -eq $duplicateResults.Count) {
    Write-Host "  OK All files renamed with unique names" -ForegroundColor Green
} else {
    Write-Host "  WARNING Some files may have same name" -ForegroundColor Yellow
}
Write-Host ""

# Final summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "E2E Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

$allResults = $uploadResults + $mixedResults
$totalTests = $allResults.Count
$successfulTests = ($allResults | Where-Object { $_.Success -or ($_.Success -eq $_.Expected) }).Count

Write-Host "Total tests: $totalTests" -ForegroundColor White
Write-Host "Successful: $successfulTests" -ForegroundColor Green
Write-Host "Failed: $($totalTests - $successfulTests)" -ForegroundColor $(if (($totalTests - $successfulTests) -eq 0) { "Green" } else { "Red" })

# Cleanup
Write-Host ""
Write-Host "Cleaning up test files..." -ForegroundColor Yellow
Remove-Item -Path $testDir -Recurse -Force -ErrorAction SilentlyContinue
Write-Host "  OK Test directory removed" -ForegroundColor Green

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "E2E Test completed" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

return @{
    TotalTests = $totalTests
    Successful = $successfulTests
    Failed = $totalTests - $successfulTests
    Results = $allResults
}

