# Edge Cases Test for Multiple Database Upload
# Tests boundary conditions, special characters, mixed file types, interruptions

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Edge Cases Test: Multiple Database Upload" -ForegroundColor Cyan
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

# Функция для загрузки файла
function Upload-DatabaseFile {
    param(
        [string]$FilePath,
        [int]$ClientID,
        [int]$ProjectID,
        [int]$TimeoutSeconds = 30
    )
    
    $url = "$baseUrl/api/clients/$ClientID/projects/$ProjectID/databases"
    
    try {
        $form = @{
            file = Get-Item $FilePath
            auto_create = "true"
        }
        
        $response = Invoke-RestMethod -Uri $url -Method POST -Form $form -TimeoutSec $TimeoutSeconds -ErrorAction Stop
        return @{
            Success = $true
            Response = $response
            FileName = Split-Path -Leaf $FilePath
        }
    } catch {
        $statusCode = if ($_.Exception.Response) { $_.Exception.Response.StatusCode.value__ } else { 0 }
        return @{
            Success = $false
            StatusCode = $statusCode
            Error = $_.Exception.Message
            FileName = Split-Path -Leaf $FilePath
        }
    }
}

# Создаем тестовую среду
Write-Host "[Setup] Creating test client and project..." -ForegroundColor Yellow
try {
    $clientBody = @{
        name = "Edge Cases Test Client $(Get-Date -Format 'yyyyMMddHHmmss')"
        legal_name = "Edge Cases Test"
        description = "Edge cases test client"
    } | ConvertTo-Json
    
    $client = Invoke-RestMethod -Uri "$baseUrl/api/clients" -Method POST -Body $clientBody -ContentType "application/json" -ErrorAction Stop
    
    $projectBody = @{
        name = "Edge Cases Test Project"
        project_type = "nomenclature"
        description = "Edge cases test project"
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

$testDir = Join-Path $env:TEMP "edge_cases_test_$(Get-Date -Format 'yyyyMMddHHmmss')"
New-Item -ItemType Directory -Path $testDir -Force | Out-Null

# Test 1: Files with special characters in names
Write-Host "[Edge Case 1] Files with special characters in names..." -ForegroundColor Yellow

$specialCharFiles = @(
    "file with spaces.db",
    "file-with-dashes.db",
    "file_with_underscores.db",
    "file.with.dots.db",
    "file(1).db",
    "file[2].db",
    "file{3}.db",
    "file@4.db",
    "file#5.db",
    "file$6.db",
    "file%7.db",
    "file&8.db",
    "file+9.db",
    "file=10.db",
    "file!11.db",
    "file~12.db",
    "file`13.db",
    "file^14.db",
    "file'15.db"
)

$specialResults = @()
foreach ($fileName in $specialCharFiles) {
    $filePath = Join-Path $testDir $fileName
    Create-TestDatabaseFile -FilePath $filePath -Size 256
    
    $result = Upload-DatabaseFile -FilePath $filePath -ClientID $clientID -ProjectID $projectID
    $specialResults += $result
    
    if ($result.Success) {
        Write-Host "  OK Uploaded: $fileName" -ForegroundColor Green
    } else {
        Write-Host "  ERROR Failed: $fileName - $($result.Error)" -ForegroundColor Red
    }
}

$specialSuccess = ($specialResults | Where-Object { $_.Success }).Count
Write-Host "  Summary: $specialSuccess/$($specialCharFiles.Count) files with special characters uploaded" -ForegroundColor $(if ($specialSuccess -eq $specialCharFiles.Count) { "Green" } else { "Yellow" })
Write-Host ""

# Test 2: Very long file names
Write-Host "[Edge Case 2] Very long file names..." -ForegroundColor Yellow

$longNameTests = @(
    @{ Length = 50; Name = "a" * 46 + ".db" },
    @{ Length = 200; Name = "a" * 196 + ".db" },
    @{ Length = 255; Name = "a" * 251 + ".db" },
    @{ Length = 300; Name = "a" * 296 + ".db" }  # Should be truncated
)

$longNameResults = @()
foreach ($test in $longNameTests) {
    $filePath = Join-Path $testDir $test.Name
    Create-TestDatabaseFile -FilePath $filePath -Size 256
    
    $result = Upload-DatabaseFile -FilePath $filePath -ClientID $clientID -ProjectID $projectID
    $longNameResults += $result
    
    if ($result.Success) {
        $savedName = $result.Response.file_name
        if ($savedName.Length -le 255) {
            Write-Host "  OK Uploaded: $($test.Name.Substring(0, [Math]::Min(50, $test.Name.Length)))... (truncated to $($savedName.Length) chars)" -ForegroundColor Green
        } else {
            Write-Host "  WARNING Name not truncated: $($savedName.Length) chars" -ForegroundColor Yellow
        }
    } else {
        Write-Host "  ERROR Failed: $($test.Name.Substring(0, [Math]::Min(50, $test.Name.Length)))..." -ForegroundColor Red
    }
}
Write-Host ""

# Test 3: Mixed file types (.db and non-.db)
Write-Host "[Edge Case 3] Mixed file types (.db and non-.db)..." -ForegroundColor Yellow

$mixedTypeFiles = @(
    @{ Name = "valid1.db"; Valid = $true },
    @{ Name = "invalid1.txt"; Valid = $false },
    @{ Name = "valid2.db"; Valid = $true },
    @{ Name = "invalid2.pdf"; Valid = $false },
    @{ Name = "valid3.db"; Valid = $true },
    @{ Name = "invalid3.exe"; Valid = $false },
    @{ Name = "valid4.db"; Valid = $true }
)

$mixedTypeResults = @()
foreach ($fileSpec in $mixedTypeFiles) {
    $filePath = Join-Path $testDir $fileSpec.Name
    
    if ($fileSpec.Valid) {
        Create-TestDatabaseFile -FilePath $filePath -Size 256
    } else {
        [System.IO.File]::WriteAllText($filePath, "This is not a database file")
    }
    
    $result = Upload-DatabaseFile -FilePath $filePath -ClientID $clientID -ProjectID $projectID
    $mixedTypeResults += $result
    
    if ($fileSpec.Valid) {
        if ($result.Success) {
            Write-Host "  OK Valid file uploaded: $($fileSpec.Name)" -ForegroundColor Green
        } else {
            Write-Host "  ERROR Valid file rejected: $($fileSpec.Name)" -ForegroundColor Red
        }
    } else {
        if (-not $result.Success) {
            Write-Host "  OK Invalid file rejected: $($fileSpec.Name) (Status: $($result.StatusCode))" -ForegroundColor Green
        } else {
            Write-Host "  ERROR Invalid file accepted: $($fileSpec.Name)" -ForegroundColor Red
        }
    }
}

$validUploaded = ($mixedTypeResults | Where-Object { $_.Success }).Count
$invalidRejected = ($mixedTypeResults | Where-Object { -not $_.Success }).Count
$expectedValid = ($mixedTypeFiles | Where-Object { $_.Valid }).Count
$expectedInvalid = ($mixedTypeFiles | Where-Object { -not $_.Valid }).Count

Write-Host "  Summary: $validUploaded/$expectedValid valid uploaded, $invalidRejected/$expectedInvalid invalid rejected" -ForegroundColor $(if ($validUploaded -eq $expectedValid -and $invalidRejected -eq $expectedInvalid) { "Green" } else { "Yellow" })
Write-Host ""

# Test 4: Duplicate file names (should be renamed)
Write-Host "[Edge Case 4] Duplicate file names (should be renamed)..." -ForegroundColor Yellow

$duplicateName = "duplicate_edge.db"
$duplicateFiles = @()
for ($i = 1; $i -le 5; $i++) {
    $filePath = Join-Path $testDir $duplicateName
    Create-TestDatabaseFile -FilePath $filePath -Size 256
    $duplicateFiles += $filePath
}

$duplicateResults = @()
$uniquePaths = @{}
foreach ($filePath in $duplicateFiles) {
    $result = Upload-DatabaseFile -FilePath $filePath -ClientID $clientID -ProjectID $projectID
    $duplicateResults += $result
    
    if ($result.Success) {
        $savedPath = $result.Response.file_path
        $savedName = Split-Path -Leaf $savedPath
        $uniquePaths[$savedName] = $true
        Write-Host "  OK Uploaded: $duplicateName -> $savedName" -ForegroundColor Green
    } else {
        Write-Host "  ERROR Failed: $duplicateName" -ForegroundColor Red
    }
}

if ($uniquePaths.Count -eq $duplicateResults.Count) {
    Write-Host "  OK All files renamed with unique names ($($uniquePaths.Count) unique)" -ForegroundColor Green
} else {
    Write-Host "  WARNING Some files may have same name" -ForegroundColor Yellow
}
Write-Host ""

# Test 5: Empty and minimal files
Write-Host "[Edge Case 5] Empty and minimal files..." -ForegroundColor Yellow

$minimalFiles = @(
    @{ Name = "empty.db"; Size = 0; ShouldFail = $true },
    @{ Name = "minimal_16b.db"; Size = 16; ShouldFail = $false },
    @{ Name = "minimal_15b.db"; Size = 15; ShouldFail = $true }
)

$minimalResults = @()
foreach ($fileSpec in $minimalFiles) {
    $filePath = Join-Path $testDir $fileSpec.Name
    
    if ($fileSpec.Size -gt 0) {
        Create-TestDatabaseFile -FilePath $filePath -Size $fileSpec.Size
    } else {
        # Create empty file
        [System.IO.File]::WriteAllBytes($filePath, @())
    }
    
    $result = Upload-DatabaseFile -FilePath $filePath -ClientID $clientID -ProjectID $projectID
    $minimalResults += $result
    
    if ($fileSpec.ShouldFail) {
        if (-not $result.Success) {
            Write-Host "  OK Correctly rejected: $($fileSpec.Name) (Size: $($fileSpec.Size) bytes)" -ForegroundColor Green
        } else {
            Write-Host "  ERROR Should be rejected: $($fileSpec.Name)" -ForegroundColor Red
        }
    } else {
        if ($result.Success) {
            Write-Host "  OK Correctly accepted: $($fileSpec.Name) (Size: $($fileSpec.Size) bytes)" -ForegroundColor Green
        } else {
            Write-Host "  ERROR Should be accepted: $($fileSpec.Name)" -ForegroundColor Red
        }
    }
}
Write-Host ""

# Test 6: Files with path traversal attempts (should be sanitized)
Write-Host "[Edge Case 6] Files with path traversal attempts..." -ForegroundColor Yellow

$pathTraversalFiles = @(
    "..\..\..\etc\passwd.db",
    "..\..\windows\system32\file.db",
    "normal_file.db"
)

$traversalResults = @()
foreach ($fileName in $pathTraversalFiles) {
    $filePath = Join-Path $testDir $fileName
    Create-TestDatabaseFile -FilePath $filePath -Size 256
    
    $result = Upload-DatabaseFile -FilePath $filePath -ClientID $clientID -ProjectID $projectID
    $traversalResults += $result
    
    if ($result.Success) {
        $savedPath = $result.Response.file_path
        # Check that path traversal was prevented
        if ($savedPath -notmatch "\.\.") {
            Write-Host "  OK Uploaded and sanitized: $fileName -> $(Split-Path -Leaf $savedPath)" -ForegroundColor Green
        } else {
            Write-Host "  ERROR Path traversal not prevented: $savedPath" -ForegroundColor Red
        }
    } else {
        Write-Host "  OK Rejected (may be due to invalid characters): $fileName" -ForegroundColor Green
    }
}
Write-Host ""

# Test 7: Unicode characters in file names
Write-Host "[Edge Case 7] Unicode characters in file names..." -ForegroundColor Yellow

$unicodeFiles = @(
    "файл_русский.db",
    "文件_中文.db",
    "ファイル_日本語.db",
    "file_émojis_😀.db"
)

$unicodeResults = @()
foreach ($fileName in $unicodeFiles) {
    $filePath = Join-Path $testDir $fileName
    Create-TestDatabaseFile -FilePath $filePath -Size 256
    
    $result = Upload-DatabaseFile -FilePath $filePath -ClientID $clientID -ProjectID $projectID
    $unicodeResults += $result
    
    if ($result.Success) {
        Write-Host "  OK Uploaded: $fileName" -ForegroundColor Green
    } else {
        Write-Host "  ERROR Failed: $fileName - $($result.Error)" -ForegroundColor Red
    }
}

$unicodeSuccess = ($unicodeResults | Where-Object { $_.Success }).Count
Write-Host "  Summary: $unicodeSuccess/$($unicodeFiles.Count) Unicode files uploaded" -ForegroundColor $(if ($unicodeSuccess -eq $unicodeFiles.Count) { "Green" } else { "Yellow" })
Write-Host ""

# Final summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Edge Cases Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

$allResults = $specialResults + $longNameResults + $mixedTypeResults + $duplicateResults + $minimalResults + $traversalResults + $unicodeResults
$totalTests = $allResults.Count
$successfulTests = ($allResults | Where-Object { $_.Success }).Count

Write-Host "Total tests: $totalTests" -ForegroundColor White
Write-Host "Successful: $successfulTests" -ForegroundColor Green
Write-Host "Failed/Rejected (expected): $($totalTests - $successfulTests)" -ForegroundColor $(if (($totalTests - $successfulTests) -gt 0) { "Gray" } else { "Yellow" })

# Cleanup
Write-Host ""
Write-Host "Cleaning up test files..." -ForegroundColor Yellow
Remove-Item -Path $testDir -Recurse -Force -ErrorAction SilentlyContinue
Write-Host "  OK Test directory removed" -ForegroundColor Green

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Edge cases test completed" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

return @{
    TotalTests = $totalTests
    Successful = $successfulTests
    Failed = $totalTests - $successfulTests
    Results = $allResults
}

